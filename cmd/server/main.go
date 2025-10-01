package main

import (
	"chatroom/internal/server"
	"chatroom/pkg/logger"
)

func main() {
	l := logger.New("server")
	srv := server.New(":9000")

	l.Info("Starting chat server on :9000")
	if err := srv.Start(); err != nil {
		l.Error("Server error:", err)
	}
}
