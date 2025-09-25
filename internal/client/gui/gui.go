package gui

import (
	"chatroom/internal/client"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

type App struct {
	client     *client.Client
	mainWindow fyne.Window
}

func NewApp(client *client.Client) *App {
	a := &App{
		client: client,
	}
	return a
}

func (a *App) Run() error {
	fyneApp := app.New()
	a.mainWindow = fyneApp.NewWindow("Chat Room")

	// Setup UI components

	a.mainWindow.ShowAndRun()
	return nil
}
