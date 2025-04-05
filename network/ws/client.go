package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/router/common/log"

	"github.com/gorilla/websocket"
)

type WSClient struct {
	ws             *websocket.Conn
	latestSendTime time.Time
	log            log.Logger
	context        sync.Map
}

func newWsClient(ws *websocket.Conn) *WSClient {
	ws.SetReadLimit(512)
	return &WSClient{
		ws:             ws,
		latestSendTime: time.Now(),
		log:            log.New("module", "websocket"),
	}
}

func (p *WSClient) sendMsg(v interface{}) error {
	var (
		bytes []byte
		err   error
	)

	if bytes, err = json.Marshal(v); err == nil {
		p.ws.SetWriteDeadline(time.Now().Add(10e9))
		err = p.ws.WriteMessage(websocket.TextMessage, bytes)
	}
	if err != nil {
		return err
	}
	p.latestSendTime = time.Now()
	return nil
}
func (c *WSClient) SetContext(key string, value interface{}) {
    c.context.Store(key, value)
}

// GetContext retrieves a value from the client's context
func (c *WSClient) GetContext(key string) interface{} {
    value, _ := c.context.Load(key)
    return value
}

// ClearContext removes a value from the client's context
func (c *WSClient) ClearContext(key string) {
    c.context.Delete(key)
}

func (p *WSClient) process(disconnectC chan<- *WSClient) {
	reqC := make(chan *WsReq)
	stop := make(chan struct{})

	defer func() {
		p.ws.Close()
		close(stop)
		disconnectC <- p
	}()

	go func() {
		defer p.ws.Close() //send에서 먼저 disconnect되었을때 close를 해야 recv에서 close를 처리를 할수있다.
		recvCount := 0

		pingTicker := time.NewTicker(1e9)
		defer pingTicker.Stop()

		for {
			select {
			case req := <-reqC:
				recvCount++
				if f, ok := wsHandlers[req.Type]; !ok {
					p.log.Error("handler not existd", "type", req.Type)
					return
				} else if res, err := f(req.Data, p); err != nil {
					p.log.Error("Failed handler ws request", "type", req.Type, "error", err)
					return
				} else {
					if err := p.sendMsg(&WsResp{Data: res}); err != nil {
						p.log.Error("Failed to write msg", "type", req.Type, "error", err)
						return
					}
					// 채널인 경우 특별 처리
					// if ch, ok := res.(chan types.TerminalOutput); ok {
					// 	// 채널에서 데이터를 읽어서 전송
					// 	for output := range ch {
					// 		resp := &types.TerminalOutput{
					// 			Content: output.Content,
					// 			Done:    output.Done,
					// 		}
					// 		if err := p.sendMsg(resp); err != nil {
					// 			p.log.Error("Failed to write msg", "type", req.Type, "error", err)
					// 			return
					// 		}
					// 		if output.Done {
					// 			break
					// 		}
					// 	}
					// } else {
					// 	// 일반적인 응답 처리
					// 	if err := p.sendMsg(&WsResp{Data: res}); err != nil {
					// 		p.log.Error("Failed to write msg", "type", req.Type, "error", err)
					// 		return
					// 	}
					// }
				}
			case <-pingTicker.C:
				if time.Since(p.latestSendTime) > 60e9 { //마지막 요청 이후 5초동안 아무런 요청이 없다면 연결을 끊는다.
					return
				}
			case <-stop:
				return
			}
		}
	}()

	for {
		req := (*WsReq)(nil)
		if _, message, err := p.ws.ReadMessage(); err == nil {
			if err := json.Unmarshal(message, &req); err == nil {
				reqC <- req
				continue
			}
		}
		break
	}
}
