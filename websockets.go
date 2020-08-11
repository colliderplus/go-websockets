package websockets

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

type WSContextConst string

const (
	ContextConstWSPool       WSContextConst = "ContextConstWSPool"
	ContextConstConnectionID WSContextConst = "ContextConstConnectionID"
	ContextIncomingMessage WSContextConst = "ContextIncomingMessage"
)

var wsupgrader = websocket.Upgrader{
	HandshakeTimeout: time.Second * 30,
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	EnableCompression: true,
}

func ConnectionUpgrade() gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil || conn == nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, nil)
			return
		}
		pool := GetWSPoolContext(c)
		if  pool  !=  nil {
			messagesChannel := make(chan Message)
			connectionId := GetConnectionIDFromContext(c)
			connection := NewWsConnection(conn, connectionId, &messagesChannel, pool)
			c.Next()
			for {
				select {
				case message := <-messagesChannel:
					setIncomingMessageFromContext(message,c)
				case <-connection.Done:
					break
				}
			}
		}
	}
}

func GetConnectionIDFromContext(c *gin.Context) string {
	return c.MustGet(string(ContextConstConnectionID)).(string)
}

func SetConnectionIDFromContext(id string, c *gin.Context) {
	c.Set(string(ContextConstConnectionID), id)
}

func GetIncomingMessageFromContext(c *gin.Context) Message {
	return c.MustGet(string(ContextIncomingMessage)).(Message)
}

func setIncomingMessageFromContext(message Message, c *gin.Context) {
	c.Set(string(ContextIncomingMessage), message)
}

func SocketsEventMiddleware(pool *WsClientsPool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(string(ContextConstWSPool), pool)
	}
}

//Ws
func GetWSPoolContext(c *gin.Context) *WsClientsPool {
	return c.MustGet(string(ContextConstWSPool)).(*WsClientsPool)
}
