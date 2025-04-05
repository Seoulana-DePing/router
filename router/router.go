package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/router/network/ws"
	"github.com/router/types"
)

func (r *Router) registerHandler() {
	//ws
	// r.RegisterGETHandler("/ws/chain", r.wsChain)
	// r.RegisterGETHandler("/ws/deploy", r.wsDeploy)
	// r.RegisterGETHandler("/ws/call", r.wsFunctionCall)
	r.RegisterGETHandler("/ws/ip-geo", r.IpGeoInfo)
	r.RegisterPOSTHandler("/gping/answer", r.HandleGPingResponse)

	//register websocket request handler
	if err := ws.AddHandler(ws.WsType(1), r.handleIpGeoInfoRequest); err != nil {
		r.log.Crit("Failed to add websocket ip geo info request handler", "error", err)
	} else if err := ws.AddHandler(ws.WsType(2), r.handleSignedTx); err != nil {
		r.log.Crit("Failed to add websocket ip geo info request handler", "error", err)
	} 

}

func (r *Router) IpGeoInfo(c *gin.Context) {
	if _, err := r.wsHub.AddClient(c); err != nil {
		r.RespError(c, http.StatusInternalServerError, err)
		return
	}
}

func (r *Router) handleIpGeoInfoRequest(req interface{}, client *ws.WSClient) (interface{}, error) {
	// Step 1: Handle initial IP request
    ipReq, ok := req.(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("invalid request format")
    }

    ip, ok := ipReq["ip"].(string)
    if !ok {
        return nil, fmt.Errorf("invalid ip format")
    }
	fmt.Println("------------------------STEP1 DONE------------------------")
	 // Step 2: Send initial response
	 initialResponse := &types.WsResponse{
        Type:    "Initiate",
        Payload: "Pings started looking for your ip geo info. ip : " + ip,
    }
	
	if err := r.wsHub.SendToClient(client, initialResponse); err != nil {
        return nil, fmt.Errorf("failed to send initial response: %v", err)
    }
	fmt.Println("------------------------STEP2 DONE------------------------")

	// Step 3: Braodcast all ip to gpings
	requestID := uuid.New().String()
    resultChan := make(chan *types.ResponseFromGping, 1)
    timeoutChan := make(chan bool, 1)
	fmt.Println("------------------------REQUEST ID------------------------" + requestID)
	
	// Store the request channels
    geoReq := &types.RequestToGping{
        RequestID: requestID,
        IP: ip,
        ResultChan: resultChan,
        TimeoutChan: timeoutChan,
    }
    r.pendingGeoRequests.Store(requestID, geoReq)
	defer func() {
        r.pendingGeoRequests.Delete(requestID)
        close(resultChan)
        close(timeoutChan)
    }()

	 //Broadcast to GPings
	 r.gpingClient.BroadcastRequest(ip, requestID)

	var result *types.ResponseFromGping
	select {
    case result = <-resultChan:
		
		locationInfo, err := r.getLocationInfo(result.Latitude, result.Longitude)
		if err != nil {
			return nil, fmt.Errorf("failed to get location info: %v", err)
		}
		val := &types.PendingRequestIdsValue{
			DisplayName: locationInfo.DisplayName,
			VaultAddress: result.Vault,
		}
		r.pendingRequestIds.Store(requestID, val)
    case <-timeoutChan:
        // Timeout occurred
        if err := r.wsHub.SendToClient(client, &types.WsResponse{
            Type: "error",
            Payload: "Request timed out waiting for GPing response",
        }); err != nil {
            return nil, fmt.Errorf("failed to send timeout error: %v", err)
        }
    case <-time.After(30 * time.Second):
        // Backup timeout
        timeoutChan <- true
        return nil, fmt.Errorf("request timed out")
    }

	fmt.Println("------------------------STEP3 DONE------------------------")
	
	// Step 4. Make raw transaction to send solana network
	unsignedTx := &types.WsResponseWithRequestID{
        Type: "unsignedTx",
		RequestID: requestID,
        Payload: map[string]interface{}{
            "program": "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA", // Solana Token Program ID
            "instruction": "Approve",
            "data": map[string]interface{}{
                "amount": "1000000000",             // 1 JitoSOL (9 decimals)
            },
            "accounts": map[string]string{
                "source": "Client's JitoSOL token account",      // Client will fill their JitoSOL token account
                "delegate": r.keyPair.PublicKey().String(),        // Router's public key that will be approved
                "owner": "Client's wallet address",       // Client will fill their wallet address
            },
        },
    }

	if err := r.wsHub.SendToClient(client, unsignedTx); err != nil {
        return nil, fmt.Errorf("failed to send unsigned transaction: %v", err)
    }
	fmt.Println("------------------------STEP4 DONE------------------------")
	return nil, nil
}

func (r *Router) handleSignedTx(req interface{}, client *ws.WSClient) (interface{}, error) {
	// Step 1: execute approval transaction
	signedTx, ok := req.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}
	approvalTx, ok := signedTx["signed_tx"].(string)
    if !ok {
        return nil, fmt.Errorf("invalid approval tx format")
    }

	requestID := signedTx["request_id"].(string)

	val, ok := r.pendingRequestIds.Load(requestID)
	if !ok {
		return nil, fmt.Errorf("request id not found")
	}
	geoResult, ok := val.(*types.PendingRequestIdsValue);	
	if !ok {
		return nil, fmt.Errorf("invalid requestIds value")
	}

	approvalTxHash, err := r.solanaClient.SendRawTransaction(approvalTx)
	if err != nil {
		r.wsHub.SendToClient(client, &types.WsResponse{
            Type: "error",
            Payload: "Failed to submit transaction",
        })
        return nil, fmt.Errorf("failed to submit transaction: %v", err)
	}

	_, err = r.solanaClient.WaitForTransactionConfirmation(approvalTxHash)
    if err != nil {
        return nil, fmt.Errorf("failed to confirm approval: %v", err)
    }


	if err := r.wsHub.SendToClient(client, &types.WsResponse{
        Type: "success",
        Payload: map[string]interface{}{
            "message": "Approval Transaction submitted successfully",
            "txHash": approvalTxHash,
        },
    }); err != nil {
        return nil, fmt.Errorf("failed to send tx hash: %v", approvalTx)
    }

	// Step2 : send transaction that exectutes JitoSOL contract's method "transferFrom".
	gPingAta, _, err := solana.FindAssociatedTokenAddress(solana.MustPublicKeyFromBase58(geoResult.VaultAddress),solana.MustPublicKeyFromBase58("9JUomKyopNpak1kZvBA6taUfV9rJxctLeFB8ac2iFDaH") )
	if err != nil {
		return nil, fmt.Errorf("failed to get associated token address: %v", err)
	}
	transferInstruction := token.NewTransferCheckedInstruction(
        1_000_000_000,                                             // amount: 1 JitoSOL
        9,                                                         // decimals
        solana.MustPublicKeyFromBase58("AHDHUrKFvYmrAm2cLScAD4xp9UFjw5JcEdAo19ofvUjZ"),    // source
        solana.MustPublicKeyFromBase58("9JUomKyopNpak1kZvBA6taUfV9rJxctLeFB8ac2iFDaH"),     // mint
		gPingAta,   // destination ==  gping 주소의 토큰계정
        r.keyPair.PublicKey(),                              // authority (Router)
        []solana.PublicKey{},                                     // signers
    ).Build()

	// Create the transaction
    recent, err := r.solanaClient.GetRecentBlockhash(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to get recent blockhash: %v", err)
    }

    tx, err := solana.NewTransaction(
        []solana.Instruction{transferInstruction},
        recent.Value.Blockhash,
        solana.TransactionPayer(r.keyPair.PublicKey()),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create transaction: %v", err)
    }

    // Sign the transaction
    _, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
        if key.Equals(r.keyPair.PublicKey()) {
            return r.keyPair
        }
        return nil
    })
    if err != nil {
        return nil, fmt.Errorf("failed to sign transaction: %v", err)
    }

    // Send the transfer transaction
    transferTxHash, err := r.solanaClient.SendTransaction(context.Background(), tx)
    if err != nil {
        r.wsHub.SendToClient(client, &types.WsResponse{
            Type: "error",
            Payload: "Failed to execute transfer",
        })
        return nil, fmt.Errorf("failed to submit transfer: %v", err)
    }

    // Wait for transfer confirmation
    _, err = r.solanaClient.WaitForTransactionConfirmation(transferTxHash)
    if err != nil {
        r.wsHub.SendToClient(client, &types.WsResponse{
            Type: "error",
            Payload: "Failed to confirm transfer",
        })
        return nil, fmt.Errorf("failed to confirm transfer: %v", err)
    }

    // Send success response
    if err := r.wsHub.SendToClient(client, &types.WsResponse{
        Type: "result",
        Payload: map[string]interface{}{
			"geoResult": geoResult.DisplayName,
        },
    }); err != nil {
        return nil, fmt.Errorf("failed to send success message: %v", err)
    }

    return nil, nil

}

func (r *Router) HandleGPingResponse(c *gin.Context) {
   var response types.ResponseFromGping
    if err := c.BindJSON(&response); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
        return
    }

    // Look up the pending request
    if reqInterface, ok := r.pendingGeoRequests.Load(response.RequestID); ok {
        req := reqInterface.(*types.RequestToGping)
        // Send the response to the waiting channel
        req.ResultChan <- &types.ResponseFromGping{
			Latitude: response.Latitude,
			Longitude: response.Longitude,
			Vault: response.Vault,
			RequestID: response.RequestID,
		}
        c.JSON(http.StatusOK, gin.H{"status": "success"})
    } else {
        c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
    }
}

func (r *Router) getLocationInfo(lat, lon string) (*types.NominatimResponse, error) {
    url := fmt.Sprintf(
        "https://nominatim.openstreetmap.org/reverse?lat=%s&lon=%s&format=json",
        lat,
        lon,
    )

    // Create request with User-Agent header (required by Nominatim)
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %v", err)
    }
    req.Header.Set("User-Agent", "YourApp/1.0") // Nominatim requires a User-Agent

    // Send request
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to send request: %v", err)
    }
    defer resp.Body.Close()

    // Parse response
    var nominatim types.NominatimResponse
    if err := json.NewDecoder(resp.Body).Decode(&nominatim); err != nil {
        return nil, fmt.Errorf("failed to decode response: %v", err)
    }

    return &nominatim, nil
}