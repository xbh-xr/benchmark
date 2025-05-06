package benchmark

import (
	"bufio"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 测试结果结构
type BenchmarkResult struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalLatency       int64  // 微秒
	MinLatency         int64  // 微秒
	MaxLatency         int64  // 微秒
	TCPEstablished     int64  // ESTABLISHED 状态的连接数
	TCPTimeWait        int64  // TIME_WAIT 状态的连接数
	TCPCloseWait       int64  // CLOSE_WAIT 状态的连接数
	MemoryUsage        uint64 // 内存使用量（字节）
	MaxGoroutines      int64  // 最大 Goroutine 数量
}

// TCP连接状态统计
type TCPStats struct {
	Established int64
	TimeWait    int64
	CloseWait   int64
}

// RunBenchmark 执行基准测试
func RunBenchmark(url string, concurrency, durationSeconds int, delayMs int) {
	fmt.Printf("开始测试 URL: %s\n", url)
	fmt.Printf("并发连接: %d, 持续时间: %d秒, 请求延迟: %dms\n", concurrency, durationSeconds, delayMs)

	var result BenchmarkResult
	result.MinLatency = int64(^uint64(0) >> 1) // 初始化为最大int64值

	// 初始化最大 Goroutine 数量
	atomic.StoreInt64(&result.MaxGoroutines, int64(runtime.NumGoroutine()))

	// 创建停止通道和等待组
	stopCh := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(concurrency)

	// 创建HTTP客户端 - 优化配置
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			MaxIdleConns:          concurrency,
			MaxIdleConnsPerHost:   concurrency,
			DisableKeepAlives:     false,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: 30 * time.Second,
	}

	// 启动工作协程
	startTime := time.Now()
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()

			for {
				select {
				case <-stopCh:
					return
				default:
					// 发送请求并测量延迟
					reqStart := time.Now()
					resp, err := client.Get(url)
					latency := time.Since(reqStart).Microseconds()

					atomic.AddInt64(&result.TotalRequests, 1)

					if err != nil {
						atomic.AddInt64(&result.FailedRequests, 1)
					} else {
						resp.Body.Close() // 必须关闭body

						if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound {
							atomic.AddInt64(&result.SuccessfulRequests, 1)
							atomic.AddInt64(&result.TotalLatency, latency)

							// 更新最小延迟
							for {
								current := atomic.LoadInt64(&result.MinLatency)
								if latency >= current {
									break
								}
								if atomic.CompareAndSwapInt64(&result.MinLatency, current, latency) {
									break
								}
							}

							// 更新最大延迟
							for {
								current := atomic.LoadInt64(&result.MaxLatency)
								if latency <= current {
									break
								}
								if atomic.CompareAndSwapInt64(&result.MaxLatency, current, latency) {
									break
								}
							}
						} else {
							atomic.AddInt64(&result.FailedRequests, 1)
						}
					}

					// 添加请求延迟
					time.Sleep(time.Duration(delayMs) * time.Millisecond)
				}
			}
		}()
	}

	// 启动统计协程
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				// 更新内存使用统计
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				atomic.StoreUint64(&result.MemoryUsage, m.Alloc)

				// 获取TCP连接状态
				tcpStats := getTCPStats()
				atomic.StoreInt64(&result.TCPEstablished, tcpStats.Established)
				atomic.StoreInt64(&result.TCPTimeWait, tcpStats.TimeWait)
				atomic.StoreInt64(&result.TCPCloseWait, tcpStats.CloseWait)

				// 更新Goroutine最大数量
				currentGoroutines := int64(runtime.NumGoroutine())
				for {
					maxGoroutines := atomic.LoadInt64(&result.MaxGoroutines)
					if currentGoroutines <= maxGoroutines {
						break
					}
					if atomic.CompareAndSwapInt64(&result.MaxGoroutines, maxGoroutines, currentGoroutines) {
						break
					}
				}

				// 实时打印状态
				fmt.Printf("\r已处理: %d 请求, 成功率: %.2f%%, RPS: %.2f, TCP-ESTAB: %d, TCP-WAIT: %d, Goroutines: %d     ",
					atomic.LoadInt64(&result.TotalRequests),
					float64(atomic.LoadInt64(&result.SuccessfulRequests))/float64(atomic.LoadInt64(&result.TotalRequests))*100,
					float64(atomic.LoadInt64(&result.TotalRequests))/time.Since(startTime).Seconds(),
					tcpStats.Established,
					tcpStats.TimeWait,
					currentGoroutines)
			}
		}
	}()

	// 等待指定的测试时间
	time.Sleep(time.Duration(durationSeconds) * time.Second)

	// 在关闭通道前记录当前的goroutine数量
	currentGoroutines := int64(runtime.NumGoroutine())
	for {
		maxGoroutines := atomic.LoadInt64(&result.MaxGoroutines)
		if currentGoroutines <= maxGoroutines {
			break
		}
		if atomic.CompareAndSwapInt64(&result.MaxGoroutines, maxGoroutines, currentGoroutines) {
			break
		}
	}

	close(stopCh)

	// 等待所有工作协程完成
	wg.Wait()
	totalDuration := time.Since(startTime)
	fmt.Println() // 换行，为了美观

	// 打印测试结果
	printBenchmarkResults(&result, totalDuration)
}

// 获取TCP连接状态统计
func getTCPStats() TCPStats {
	var stats TCPStats

	if runtime.GOOS == "windows" {
		// Windows平台使用netstat命令
		cmd := exec.Command("netstat", "-an")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "ESTABLISHED") {
					stats.Established++
				} else if strings.Contains(line, "TIME_WAIT") {
					stats.TimeWait++
				} else if strings.Contains(line, "CLOSE_WAIT") {
					stats.CloseWait++
				}
			}
		}
	} else {
		// Linux/Unix平台使用ss命令
		cmd := exec.Command("ss", "-tan", "state", "established")
		output, err := cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			lineCount := 0
			for scanner.Scan() {
				lineCount++
			}
			stats.Established = int64(lineCount - 1) // 减去标题行
		}

		cmd = exec.Command("ss", "-tan", "state", "time-wait")
		output, err = cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			lineCount := 0
			for scanner.Scan() {
				lineCount++
			}
			stats.TimeWait = int64(lineCount - 1) // 减去标题行
		}

		cmd = exec.Command("ss", "-tan", "state", "close-wait")
		output, err = cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			lineCount := 0
			for scanner.Scan() {
				lineCount++
			}
			stats.CloseWait = int64(lineCount - 1) // 减去标题行
		}
	}

	return stats
}

// 打印基准测试结果
func printBenchmarkResults(result *BenchmarkResult, duration time.Duration) {
	fmt.Println("\n测试结果:")
	fmt.Printf("总请求数: %d\n", result.TotalRequests)
	fmt.Printf("成功请求: %d (%.2f%%)\n",
		result.SuccessfulRequests,
		float64(result.SuccessfulRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("失败请求: %d\n", result.FailedRequests)

	rps := float64(result.TotalRequests) / duration.Seconds()
	fmt.Printf("每秒请求数 (RPS): %.2f\n", rps)

	if result.SuccessfulRequests > 0 {
		avgLatency := float64(result.TotalLatency) / float64(result.SuccessfulRequests)
		fmt.Printf("平均延迟: %.2f 微秒\n", avgLatency)
		fmt.Printf("最小延迟: %d 微秒\n", result.MinLatency)
		fmt.Printf("最大延迟: %d 微秒\n", result.MaxLatency)
	}

	fmt.Printf("内存使用: %.2f MB\n", float64(result.MemoryUsage)/1024/1024)
	fmt.Printf("当前 Goroutine 数量: %d\n", runtime.NumGoroutine())
	fmt.Printf("最大 Goroutine 数量: %d\n", result.MaxGoroutines)

	fmt.Println("\nTCP连接状态:")
	fmt.Printf("ESTABLISHED: %d\n", result.TCPEstablished)
	fmt.Printf("TIME_WAIT: %d\n", result.TCPTimeWait)
	fmt.Printf("CLOSE_WAIT: %d\n", result.TCPCloseWait)
}
