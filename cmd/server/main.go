package main

import (
	"chatroom/internal/server"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	newKey := flag.Bool("n", false, "Generate a new room key (delete existing savestate)")
	oldKey := flag.Bool("o", false, "Use existing room key if available")
	flag.Parse()

	if *newKey && *oldKey {
		log.Fatal("You cannot use both -n and -o at the same time.")
	}

	const saveFile1 = "room.key"
	const saveFile2 = "server_state.json"
	if *newKey {
		fmt.Println("[INFO] Starting server with a NEW room key...")
		if err := os.Remove(saveFile1); err != nil && !os.IsNotExist(err) {
			log.Fatalf("Failed to remove old savestate: %v", err)
		}
		if err := os.Remove(saveFile2); err != nil && !os.IsNotExist(err) {
			log.Fatalf("Failed to remove old savestate: %v", err)
		}
	} else if *oldKey {
		fmt.Println("[INFO] Starting server with EXISTING room key...")
	} else {
		fmt.Println("[INFO] Starting server (default behavior — use existing room key if any)")
	}

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
