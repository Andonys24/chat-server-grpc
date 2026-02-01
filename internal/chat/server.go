package chat

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	pb "github.com/Andonys24/chat-server-grpc/proto"
)

type Server struct {
	pb.UnimplementedChatServiceServer
	mu             sync.Mutex
	maxConnections int
	users          map[string]*pb.User
	messages       []*pb.Message
	clients        map[string]pb.ChatService_ChatStreamServer
}

func NewServer(maxConnections int) *Server {
	return &Server{
		maxConnections: maxConnections,
		users:          make(map[string]*pb.User),
		clients:        make(map[string]pb.ChatService_ChatStreamServer),
	}
}

func (s *Server) Join(ctx context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	user := &pb.User{
		Id:   fmt.Sprintf("%d", time.Now().UnixNano()),
		Name: req.GetUsername(),
	}

	s.mu.Lock()
	s.users[user.Id] = user
	s.mu.Unlock()

	return &pb.JoinResponse{
		User:    user,
		Message: "joined",
	}, nil
}

func (s *Server) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	msg := &pb.Message{
		Id:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Sender:    s.users[req.GetUserId()],
		Content:   req.GetContent(),
		Timestamp: time.Now().Unix(),
	}

	s.mu.Lock()
	s.messages = append(s.messages, msg)
	s.mu.Unlock()

	return &pb.SendMessageResponse{
		Success:   true,
		MessageId: msg.Id,
	}, nil
}

func (s *Server) ChatStream(stream pb.ChatService_ChatStreamServer) error {
	var userId string

	for {
		in, err := stream.Recv()

		if err != nil {
			break
		}

		switch ev := in.GetEvent().(type) {
		case *pb.ChatMessage_UserJoined:
			user := ev.UserJoined
			userId = user.GetId()

			s.mu.Lock()
			if len(s.clients) >= s.maxConnections {
				s.mu.Unlock()
				log.Printf("Max connections reached, rejecting user %s", user.Name)
				return fmt.Errorf("Server full")
			}

			s.users[user.Id] = user
			s.clients[user.Id] = stream
			s.mu.Unlock()

			log.Printf("User %s joined (ID: %s, total: %d)", user.Name, user.Id, len(s.clients))

			s.broadcast(&pb.ChatMessage{
				Event: &pb.ChatMessage_UserJoined{UserJoined: user},
			})
		case *pb.ChatMessage_UserLeft:
			user := ev.UserLeft

			s.mu.Lock()
			delete(s.users, user.GetId())
			delete(s.clients, user.GetId())
			totalClients := len(s.clients)
			s.mu.Unlock()

			log.Printf("User %s left (ID: %s, total: %d)", user.Name, user.Id, totalClients)

			s.broadcast(&pb.ChatMessage{
				Event: &pb.ChatMessage_UserLeft{UserLeft: user},
			})
		case *pb.ChatMessage_Message:
			msg := ev.Message

			log.Printf("Message from %s: %s", msg.Sender.Name, msg.Content)

			s.mu.Lock()
			s.messages = append(s.messages, msg)
			s.mu.Unlock()

			s.broadcast(&pb.ChatMessage{
				Event: &pb.ChatMessage_Message{Message: msg},
			})
		}
	}

	if userId != "" {
		s.mu.Lock()
		_, exists := s.clients[userId]
		if exists {
			delete(s.clients, userId)
			totalClients := len(s.clients)
			s.mu.Unlock()
			log.Printf("User ID %s disconnected unexpectedly (total: %d)", userId, totalClients)
		} else {
			s.mu.Unlock()
		}
	}

	return nil
}

func (s *Server) broadcast(msg *pb.ChatMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, client := range s.clients {
		_ = client.Send(msg)
	}
}
