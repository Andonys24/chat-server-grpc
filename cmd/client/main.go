package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Andonys24/chat-server-grpc/internal/chat"
	"github.com/Andonys24/chat-server-grpc/internal/config"
	pb "github.com/Andonys24/chat-server-grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Load Configuration
	cfg := config.LoadConfig()

	// Connect to server
	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	defer conn.Close()

	client := pb.NewChatServiceClient(conn)

	scanner := bufio.NewScanner(os.Stdin)
	var username string

	for {
		fmt.Print("Enter Username: ")
		scanner.Scan()
		username = scanner.Text()

		if chat.IsValidNickname(username) {
			break
		}

		fmt.Println("Invalid username. Use 3-12 chars, start with letter, only letters/numbers/_")
	}

	// Join Chat
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	joinResp, err := client.Join(ctx, &pb.JoinRequest{Username: username})

	if err != nil {
		log.Fatalf("Failed to join: %v", err)
	}

	user := joinResp.GetUser()
	fmt.Printf("Joined as %s (ID: %s)\n", user.GetName(), user.GetId())

	// Create ChatStream
	ctx = context.Background()
	stream, err := client.ChatStream(ctx)
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}

	// Send user joined event
	stream.Send(&pb.ChatMessage{
		Event: &pb.ChatMessage_UserJoined{
			UserJoined: user,
		},
	})

	// Channels for coordination
	done := make(chan struct{})
	msgChan := make(chan string)

	// Start receiving messages from stream
	go receiveMessages(stream, done)

	// Start sending messages to stream
	go sendMessages(stream, user, msgChan, done)

	// Read messages from stdin
	fmt.Println("Type messages (or 'exit' to quit):")

	for scanner.Scan() {
		text := scanner.Text()

		if strings.ToLower(text) == "exit" {
			close(msgChan)
			close(done)
			break
		}

		msgChan <- text
	}
}

func receiveMessages(stream pb.ChatService_ChatStreamClient, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
			msg, err := stream.Recv()

			if err != nil {
				log.Printf("Stream closed: %v", err)
				return
			}

			switch ev := msg.GetEvent().(type) {
			case *pb.ChatMessage_Message:
				m := ev.Message
				fmt.Printf("[%s]: %s\n", m.GetSender().GetName(), m.GetContent())
			case *pb.ChatMessage_UserJoined:
				u := ev.UserJoined
				fmt.Printf(">>> %s joined\n", u.GetName())
			case *pb.ChatMessage_UserLeft:
				u := ev.UserLeft
				fmt.Printf("<<< %s left\n", u.GetName())
			}
		}
	}
}

func sendMessages(stream pb.ChatService_ChatStreamClient, user *pb.User, msgChan <-chan string, done <-chan struct{}) {
	// Send UserLeft whenever function exits
	defer func() {
		stream.Send(&pb.ChatMessage{
			Event: &pb.ChatMessage_UserLeft{
				UserLeft: user,
			},
		})
	}()

	for {
		select {
		case text, ok := <-msgChan:
			if !ok {
				return // postpone execution UserLeft
			}

			stream.Send(&pb.ChatMessage{
				Event: &pb.ChatMessage_Message{
					Message: &pb.Message{
						Id:        fmt.Sprintf("%d", time.Now().UnixNano()),
						Sender:    user,
						Content:   text,
						Timestamp: time.Now().Unix(),
					},
				},
			})
		case <-done:
			return
		}
	}
}
