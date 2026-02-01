# chat-server-grpc

Real-time chat server implemented in Go with gRPC and bidirectional streaming.

## 🚀 Features

- **gRPC Framework**: Modern HTTP/2-based protocol with Protocol Buffers
- **Bidirectional Streaming**: Efficient full-duplex communication between client and server
- **Thread-Safe Concurrency**: Synchronization with `sync.Mutex` and Go channels
- **Global Broadcast**: Real-time message broadcasting to all connected users
- **User Management**: Dynamic listing of connected users via RPC
- **Name Validation**: Regex-based system for valid usernames (3-12 characters)
- **Connection Limits**: Configurable simultaneous connection control
- **Command System**: CLI with slash commands (`/users`, `/clear`, `/exit`, etc.)
- **Structured Logging**: Complete event traceability with timestamps
- **Clean Disconnection**: Automatic notification when users leave

## 📋 Requirements

- Go 1.25.5 or higher
- Protocol Buffers compiler (`protoc`)
- Go plugins for protoc:
  - `protoc-gen-go`
  - `protoc-gen-go-grpc`

## 🛠️ Installation

### 1. Clone the repository
```sh
git clone <your-repository>
cd chat-server-grpc
```

### 2. Install Go dependencies
```sh
go mod download
```

### 3. Install protobuf tools (if you don't have them)

#### Protocol Buffers Compiler
**Windows (Chocolatey)**:
```sh
choco install protoc
```

**macOS**:
```sh
brew install protobuf
```

**Linux**:
Download from [GitHub Releases](https://github.com/protocolbuffers/protobuf/releases)

#### Go Plugins
```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### 4. Generate code from proto (optional, already generated)
```sh
protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. proto/chat.proto
```

### 5. Create `.env` file (optional)
```env
HOST=127.0.0.1
PORT=8080
MAX_CONNECTIONS=50
```

## 🎮 Usage

### Start the Server

```sh
go run cmd/server/main.go
```

Expected output:
```
2026/02/01 14:00:00 gRPC Chat Server listening on 127.0.0.1:8080
```

### Connect a Client

```sh
go run cmd/client/main.go
```

On connection you'll see:
```
************************************************************************************
*                           Chat server with Go and gRPC                           *
************************************************************************************

Enter Username: 
```

Enter a valid name (3-12 characters, starts with letter, alphanumeric and `_` only).

## 📡 Client Commands

| Command | Description | Example |
|---------|-------------|---------|
| `/all <message>` | Send message to all users | `/all Hello everyone` |
| `/users` | List connected users | `/users` |
| `/clear` | Clear the console | `/clear` |
| `/help` | Show available commands | `/help` |
| `/exit` | Disconnect from server | `/exit` |

**Note**: Messages without `/` are automatically sent to all (equivalent to `/all`).

## 🏗️ Architecture

```
chat-server-grpc/
├── cmd/
│   ├── client/
│   │   └── main.go              # gRPC client with CLI
│   └── server/
│       └── main.go              # gRPC server with TCP listener
├── internal/
│   ├── chat/
│   │   ├── server.go            # ChatServiceServer implementation
│   │   ├── ui.go                # Interface helpers (title, clear)
│   │   └── validators.go        # Nickname validation with regex
│   └── config/
│       └── config.go            # Configuration loading from .env
├── proto/
│   ├── chat.proto               # Service and message definitions
│   ├── chat.pb.go               # Generated code (messages)
│   └── chat_grpc.pb.go          # Generated code (service)
├── go.mod
└── go.sum
```

### Main Components

#### Protocol Buffers ([proto/chat.proto](proto/chat.proto))
Defines the gRPC contract with:
- **Messages**: `User`, `Message`, `JoinRequest`, `JoinResponse`, `ChatMessage`, `ListUsersRequest`, `ListUsersResponse`
- **Service `ChatService`**:
  - `Join(JoinRequest) → JoinResponse` (unary RPC)
  - `SendMessage(SendMessageRequest) → SendMessageResponse` (unary RPC)
  - `ChatStream(stream ChatMessage) → stream ChatMessage` (bidirectional streaming)
  - `ListUsers(ListUsersRequest) → ListUsersResponse` (unary RPC)

#### Server ([internal/chat/server.go](internal/chat/server.go))
- **Fields**:
  - `users`: map `userId → User` for user information
  - `clients`: map `userId → stream` for active connections
  - `messages`: slice of message history
  - `maxConnections`: simultaneous connection limit
  - `mu`: mutex to protect concurrent access
- **Methods**:
  - `Join()`: registers user with unique timestamp-based ID
  - `ListUsers()`: returns list of connected usernames
  - `ChatStream()`: handles events (UserJoined, UserLeft, Message)
  - `broadcast()`: sends messages to all connected clients

#### Client ([cmd/client/main.go](cmd/client/main.go))
- **Flow**:
  1. gRPC connection with insecure credentials (development)
  2. Join RPC to obtain User with ID
  3. Open bidirectional ChatStream
  4. Two concurrent goroutines:
     - `receiveMessages()`: listens for server events
     - `sendMessages()`: sends user messages
  5. Main loop: parses commands with `handleInput()`
- **Commands**:
  - Processed locally: `/clear`, `/help`, `/exit`
  - RPC to server: `/users` (ListUsers RPC)
  - Stream: normal messages and `/all`

## 🔒 Concurrency and Safety

### Server
- **`sync.Mutex`**: Protects access to `users`, `clients`, and `messages`
- **Safe Broadcast**: Full lock during sending to all streams
- **Automatic Cleanup**: Removes disconnected clients from map
- **Limit Validation**: Rejects connections if `len(clients) >= maxConnections`
- **Dual Registry**: Separation of `users` (data) and `clients` (streams)

### Client
- **Coordination Channels**:
  - `done`: shutdown signal for goroutines
  - `msgChan`: transmits messages from input to stream
- **Defer Pattern**: `sendMessages` guarantees `UserLeft` sent on exit
- **Context Timeout**: RPCs with 5s timeout to avoid blocking
- **Auto-exit**: Automatic shutdown on server errors

## 🎨 gRPC Protocol

### Message Types

#### `ChatMessage` (oneof event)
Polymorphic message for streaming:
```protobuf
message ChatMessage {
  oneof event {
    Message message = 1;         // Chat message
    User user_joined = 2;         // User joined
    User user_left = 3;           // User left
  }
}
```

#### `Message`
```protobuf
message Message {
  string id = 1;           // Unique ID (timestamp)
  User sender = 2;         // Sender user
  string content = 3;      // Message content
  int64 timestamp = 4;     // Unix timestamp
}
```

### Service RPCs

1. **Join** (Unary)
   - Client sends username
   - Server validates, generates unique ID, returns User

2. **ChatStream** (Bidirectional Streaming)
   - Client: sends UserJoined on connection
   - Client/Server: exchange Message events
   - Client: sends UserLeft on disconnection
   - Server: broadcasts all events

3. **ListUsers** (Unary)
   - Client requests list
   - Server returns array of connected usernames

## ⚙️ Configuration

Environment variables (`.env` file or default values):

| Variable | Description | Default Value |
|----------|-------------|---------------|
| `HOST` | Server host | `localhost` |
| `PORT` | TCP port | `8080` |
| `MAX_CONNECTIONS` | Maximum simultaneous connections | `50` |

Configuration loaded from [internal/config/config.go](internal/config/config.go) with `godotenv`.

## 🧪 Validations

### Nickname ([internal/chat/validators.go](internal/chat/validators.go))
```go
^[a-zA-Z][a-zA-Z0-9_]{2,11}$
```
- **Length**: 3-12 characters
- **First character**: Letter (a-z, A-Z)
- **Allowed characters**: Alphanumeric + underscore (`_`)

## 🔄 Connection Flow

```
Client                           Server
  |                                 |
  |--- gRPC.NewClient ------------->|
  |                                 |
  |--- Join(username) ------------->|
  |                                 |--- Validates username
  |                                 |--- Generates User(id, name)
  |                                 |--- Saves in users map
  |<-- JoinResponse(user) ----------|
  |                                 |
  |--- ChatStream() --------------->|
  |                                 |
  |--- UserJoined event ----------->|
  |                                 |--- Saves in clients map
  |                                 |--- broadcast(UserJoined)
  |<-- UserJoined broadcast --------|
  |                                 |
  |--- Message event -------------->|
  |                                 |--- Saves in messages
  |                                 |--- broadcast(Message)
  |<-- Message broadcast -----------|
  |                                 |
  |    (user types /exit)           |
  |--- UserLeft event ------------->|
  |                                 |--- Removes from users/clients
  |                                 |--- broadcast(UserLeft)
  |<-- UserLeft broadcast ----------|
  |                                 |
  |--- Closes stream -------------->|
  |                                 |--- Automatic cleanup
```

## 📊 Server Logs

Example output:
```
2026/02/01 14:00:00 gRPC Chat Server listening on 127.0.0.1:8080
2026/02/01 14:00:15 User Luis joined (ID: 1769968835420676100, total: 1)
2026/02/01 14:00:20 Message from Luis: Hello everyone
2026/02/01 14:00:25 User Pablo joined (ID: 1769968862402786500, total: 2)
2026/02/01 14:00:30 ListUsers called, returning 2 users
2026/02/01 14:00:40 User Luis left (ID: 1769968835420676100, total: 1)
2026/02/01 14:00:45 User Pablo left (ID: 1769968862402786500, total: 0)
```

## 🆚 Comparison with chat-server-go

Reference repo: https://github.com/Andonys24/chat-server-go.git

| Aspect | chat-server-go | chat-server-grpc |
|--------|----------------|------------------|
| **Protocol** | Custom (`HEADER\|CONTENT`) | gRPC + Protocol Buffers |
| **Serialization** | Manual with delimiters | Automatic (protobuf) |
| **Streaming** | Custom full-duplex | Native bidirectional streaming |
| **Code generation** | Manual | Auto-generated from `.proto` |
| **Type safety** | Manual parsing | Strongly typed |
| **Versioning** | Ad-hoc | Backward compatible (protobuf) |
| **Performance** | Good | Excellent (HTTP/2, binary compression) |
| **Standard** | Proprietary | Industry standard (Google) |

## 🚀 gRPC Advantages

- ✅ **Automatic generation**: Client and server from a `.proto` file
- ✅ **Strong typing**: Compiler catches errors
- ✅ **HTTP/2**: Multiplexing, header compression
- ✅ **Native streaming**: First-class support for streams
- ✅ **Interoperability**: Compatible with other languages
- ✅ **Backward compatibility**: API evolution without breaking clients

## 📄 License

This project is under the MIT License.

## 👤 Author

Developed as an educational project to learn gRPC, Protocol Buffers, and real-time chat architectures with Go.

---

**Note**: This server uses insecure credentials (`insecure.NewCredentials()`) for local development. For production, implement TLS/SSL, token-based authentication, rate limiting, and additional security validations.
