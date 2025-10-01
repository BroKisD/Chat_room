package gui

import (
	"fmt"
	"image/color"
	"strings"

	"chatroom/internal/client"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type App struct {
	client     *client.Client
	mainWindow fyne.Window
	messages   *widget.RichText
	userList   *widget.List
	input      *widget.Entry
	users      []string
	connected  bool
	msgHistory []string
}

func NewApp(client *client.Client) *App {
	a := &App{
		client:     client,
		msgHistory: make([]string, 0),
		users:      make([]string, 0),
	}

	// Set message handler
	client.SetMessageHandler(a.handleMessage)

	return a
}

func createBorderedContainer(content fyne.CanvasObject, title string) *fyne.Container {
	// Create a red border
	border := canvas.NewRectangle(color.NRGBA{R: 255, A: 255})
	border.StrokeWidth = 2
	border.StrokeColor = color.NRGBA{R: 255, A: 255}
	border.FillColor = color.NRGBA{R: 0, G: 0, B: 0, A: 0}

	// Create title if provided
	var titleObj fyne.CanvasObject
	if title != "" {
		titleText := canvas.NewText(title, color.White)
		titleText.TextSize = 20
		titleBg := canvas.NewRectangle(color.NRGBA{R: 255, A: 255})
		titleBg.StrokeWidth = 0
		titleObj = container.NewStack(titleBg, titleText)
	}

	if titleObj != nil {
		return container.NewBorder(titleObj, nil, nil, nil,
			container.NewStack(border, content))
	}
	return container.NewStack(border, content)
}

func (a *App) Run() error {
	fyneApp := app.NewWithID("com.chatroom.app")
	a.mainWindow = fyneApp.NewWindow("FUV Chatroom")
	a.mainWindow.Resize(fyne.NewSize(1000, 700))

	// Setup main UI components
	a.setupUI()

	// Show login dialog first
	a.showLoginDialog()

	// Set close handler
	a.mainWindow.SetCloseIntercept(func() {
		if a.connected {
			a.client.Disconnect()
		}
		a.mainWindow.Close()
	})

	a.mainWindow.ShowAndRun()
	return nil
}

func (a *App) setupUI() {
	// Message display area with rich text
	a.messages = widget.NewRichText()
	messagesScroll := container.NewScroll(a.messages)
	messagesContainer := createBorderedContainer(messagesScroll, "FUV Chatroom")

	// User list with status indicators
	a.userList = widget.NewList(
		func() int { return len(a.users) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				canvas.NewCircle(color.NRGBA{G: 255, A: 255}),
				widget.NewLabel("Template"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*fyne.Container).Objects[1].(*widget.Label)
			label.SetText(a.users[id])
			circle := obj.(*fyne.Container).Objects[0].(*canvas.Circle)
			circle.FillColor = color.NRGBA{G: 255, A: 255}
			circle.Resize(fyne.NewSize(8, 8))
		},
	)
	userScroll := container.NewScroll(a.userList)
	userContainer := createBorderedContainer(userScroll, "Active")

	// Input area
	a.input = widget.NewMultiLineEntry()
	a.input.SetPlaceHolder("Type your message...")
	sendBtn := widget.NewButtonWithIcon("Send", theme.MailSendIcon(), a.sendMessage)
	emojiBtn := widget.NewButton("☺", a.showEmojiPicker)

	inputBox := container.NewBorder(
		nil, nil,
		emojiBtn, sendBtn,
		a.input,
	)
	inputContainer := createBorderedContainer(inputBox, "")

	// Layout
	right := container.NewBorder(
		nil, nil, nil, nil,
		userContainer,
	)

	content := container.NewBorder(
		nil,            // top
		inputContainer, // bottom
		nil, nil,       // left, right
		container.NewHSplit(messagesContainer, right),
	)

	a.mainWindow.SetContent(content)
}

func (a *App) showLoginDialog() {
	username := widget.NewEntry()
	username.SetPlaceHolder("Enter your username")

	content := container.NewVBox(
		widget.NewLabel("Welcome to Chat Room"),
		username,
	)

	dialog.ShowCustomConfirm("Login", "Connect", "Cancel", content, func(connect bool) {
		if !connect || username.Text == "" {
			a.mainWindow.Close()
			return
		}

		// Try to login and connect
		if err := a.client.Login(username.Text); err != nil {
			dialog.ShowError(err, a.mainWindow)
			return
		}

		if err := a.client.Connect(":9000"); err != nil {
			dialog.ShowError(err, a.mainWindow)
			return
		}

		a.connected = true
	}, a.mainWindow)
}

func (a *App) sendMessage() {
	text := strings.TrimSpace(a.input.Text)
	if text == "" {
		return
	}

	var err error
	if strings.HasPrefix(text, "/w ") {
		// Extract target and message
		parts := strings.SplitN(text, " ", 3)
		if len(parts) < 3 {
			dialog.ShowError(fmt.Errorf("Invalid whisper format. Use: /w username message"), a.mainWindow)
			return
		}
		err = a.client.SendPrivateMessage(parts[1], parts[2])
	} else {
		err = a.client.SendMessage(text)
	}

	if err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}

	a.input.SetText("")
}

func (a *App) showEmojiPicker() {
	// Simple emoji picker
	emojis := []string{"😊", "❤️", "👍", "😀", "😃", "😄", "😁", "😅", "😂", "🤣"}
	emojiButtons := make([]fyne.CanvasObject, len(emojis))

	for i, emoji := range emojis {
		e := emoji // Create a new variable for closure
		emojiButtons[i] = widget.NewButton(e, func() {
			a.input.SetText(a.input.Text + e)
		})
	}

	emojiGrid := container.NewGridWithColumns(5, emojiButtons...)
	dialog.ShowCustom("Emojis", "Close", emojiGrid, a.mainWindow)
}

func (a *App) handleMessage(msg string) {
	if a.messages == nil {
		return
	}

	var segment widget.RichTextSegment

	// Add timestamp
	if strings.Contains(msg, "(System)") {
		segment = &widget.TextSegment{
			Style: widget.RichTextStyle{
				ColorName: theme.ColorNameForeground,
				TextStyle: fyne.TextStyle{Italic: true},
			},
			Text: msg + "\n",
		}
	} else if strings.Contains(msg, "(Private)") {
		segment = &widget.TextSegment{
			Style: widget.RichTextStyle{
				ColorName: theme.ColorNameError,
			},
			Text: msg + "\n",
		}
	} else if strings.Contains(msg, "(Global)") {
		segment = &widget.TextSegment{
			Style: widget.RichTextStyle{
				ColorName: theme.ColorNamePrimary,
			},
			Text: msg + "\n",
		}
	} else if strings.HasPrefix(msg, "Active users: ") {
		users := strings.TrimPrefix(msg, "Active users: ")
		a.users = strings.Split(users, ", ")
		a.userList.Refresh()
		return
	}

	a.messages.Segments = append(a.messages.Segments, segment)
	a.messages.Refresh()
}
