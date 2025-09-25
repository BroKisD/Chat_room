package main

import (
	"log"

	"chatroom/internal/client"
	"chatroom/internal/client/gui"
)

func main() {
	client := client.New()
	app := gui.NewApp(client)

	if err := app.Run(); err != nil {
		log.Fatal("Client error:", err)
	}
}
