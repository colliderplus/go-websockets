package websockets

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
	"log"
)

type WSContextConst string

const (
	ContextConstWSPool       WSContextConst = "ContextConstWSPool"
	ContextConstConnectionID WSContextConst = "ContextConstConnectionID"
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
		messagesChannel := make(chan Message)
		connectionId := GetConnectionIDFromContext(c)
		connection := NewWsConnection(conn, connectionId, messagesChannel, pool)
		c.Next()
		for {
			select {
			case message := <-messagesChannel:
				log.Println(message)
			case <-connection.Done:
				break
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

func SocketsEventMiddleware(pool *WsClientsPool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(string(ContextConstWSPool), pool)
	}
}

//Ws
func GetWSPoolContext(c *gin.Context) *WsClientsPool {
	return c.MustGet(string(ContextConstWSPool)).(*WsClientsPool)
}
