package keystore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gagliardetto/solana-go"
	"github.com/xdg-go/pbkdf2"
)

type EncryptedKeystore struct {
    Address    string `json:"address"`
    KeyData    string `json:"keydata"`
    Version    int    `json:"version"`
}


const (
    keySize    = 32 // AES-256
    saltSize   = 32
    iterations = 10000
)

// deriveKey derives an encryption key from a password using PBKDF2
func deriveKey(password string, salt []byte) []byte {
    return pbkdf2.Key([]byte(password), salt, iterations, keySize, sha256.New)
}

// encryptPrivateKey encrypts the private key using AES-GCM
func encryptPrivateKey(privateKey, password string) (string, error) {
    // Generate a random salt
    salt := make([]byte, saltSize)
    if _, err := io.ReadFull(rand.Reader, salt); err != nil {
        return "", fmt.Errorf("failed to generate salt: %v", err)
    }

    // Derive encryption key from password and salt
    key := deriveKey(password, salt)

    // Create cipher block
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", fmt.Errorf("failed to create cipher: %v", err)
    }

    // Create GCM cipher mode
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("failed to create GCM: %v", err)
    }

    // Generate nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", fmt.Errorf("failed to generate nonce: %v", err)
    }

    // Encrypt the private key
    ciphertext := gcm.Seal(nil, nonce, []byte(privateKey), nil)

    // Combine salt + nonce + ciphertext and encode as base64
    combined := make([]byte, 0, len(salt)+len(nonce)+len(ciphertext))
    combined = append(combined, salt...)
    combined = append(combined, nonce...)
    combined = append(combined, ciphertext...)

    return base64.StdEncoding.EncodeToString(combined), nil
}

// decryptPrivateKey decrypts the private key using AES-GCM
func decryptPrivateKey(encryptedKey, password string) (string, error) {
    // Decode from base64
    combined, err := base64.StdEncoding.DecodeString(encryptedKey)
    if err != nil {
        return "", fmt.Errorf("failed to decode encrypted key: %v", err)
    }

    // Extract salt, nonce, and ciphertext
    salt := combined[:saltSize]
    key := deriveKey(password, salt)

    // Create cipher block
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", fmt.Errorf("failed to create cipher: %v", err)
    }

    // Create GCM cipher mode
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("failed to create GCM: %v", err)
    }

    nonceSize := gcm.NonceSize()
    if len(combined) < saltSize+nonceSize {
        return "", fmt.Errorf("invalid encrypted key format")
    }

    nonce := combined[saltSize : saltSize+nonceSize]
    ciphertext := combined[saltSize+nonceSize:]

    // Decrypt the private key
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", fmt.Errorf("failed to decrypt private key: %v", err)
    }

    return string(plaintext), nil
}

// GenerateNewKeypair generates a new Solana keypair and saves it to an encrypted file
func GenerateNewKeypair(password, keystorePath string) (*solana.PrivateKey, error) {
    // Generate new keypair
    account := solana.NewWallet()
    privateKey := account.PrivateKey
    
    // Create keystore directory if it doesn't exist
    if err := os.MkdirAll(filepath.Dir(keystorePath), 0700); err != nil {
        return nil, fmt.Errorf("failed to create keystore directory: %v", err)
    }

    // Encrypt private key with password
    keyData, _ := encryptPrivateKey(privateKey.String(), password)
    
    // Create keystore structure
    keystore := EncryptedKeystore{
        Address: privateKey.PublicKey().String(),
        KeyData: keyData,
        Version: 1,
    }

    // Save to file
    file, err := json.MarshalIndent(keystore, "", "  ")
    if err != nil {
        return nil, fmt.Errorf("failed to marshal keystore: %v", err)
    }

    if err := os.WriteFile(keystorePath, file, 0600); err != nil {
        return nil, fmt.Errorf("failed to write keystore file: %v", err)
    }

    return &privateKey, nil
}

// LoadKeypair loads a keypair from an encrypted keystore file
func LoadKeypair(keystorePath, password string) (*solana.PrivateKey, error) {
    // Read keystore file
    file, err := os.ReadFile(keystorePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read keystore file: %v", err)
    }

    // Parse keystore
    var keystore EncryptedKeystore
    if err := json.Unmarshal(file, &keystore); err != nil {
        return nil, fmt.Errorf("failed to parse keystore: %v", err)
    }

    // Decrypt private key
    privateKeyStr, _ := decryptPrivateKey(keystore.KeyData, password)
    privateKey := solana.MustPrivateKeyFromBase58(privateKeyStr)

    return &privateKey, nil
}