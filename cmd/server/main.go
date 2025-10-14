package main

import (
	"chatroom/internal/server"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	srv := server.New(":9000")

	defer func() {
		if r := recover(); r != nil {
			log.Println("[PANIC] Recovered, saving state before exit...")
			srv.SaveState()
			panic(r)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Println("Server started on :9000")

	<-stop
	log.Println("[INFO] Interrupt signal received. Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[ERROR] Server forced to shutdown: %v", err)
	}

	log.Println("[INFO] Server shutdown complete.")
}
