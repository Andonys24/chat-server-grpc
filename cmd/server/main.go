package main

import (
	"fmt"
	"log"
	"net"

	"github.com/Andonys24/chat-server-grpc/internal/chat"
	"github.com/Andonys24/chat-server-grpc/internal/config"
	pb "github.com/Andonys24/chat-server-grpc/proto"
	"google.golang.org/grpc"
)

func main() {
	// Load Configuration
	cfg := config.LoadConfig()

	// Create TCP listener
	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	listener, err := net.Listen("tcp", address)

	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", address, err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register ChatService
	chatServer := chat.NewServer()
	pb.RegisterChatServiceServer(grpcServer, chatServer)

	log.Printf("gRPC Chat Server listening on %s", address)

	// Start serving
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

}
