package gping

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/router/config"
)

// type Gping struct {
// 	Url string // json rpc url
// 	Address string
// 	VaultAddress string
// }

type GpingClient struct {
	gpings []config.Gping
}

func NewGpingClient(cfg *config.Config) *GpingClient {
	return &GpingClient{gpings: cfg.GpingList}
}


func (c *GpingClient) BroadcastRequest(ip, requestID string) error {

    // Broadcast to all GPings concurrently
    for _, gping := range c.gpings {
        go func(g config.Gping) {
            rpcRequest := struct {
                JsonRPC string      `json:"jsonrpc"`
                Method  string      `json:"method"`
                Params  interface{} `json:"params"`
				ID int64 `json:"id"`
            }{
                JsonRPC: "2.0",
                Method:  "_location",
                Params:  struct {
                    IP        string `json:"ip"`
                    RequestID string `json:"request_id"`
                }{
                    IP:        ip,
                    RequestID: requestID,
                },
				ID: time.Now().UnixNano(),
            }

			rpcData, err := json.Marshal(rpcRequest)
            if err != nil {
                return // Error ignored as we only need one successful response
            }

			// Send JSON-RPC request
            resp, err := http.Post(
                g.Url,  // JSON-RPC endpoint from config
                "application/json",
                bytes.NewBuffer(rpcData),
            )
            if err != nil {
                return // Error ignored as we only need one successful response
            }
            defer resp.Body.Close()
            
        }(gping)
    }
    return nil
}