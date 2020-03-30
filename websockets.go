package middleware

import (
	"collider/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
	"github.com/gorilla/websocket"
)

type WSContextConst string
const (
	ContextConstWSPool WSContextConst = "ContextConstWSPool"
	ContextConstConnectionID WSContextConst = "ContextConstConnectionID"
)

var wsupgrader = websocket.Upgrader{
	HandshakeTimeout: time.Second * 30,
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	EnableCompression:true,
}

func ConnectionUpgrade()  gin.HandlerFunc {
	return func (c *gin.Context) {
		conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil || conn == nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, utils.DefaultResponseError(nil))
			return
		}
		pool := GetWSPoolContext(c)
		connectionId := GetConnectionIDFromContext(c)
		connection := NewWsConnection(conn, connectionId)
		pool.Append(connection, connectionId)
		c.Next()
		<-connection.Done
	}
}

func GetConnectionIDFromContext(c *gin.Context) string {
	return c.MustGet(string(ContextConstConnectionID)).(string)
}
func SetConnectionIDFromContext(id string,c *gin.Context) {
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