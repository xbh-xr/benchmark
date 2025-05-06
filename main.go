package main

import (
	fiberServer "benchmark/fiber"
	hertzServer "benchmark/hertz"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	// 命令行参数
	framework := flag.String("framework", "both", "服务框架: fiber, hertz, 或 both")
	fiberPort := flag.Int("fiber-port", 8080, "Fiber服务端口")
	hertzPort := flag.Int("hertz-port", 8081, "Hertz服务端口")
	flag.Parse()

	fmt.Println("性能测试服务端")
	fmt.Println("按 Ctrl+C 停止服务...")

	// 根据选择启动对应的服务器
	var wg sync.WaitGroup

	if *framework == "fiber" || *framework == "both" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fiberServer.StartFiberServer(*fiberPort)
		}()
		fmt.Printf("Fiber 服务已启动，监听端口: %d\n", *fiberPort)
	}

	if *framework == "hertz" || *framework == "both" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hertzServer.StartHertzServer(*hertzPort)
		}()
		fmt.Printf("Hertz 服务已启动，监听端口: %d\n", *hertzPort)
	}

	// 创建信号通道捕获退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 等待退出信号
	<-sigCh
	fmt.Println("\n收到退出信号，正在关闭服务器...")

	// 此处可以添加优雅关闭逻辑

	// 等待服务器关闭
	wg.Wait()
	fmt.Println("服务器已关闭")
}
