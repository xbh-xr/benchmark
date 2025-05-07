package fiberServer

import (
	"fmt"
	"log"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/valyala/fasthttp"
)

// 简化的广告服务，使用原子计数器记录请求
var fiberCounter int64

// 定期触发GC
func startGCController() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			runtime.GC()
			debug := new(runtime.MemStats)
			runtime.ReadMemStats(debug)
			log.Printf("Fiber内存使用: 已分配: %.2f MB, 系统: %.2f MB, GC运行次数: %d\n",
				float64(debug.Alloc)/1024/1024,
				float64(debug.Sys)/1024/1024,
				debug.NumGC)
		}
	}()
}

// StartFiberServer 启动Fiber服务器
func StartFiberServer(port int) {
	// 启动GC控制器
	startGCController()

	// 优化Fiber配置以处理高并发场景
	app := fiber.New(fiber.Config{
		Prefork:          false,
		ServerHeader:     "Fiber",
		BodyLimit:        1 * 1024 * 1024, // 1MB
		ReadTimeout:      60 * time.Second,
		WriteTimeout:     60 * time.Second,
		IdleTimeout:      180 * time.Second,
		ReadBufferSize:   4096,       // 减小读缓冲区以节省内存
		WriteBufferSize:  4096,       // 减小写缓冲区以节省内存
		Concurrency:      256 * 1024, // 最大并发连接数
		DisableKeepalive: false,      // 启用keepalive
		// 配置底层FastHTTP客户端
		DisablePreParseMultipartForm: true,
		StreamRequestBody:            true,
		ReduceMemoryUsage:            true,
		Network:                      "tcp",
	})

	// 添加恢复中间件，防止panic导致服务器崩溃
	app.Use(recover.New())

	// 添加一个简单的中间件来记录请求，帮助诊断
	app.Use(func(c *fiber.Ctx) error {
		// 处理前记录时间
		start := time.Now()

		// 继续处理请求
		err := c.Next()

		// 计算请求处理时间
		duration := time.Since(start)

		// 只记录超过1秒的慢请求，避免日志过多
		if duration > time.Second {
			log.Printf("慢请求: %s %s 耗时: %v\n", c.Method(), c.Path(), duration)
		}

		return err
	})

	// 广告重定向路由
	app.Get("/ad", func(c *fiber.Ctx) error {
		// 增加计数器
		atomic.AddInt64(&fiberCounter, 1)

		// 获取广告ID参数
		adID := c.Query("id")
		if adID == "" {
			return c.Status(fiber.StatusBadRequest).SendString("缺少广告ID参数")
		}

		// 模拟查找广告目标URL
		targetURL := fmt.Sprintf("https://example.com/product/%s", adID)

		// 返回302重定向
		return c.Redirect(targetURL, fiber.StatusFound)
	})

	// 统计接口
	app.Get("/stats", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"framework": "Fiber",
			"requests":  atomic.LoadInt64(&fiberCounter),
		})
	})

	// 健康检查接口
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 配置服务器
	server := &fasthttp.Server{
		Handler:                       app.Handler(),
		Name:                          "Fiber",
		ReadTimeout:                   60 * time.Second,
		WriteTimeout:                  60 * time.Second,
		IdleTimeout:                   180 * time.Second,
		MaxRequestBodySize:            1 * 1024 * 1024, // 1MB
		DisableHeaderNamesNormalizing: false,
		DisableKeepalive:              false,
		MaxConnsPerIP:                 0, // 不限制单IP连接数
		TCPKeepalive:                  true,
		TCPKeepalivePeriod:            30 * time.Second,
		Concurrency:                   256 * 1024,
		ReadBufferSize:                4096,
		WriteBufferSize:               4096,
		GetOnly:                       false,
		ReduceMemoryUsage:             true,
		CloseOnShutdown:               true,
	}

	// 启动服务器
	address := fmt.Sprintf(":%d", port)
	fmt.Printf("Fiber服务启动在端口 %d\n", port)
	log.Printf("Fiber服务器配置: 最大并发: %d, 读超时: %v, 写超时: %v\n",
		server.Concurrency, server.ReadTimeout, server.WriteTimeout)

	// 发起手动GC
	runtime.GC()

	err := app.Listen(address)
	if err != nil {
		log.Fatalf("Fiber服务器启动失败: %v", err)
	}

	/*if err := server.ListenAndServe(address); err != nil {
		log.Fatalf("Fiber服务器启动失败: %v", err)
	}*/
}
