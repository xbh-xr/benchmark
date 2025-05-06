package hertzServer

import (
	"context"
	"fmt"
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
	// 创建Hertz服务器
	h := server.New(
		server.WithHostPorts(fmt.Sprintf(":%d", port)),
		server.WithMaxRequestBodySize(1024*1024), // 1MB请求体限制
		server.WithReadTimeout(30*time.Second),
		server.WithWriteTimeout(30*time.Second),
		server.WithIdleTimeout(120*time.Second),
	)

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

	fmt.Printf("Hertz服务启动在端口 %d\n", port)
	h.Spin()
}
