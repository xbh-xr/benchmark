package main

import (
	"benchmark/benchmark"
	"flag"
	"fmt"
	"os"
)

func main() {
	// 命令行参数
	host := flag.String("host", "127.0.0.1", "服务器主机地址")
	framework := flag.String("framework", "both", "测试框架: fiber, hertz, 或 both")
	concurrency := flag.Int("c", 1000, "并发连接数")
	duration := flag.Int("d", 10, "测试持续时间(秒)")
	delay := flag.Int("delay", 100, "每个请求的延迟时间(毫秒)")
	fiberPort := flag.Int("fiber-port", 8080, "Fiber服务端口")
	hertzPort := flag.Int("hertz-port", 8081, "Hertz服务端口")
	flag.Parse()

	fmt.Println("性能测试客户端 - 远程模式")
	fmt.Printf("目标主机: %s\n", *host)

	// 运行基准测试
	if *framework == "fiber" || *framework == "both" {
		fmt.Printf("\n===== 测试 Fiber 框架 (主机: %s, 端口: %d) =====\n", *host, *fiberPort)
		benchmark.RunBenchmark(fmt.Sprintf("http://%s:%d/ad?id=ad1", *host, *fiberPort), *concurrency, *duration, *delay)
	}

	if *framework == "hertz" || *framework == "both" {
		fmt.Printf("\n===== 测试 Hertz 框架 (主机: %s, 端口: %d) =====\n", *host, *hertzPort)
		benchmark.RunBenchmark(fmt.Sprintf("http://%s:%d/ad?id=ad1", *host, *hertzPort), *concurrency, *duration, *delay)
	}

	fmt.Println("\n测试完成！")
	os.Exit(0)
}
