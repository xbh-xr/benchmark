# 高性能框架测试工具

这是一个用于测试 Fiber 和 Hertz 等高性能 Go Web 框架的工具。该工具支持在分布式环境中进行测试，通过将客户端和服务端分离，可以更准确地测量框架的性能表现。

## 功能特点

- 支持测试 Fiber 和 Hertz 两种框架
- 客户端和服务端可分离部署
- 支持自定义并发连接数和测试持续时间
- 可设置请求延迟模拟真实环境
- 实时监控 TCP 连接状态
- 详细的性能测试报告

## 使用方法

### 服务端

服务端负责启动 Web 框架服务，接收和处理客户端的请求。

```bash
# 启动服务端（默认同时启动 Fiber 和 Hertz）
go run main.go

# 仅启动 Fiber 框架
go run main.go -framework fiber -fiber-port 8080

# 仅启动 Hertz 框架
go run main.go -framework hertz -hertz-port 8081

# 自定义端口
go run main.go -fiber-port 9000 -hertz-port 9001
```

### 客户端

客户端负责向服务端发送测试请求，并收集、分析性能数据。

```bash
# 默认测试本地服务器
go run client/main.go

# 测试远程服务器
go run client/main.go -host 192.168.1.100

# 自定义测试参数
go run client/main.go -host 192.168.1.100 -c 2000 -d 30 -delay 100 -fiber-port 9000 -hertz-port 9001

# 仅测试 Fiber 框架
go run client/main.go -framework fiber -host 192.168.1.100
```

## 参数说明

### 服务端参数

- `-framework`: 选择启动哪个框架服务，可选值：fiber、hertz、both（默认）
- `-fiber-port`: Fiber 服务监听端口（默认 8080）
- `-hertz-port`: Hertz 服务监听端口（默认 8081）

### 客户端参数

- `-host`: 服务器主机地址（默认 127.0.0.1）
- `-framework`: 选择测试哪个框架，可选值：fiber、hertz、both（默认）
- `-c`: 并发连接数（默认 1000）
- `-d`: 测试持续时间，单位秒（默认 10）
- `-delay`: 每个请求的延迟时间，单位毫秒（默认 100）
- `-fiber-port`: Fiber 服务端口（默认 8080）
- `-hertz-port`: Hertz 服务端口（默认 8081）

## 测试报告

测试完成后，客户端会生成详细的性能测试报告，包括：

- 总请求数和成功率
- 每秒请求数（RPS）
- 请求延迟统计（平均、最小、最大）
- 内存使用情况
- Goroutine 数量
- TCP 连接状态统计（ESTABLISHED、TIME_WAIT、CLOSE_WAIT）

## 系统要求

- Go 1.16 或更高版本
- Windows、Linux 或 macOS 操作系统