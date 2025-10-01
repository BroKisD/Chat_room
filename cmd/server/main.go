package main

import (
	"chatroom/internal/server"
	"chatroom/pkg/logger"
)

func main() {
	l := logger.New("server")
	srv := server.New(":8088")

	l.Info("Starting chat server on :8088")
	if err := srv.Start(); err != nil {
		l.Error("Server error:", err)
	}
}
