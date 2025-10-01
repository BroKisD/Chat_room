package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"chatroom/internal/client"
)

func main() {
	client := client.New()

	// Set message handler
	client.SetMessageHandler(func(msg string) {
		fmt.Println(msg)
	})

	// Get username
	fmt.Print("Enter your username: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	username := strings.TrimSpace(scanner.Text())

	if username == "" {
		log.Fatal("Username cannot be empty")
	}

	// Login
	if err := client.Login(username); err != nil {
		log.Fatal("Login error:", err)
	}

	// Connect to server
	log.Println("Connecting to chat server...")
	if err := client.Connect(":8088"); err != nil {
		log.Fatal("Connection error:", err)
	}
	log.Println("Connected to chat server")

	// Print help
	fmt.Println("\nCommands:")
	fmt.Println("  /w <username> <message> - Send private message")
	fmt.Println("  quit - Exit the chat")
	fmt.Println("\nStart typing messages:")

	// Start chat loop
	for scanner.Scan() {
		msg := scanner.Text()
		if strings.ToLower(msg) == "quit" {
			break
		}

		var err error
		if strings.HasPrefix(msg, "/w ") {
			// Parse private message command
			parts := strings.SplitN(msg, " ", 3)
			if len(parts) < 3 {
				fmt.Println("Invalid whisper format. Use: /w username message")
				continue
			}
			err = client.SendPrivateMessage(parts[1], parts[2])
		} else {
			err = client.SendMessage(msg)
		}

		if err != nil {
			log.Printf("Error sending message: %v", err)
			break
		}
	}

	// Disconnect
	if err := client.Disconnect(); err != nil {
		log.Printf("Error disconnecting: %v", err)
	}
}
