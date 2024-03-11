# websocket_packet

### 介绍
 - 针对github.com/gorilla/websocket不能高并发写入
 - 针对golang.org/x/net/websocket不能高并发的写入
 - 总的来说,读写效率提高了不少,但由于golang的Runtime的机制,内存的占用比 c/c++,及rust写的包高一些
### 软件架构

~~~text
|
|- frame                              # 帧
|   |- codec.go                       # 帧解码器
|   |- frame.go                       # 帧结构
|   |- uity.go                        # 帧工具
|
|- session                            # session
|   |- session_config.go              # session配置
|   |- session_id.go                  # sessionId生成器
|   |- session_status.go              # session状态
|   |- websocket_session.go           # session接口
|
|- client.go                          # 客户端
|- example_test.go                    # 样例与测试 
|- server_handle.go                   # 服务端 
|- README.md                          # readme文件
~~~

### 安装教程
~~~shell
go get github.com/qdmc/websocket_packet
~~~

### 使用说明
 - [pkg.go.dev](https://pkg.go.dev/github.com/qdmc/websocket_packet)


