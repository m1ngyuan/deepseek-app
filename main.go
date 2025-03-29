package main

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

var wsUpgrader = websocket.Upgrader{
	HandshakeTimeout: time.Second * 10,
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WebSocketHandler(c *gin.Context) {

	// 获取WebSocket连接
	ws, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		panic(err)
	}

	defer ws.Close()

	// 处理WebSocket消息
	for {
		messageType, p, err := ws.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		switch messageType {
		case websocket.TextMessage:
			log.Printf("处理文本消息, %s\n", string(p))
			ws.WriteMessage(websocket.TextMessage, p)
			// c.Writer.Write(p)
		case websocket.BinaryMessage:
			log.Println("处理二进制消息")
		case websocket.CloseMessage:
			log.Println("关闭websocket连接")
			return
		case websocket.PingMessage:
			log.Println("处理ping消息")
			ws.WriteMessage(websocket.PongMessage, []byte("ping"))
		case websocket.PongMessage:
			log.Println("处理pong消息")
			ws.WriteMessage(websocket.PongMessage, []byte("pong"))
		default:
			log.Printf("未知消息类型: %d\n", messageType)
			return
		}
	}

}

func NewServer() *gin.Engine {
	gin.SetMode(gin.DebugMode) // 设置运行模式
	gin.DisableConsoleColor()  // 禁用控制台输出的颜色
	router := gin.Default()
	return router
}

func main() {
	// 创建Gin应用
	app := NewServer()

	// 注册WebSocket路由
	app.GET("/ws", WebSocketHandler)

	// 启动应用
	err := app.Run(":8080")
	if err != nil {
		panic(err)
	}
}
