package ws

import (
	"encoding/json"
	"fmt"
)

type tyHandler func(interface{}, *WSClient) (interface{}, error)

var wsHandlers = make(map[WsType]tyHandler)

func AddHandler(handlerType WsType, handler tyHandler) error {
	if _, ok := wsHandlers[handlerType]; ok {
		return fmt.Errorf("ws handler type %d already existed ", handlerType)
	}

	wsHandlers[handlerType] = handler

	return nil
}

func VerifyRequest(v, r interface{}) error {
	if bytes, err := json.Marshal(v); err != nil {
		return err
	} else if err := json.Unmarshal(bytes, r); err != nil {
		return err
	}

	return nil
}
