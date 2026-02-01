package main

import (
	"fmt"

	"github.com/Andonys24/chat-server-grpc/internal/chat"
	"github.com/Andonys24/chat-server-grpc/internal/config"
)

func main() {
	cfg := config.LoadConfig()
	addres := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	chat.GenerateTitle("Chat Server Go - gRPC", true)

	fmt.Printf("Host: %s\n", addres)
}
