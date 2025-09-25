# Chat Room Application

A feature-rich chat room application built with Go, featuring:

- Real-time messaging
- File sharing
- End-to-end encryption
- Emoji support
- Modern GUI using Fyne framework
- Private messaging
- User authentication and session management

## Project Structure

```
chatroom/
├── cmd/                    # Application entrypoints
├── internal/              # Private application code
├── pkg/                   # Public libraries
└── assets/               # Static resources
```

## Getting Started

1. Run the server:
```bash
go run cmd/server/main.go
```

2. Run the client:
```bash
go run cmd/client/main.go
```

## Features

- Secure communication using AES/RSA encryption
- File sharing capabilities
- Emoji support with picker
- Private messaging between users
- Session management
- Modern and intuitive GUI