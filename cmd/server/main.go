package main

import (
	"log"

	"chatroom/internal/server"
	"chatroom/pkg/logger"
)

func main() {
	logger := logger.New("server")
	srv := server.New(":8080")

	log.Printf("Starting chat server on :8080")
	if err := srv.Start(); err != nil {
		log.Fatal("Server error:", err)
	}
}
