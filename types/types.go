package types


type ParamInfo struct {
    Name  string `json:"Name"`
    Type  string `json:"Type"`
    Value string `json:"Value"`
}

type IpGeoInfoRequest struct {
    Ip string `json:"ip"`
}

type SendTxRequest struct {
	Tx string `json:"tx"`
}

type WsResponse struct {
    Type    string    `json:"type"`
    Payload interface{} `json:"payload"`
}

type WsResponseWithRequestID struct {
    Type    string    `json:"type"`
    Payload interface{} `json:"payload"`
	RequestID string `json:"request_id"`
}

type RequestToGping struct {
    RequestID string
    IP        string
    ResultChan chan *ResponseFromGping
    TimeoutChan chan bool
}

type ResponseFromGping struct {
    Latitude string `json:"latitude"`
    Longitude string   `json:"longitude"`      
	Vault    string `json:"vault"` // Vault contract address`
	RequestID string `json:"request_id"`
}

type NominatimResponse struct {
    DisplayName string `json:"display_name"`
    Address struct {
        Country     string `json:"country"`
        CountryCode string `json:"country_code"`
        City        string `json:"city"`
    } `json:"address"`
}

type PendingRequestIdsValue struct {
    DisplayName string `json:"display_name"`
    VaultAddress string `json:"vault_address"`
}
// type RawTxResponse

// type IpGeoInfoResponse

// type SendTxResponse


