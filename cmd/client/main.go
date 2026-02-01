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
	chat.GenerateTitle("Chat server with Go and gRPC", true)
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

		if text == "" {
			continue
		}

		_, shouldExit := handleInput(text, user, client, msgChan, done)
		if shouldExit {
			break
		}
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
				os.Exit(1)
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

func handleInput(text string, user *pb.User, client pb.ChatServiceClient, msgChan chan<- string, done chan<- struct{}) (handled bool, shouldExit bool) {
	if !strings.HasPrefix(text, "/") {
		// It's not a command, it sends as a normal message
		msgChan <- text
		return false, false
	}

	parts := strings.SplitN(text, " ", 2)
	cmd := strings.ToLower(parts[0])
	shouldExit = false

	switch cmd {
	case "/clear":
		chat.CleanConsole()
	case "/help":
		printHelp()
	case "/exit":
		close(msgChan)
		close(done)
		shouldExit = true
	case "/all":
		if len(parts) < 2 {
			fmt.Println("Usage: /all <message>")
		} else {
			msgChan <- parts[1]
		}
	case "/users":
		listUser(client)
	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", cmd)
	}

	return true, shouldExit
}

func printHelp() {
	fmt.Println(`=== Available Commands ===
/all <message>   - Send message to all users
/users          - List connected users
/clear          - Clear screen
/help           - Show this help message
/exit           - Disconnect and exit
========================`)
}

func listUser(client pb.ChatServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := client.ListUsers(ctx, &pb.ListUsersRequest{})

	if err != nil {
		fmt.Printf("Error listing users: %v\n", err)
		return
	}

	fmt.Print("\n=== Connected Users ===\n")
	if len(resp.Usernames) == 0 {
		fmt.Println("(No users connected)")
	} else {
		for i, username := range resp.Usernames {
			fmt.Printf("%d. %s\n", i+1, username)
		}
	}
	fmt.Print("=======================\n\n")
}
