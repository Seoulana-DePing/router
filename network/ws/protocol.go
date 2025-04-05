package ws

type WsType int

type WsReq struct {
	Type WsType      `json:"type"`
	Data interface{} `json:"data"`
}

type WsResp struct {
	Data interface{} `json:"data"`
}
