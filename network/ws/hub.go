package ws

import (
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSHub struct {
	upgrader        *websocket.Upgrader
	connectC        chan *WSClient
	disconnectC     chan *WSClient
	countLiveSocket int64
}

func NewWsHub() *WSHub {
	r := &WSHub{
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1 << 20,
			WriteBufferSize: 1 << 20,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		connectC:    make(chan *WSClient, 1000),
		disconnectC: make(chan *WSClient, 1000),
	}
	go r.loop()
	return r
}

func (p *WSHub) AddClient(c *gin.Context) (*WSClient, error) {
	if ws, err := p.upgrader.Upgrade(c.Writer, c.Request, nil); err != nil {
		return nil, err
	} else {
		client := newWsClient(ws)
		p.connectC <- client
		return client, nil
	}
}

func (p *WSHub) SendToClient(client *WSClient, message interface{}) error {
    return client.sendMsg(message)
}


func (p *WSHub) GetLiveSocketCount() int64 {
	return atomic.LoadInt64(&p.countLiveSocket)
}


func (p *WSHub) loop() {
	for {
		select {
		case client := <-p.connectC:
			go client.process(p.disconnectC)
			atomic.AddInt64(&p.countLiveSocket, 1)

		case <-p.disconnectC:
			atomic.AddInt64(&p.countLiveSocket, -1)
		}
	}
}
