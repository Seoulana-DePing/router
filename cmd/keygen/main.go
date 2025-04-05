package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/router/keystore"
)

func main() {
	// 커맨드라인 플래그 정의
    password := flag.String("password", "", "Password for the keystore")
    outputDir := flag.String("output", "./keystore", "Directory to store the keystore file")
    flag.Parse()

    // 비밀번호 체크
    if *password == "" {
        // 환경변수에서 비밀번호 확인
        *password = os.Getenv("KEYSTORE_PASSWORD")
        if *password == "" {
            log.Fatal("Password must be provided either via -password flag or KEYSTORE_PASSWORD environment variable")
        }
    }

    // 출력 디렉토리 생성
    if err := os.MkdirAll(*outputDir, 0700); err != nil {
        log.Fatalf("Failed to create output directory: %v", err)
    }

    // keystore 파일 경로 설정
    keystorePath := filepath.Join(*outputDir, "routerkey.json")

    // 이미 파일이 존재하는지 확인
    if _, err := os.Stat(keystorePath); err == nil {
        log.Fatal("Keystore file already exists. Please remove it first or use a different output directory")
    }

    // 새 키페어 생성
    privateKey, err := keystore.GenerateNewKeypair(*password, keystorePath)
    if err != nil {
        log.Fatalf("Failed to generate keypair: %v", err)
    }

    fmt.Println("Successfully generated new keypair!")
    fmt.Printf("Public Key: %s\n", privateKey.PublicKey().String())
    fmt.Printf("Keystore saved to: %s\n", keystorePath)
    fmt.Println("\nPlease backup your keystore file and keep your password safe!")
}
