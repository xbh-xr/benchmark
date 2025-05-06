package fiberServer

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
)

// 简化的广告服务，使用原子计数器记录请求
var fiberCounter int64

// StartFiberServer 启动Fiber服务器
func StartFiberServer(port int) {
	app := fiber.New(fiber.Config{
		Prefork:      false,
		ServerHeader: "Fiber",
		BodyLimit:    1 * 1024 * 1024,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
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

	// 启动服务器
	fmt.Printf("Fiber服务启动在端口 %d\n", port)
	if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatalf("Fiber服务器启动失败: %v", err)
	}
}
