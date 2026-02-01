package main

import (
	"chat-server-grpc/internal/chat"
	"chat-server-grpc/internal/config"
	"fmt"
)

func main() {
	cfg := config.LoadConfig()
	addres := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	chat.GenerateTitle("Chat Server Go - gRPC", true)

	fmt.Printf("Host: %s\n", addres)
}
