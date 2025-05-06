package hertzServer

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// 简化的广告服务，使用原子计数器记录请求
var hertzCounter int64

// StartHertzServer 启动Hertz服务器
func StartHertzServer(port int) {
	// 创建Hertz服务器，优化配置以处理高并发场景
	h := server.New(
		server.WithHostPorts(fmt.Sprintf(":%d", port)),
		server.WithMaxRequestBodySize(1024*1024), // 1MB请求体限制
		server.WithReadTimeout(60*time.Second),
		server.WithWriteTimeout(60*time.Second),
		server.WithIdleTimeout(180*time.Second),
		server.WithExitWaitTime(time.Second*10), // 优雅退出等待时间
		server.WithNetwork("tcp"),
		server.WithKeepAlive(true),
		server.WithReadBufferSize(8*1024),
	)

	// 添加一个简单的中间件来恢复panic
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Hertz服务器发生panic: %v", err)
				c.String(consts.StatusInternalServerError, "服务器内部错误")
				c.Abort()
			}
		}()
		c.Next(ctx)
	})

	// 添加一个简单的中间件来记录请求，帮助诊断
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		// 处理前记录时间
		start := time.Now()

		// 继续处理请求
		c.Next(ctx)

		// 计算请求处理时间
		duration := time.Since(start)

		// 只记录超过1秒的慢请求，避免日志过多
		if duration > time.Second {
			log.Printf("慢请求[Hertz]: %s %s 耗时: %v\n",
				string(c.Method()), string(c.Path()), duration)
		}
	})

	// 广告重定向路由
	h.GET("/ad", func(ctx context.Context, c *app.RequestContext) {
		// 增加计数器
		atomic.AddInt64(&hertzCounter, 1)

		// 获取广告ID参数
		adID := c.Query("id")
		if adID == "" {
			c.String(consts.StatusBadRequest, "缺少广告ID参数")
			return
		}

		// 模拟查找广告目标URL
		targetURL := fmt.Sprintf("https://example.com/product/%s", adID)

		// 返回302重定向
		c.Redirect(consts.StatusFound, []byte(targetURL))
	})

	// 统计接口
	h.GET("/stats", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, utils.H{
			"framework": "Hertz",
			"requests":  atomic.LoadInt64(&hertzCounter),
		})
	})

	// 健康检查接口
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, utils.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	fmt.Printf("Hertz服务启动在端口 %d\n", port)
	log.Printf("Hertz服务器配置: 读超时: %v, 写超时: %v, 空闲超时: %v\n",
		60*time.Second, 60*time.Second, 180*time.Second)

	// 启动服务器
	h.Spin()
}
