package websockets

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"sync"
	"time"
)

const (
	pongWait   = time.Second * 30
	writeWait  = 10 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type WsClientArray []*WsConnection

type WsClientsPool struct {
	clients *sync.Map
	mux     sync.Mutex
}

func NewPool() *WsClientsPool {
	return &WsClientsPool{clients: &sync.Map{}, mux: sync.Mutex{}}
}

func NewPoolWithClients(clientsId string, clients *WsClientArray) *WsClientsPool {
	pool := &WsClientsPool{clients: &sync.Map{}, mux: sync.Mutex{}}
	pool.clients.Store(clientsId, clients)
	return pool
}

func (p *WsClientsPool) Append(client *WsConnection, id string) {
	if client == nil {
		return
	}
	arr, ok := p.clients.Load(id)
	p.mux.Lock()
	if ok {
		ar := arr.(WsClientArray)
		ar = append(ar, client)
		p.clients.Store(id, ar)
	} else {
		p.clients.Store(id, WsClientArray{client})
	}
	client.pool = p
	p.mux.Unlock()
}

func (p *WsClientsPool)GetConnectionsByID(id string) *WsClientArray {
	arr, ok := p.clients.Load(id)
	if ok {
		a := arr.(WsClientArray)
		return &a
	}
	return nil
}

func (p *WsClientsPool) Send(clientId string, event interface{}) {
	c := p.GetConnectionsByID(clientId)
	if c != nil {
		p.sendClients(event, *c)
	}
}

func (p *WsClientsPool) sendClients(event interface{}, clients WsClientArray) {
	for _, cl := range clients {
		cl.events <- event
	}
}

func (p *WsClientsPool) Broadcast(event interface{}) {
	p.clients.Range(func(key, value interface{}) bool {
		ar := value.(WsClientArray)
		p.sendClients(event, ar)
		return true
	})
}

func (p *WsClientsPool) Delete(id string) {
	arr, ok := p.clients.Load(id)
	if ok {
		p.mux.Lock()
		ar := arr.(WsClientArray)
		newCl := WsClientArray{}
		for _, cl := range ar {
			if cl.id != id {
				newCl = append(newCl, cl)
			}
		}
		p.mux.Unlock()
		p.clients.Store(id, newCl)
	}
}
type MessageType int
const (
	MessageTypeText = websocket.TextMessage
	MessageTypeData = websocket.BinaryMessage
)
type Message struct {
	MessageType MessageType
	Data []byte
}

type WsConnection struct {
	conn   *websocket.Conn
	events chan interface{}
	MessagesChannel chan Message
	Done   chan bool
	pool   *WsClientsPool
	id     string
}

func NewWsConnection(conn *websocket.Conn, id string, messages chan Message, pool *WsClientsPool) *WsConnection {
	events := make(chan interface{})
	done := make(chan bool, 1)
	connectionId := uuid.New().String()
	connection := &WsConnection{conn: conn, events: events, MessagesChannel: messages, Done: done, pool: nil, id: connectionId}
	ticker := time.NewTicker(pingPeriod)
	finish := make(chan bool, 1)

	connection.conn.SetReadDeadline(time.Now().Add(pongWait))
	connection.conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	go func() {
		for {
			types, data, err := connection.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("error: %v", err)
				}
				connection.conn.Close()
				if connection.pool != nil {
					connection.pool.Delete(id)
				}
				finish <- true
				break
			}
			if types == websocket.BinaryMessage || types == websocket.TextMessage {
				messages <- Message{MessageType: MessageType(types), Data: data}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ticker.C:
				connection.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := connection.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					break
				}
			case event := <-connection.events:
				connection.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := connection.conn.WriteJSON(event); err != nil {
					break
				}
			case <-finish:
				go func() { connection.Done <- true }()
				break
			}
		}
	}()
	pool.Append(connection, id)
	return connection
}
