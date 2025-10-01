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

	// Get username
	fmt.Print("Enter your username: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	username := scanner.Text()

	// Login
	if err := client.Login(username); err != nil {
		log.Fatal("Login error:", err)
	}

	// Connect to server
	log.Println("Connecting to chat server...")
	if err := client.Connect(":8080"); err != nil {
		log.Fatal("Connection error:", err)
	}
	log.Println("Connected to chat server")

	// Start chat loop
	fmt.Println("Start typing messages (type 'quit' to exit)")
	for scanner.Scan() {
		msg := scanner.Text()
		if strings.ToLower(msg) == "quit" {
			break
		}

		if err := client.SendMessage(msg); err != nil {
			log.Printf("Error sending message: %v", err)
			break
		}
	}

	// Disconnect
	if err := client.Disconnect(); err != nil {
		log.Printf("Error disconnecting: %v", err)
	}
}
