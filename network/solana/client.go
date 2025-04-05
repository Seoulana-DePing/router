package solana

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type SolanaClient struct {
    client *rpc.Client
}

func NewSolanaClient(endpoint string) *SolanaClient {
    rpcClient := rpc.New(endpoint) // e.g., "https://api.mainnet-beta.solana.com"
    return &SolanaClient{
        client: rpcClient,
	}
}

// SendRawTransaction sends a base64 encoded signed transaction
func (s *SolanaClient) SendRawTransaction(signedTxBase64 string) (string, error) {
    // Decode base64 transaction
    txBytes, err := base64.StdEncoding.DecodeString(signedTxBase64)
    if err != nil {
        return "", fmt.Errorf("failed to decode transaction: %v", err)
    }

    // Create transaction from bytes
    tx, err := solana.TransactionFromBytes(txBytes)
    if err != nil {
        return "", fmt.Errorf("failed to parse transaction: %v", err)
    }

    // Send transaction
    sig, err := s.client.SendTransactionWithOpts(context.Background(), tx,
        rpc.TransactionOpts{
            SkipPreflight:       false,
            PreflightCommitment: rpc.CommitmentConfirmed,
        },
    )
    if err != nil {
        return "", fmt.Errorf("failed to send transaction: %v", err)
    }

    return sig.String(), nil
}

// WaitForTransactionConfirmation waits for a transaction to be confirmed
func (s *SolanaClient) WaitForTransactionConfirmation(txHash string) (*rpc.GetSignatureStatusesResult, error) {
    sig := solana.MustSignatureFromBase58(txHash)

    status, err := s.client.GetSignatureStatuses(
        context.Background(),
        true,  // searchTransactionHistory
        sig,   // can pass multiple signatures
    )
    if err != nil {
        return nil, fmt.Errorf("failed to get transaction status: %v", err)
    }

    if status == nil || status.Value[0] == nil {
        return nil, fmt.Errorf("transaction not found")
    }

    return status, nil
}

// GetBalance gets the SOL balance of an account
func (s *SolanaClient) GetBalance(address string) (uint64, error) {
    pubKey := solana.MustPublicKeyFromBase58(address)
    
    balance, err := s.client.GetBalance(
        context.Background(),
        pubKey,
        rpc.CommitmentConfirmed, // specify the commitment level
    )
    if err != nil {
        return 0, fmt.Errorf("failed to get balance: %v", err)
    }

    return balance.Value, nil
}
// GetRecentBlockhash gets the most recent blockhash
func (s *SolanaClient) GetRecentBlockhash(ctx context.Context) (*rpc.GetRecentBlockhashResult, error) {
    return s.client.GetRecentBlockhash(ctx, rpc.CommitmentConfirmed)
}

// SendTransaction sends a transaction to the network
func (s *SolanaClient) SendTransaction(ctx context.Context, tx *solana.Transaction) (string, error) {
    sig, err := s.client.SendTransaction(ctx, tx)
    if err != nil {
        return "", fmt.Errorf("failed to send transaction: %v", err)
    }
    return sig.String(), nil
}