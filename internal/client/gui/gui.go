package gui

import (
	"os"
	"fmt"
	"image/color"
	"log"
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
	client         *client.Client
	mainWindow     fyne.Window
	messages       *widget.RichText
	messagesScroll *container.Scroll
	userList       *widget.List
	input          *widget.Entry
	users          []string
	connected      bool
	msgHistory     []string
	incoming       chan string
}

func NewApp(client *client.Client) *App {
	a := &App{
		client:     client,
		msgHistory: make([]string, 0),
		users:      make([]string, 0),
		incoming:   make(chan string, 100),
	}

	client.SetMessageHandler(func(msg string) {
		fmt.Println("[DEBUG] Raw from server:", msg)
		a.incoming <- msg
	})

	go a.dispatchMessages()

	return a
}

func createBorderedContainer(content fyne.CanvasObject, title string) *fyne.Container {
	// Create an orange border
	orange := color.NRGBA{R: 255, G: 140, A: 255} // orange
	border := canvas.NewRectangle(color.White)
	border.StrokeWidth = 3
	border.StrokeColor = orange
	border.FillColor = color.White

	// Create title if provided
	var titleObj fyne.CanvasObject
	if title != "" {
		titleText := canvas.NewText(title, orange)
		titleText.TextSize = 20
		titleBg := canvas.NewRectangle(color.White)
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
	fyneApp.Settings().SetTheme(theme.LightTheme())
	a.mainWindow = fyneApp.NewWindow("Talkie")
	a.mainWindow.Resize(fyne.NewSize(500, 600))

	a.mainWindow.CenterOnScreen()
	
	iconPath := "assets/app_icon.png"
	iconData, err := os.ReadFile(iconPath)
	if err != nil {
		fyne.LogError("Failed to load icon:", err)
	} else {
		iconResource := fyne.NewStaticResource("app_icon.png", iconData)
		a.mainWindow.SetIcon(iconResource)
	}

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
	a.messagesScroll = container.NewScroll(a.messages)
	messagesContainer := createBorderedContainer(a.messagesScroll, "Talkie Chat Room")

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
	userContainer.Resize(fyne.NewSize(10, 10))

	// Input area
	a.input = widget.NewMultiLineEntry()
	a.input.SetPlaceHolder("Type your message...")

	sendBtn := widget.NewButtonWithIcon("Send", theme.MailSendIcon(), nil)
	sendBtn.Importance = widget.HighImportance
	sendBtn.OnTapped = func() {
		content := a.input.Text
		content = ConvertEmojis(content)
		a.sendMessage(content)
	}

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

func (a *App) showEmojiPicker() {
	tabs := container.NewAppTabs()

	for category, group := range emojiMap {
		buttons := []fyne.CanvasObject{}
		for _, emoji := range group {
			e := emoji
			buttons = append(buttons, widget.NewButton(e, func() {
				a.input.SetText(a.input.Text + e)
			}))
		}
		grid := container.NewGridWithColumns(6, buttons...)
		tabs.Append(container.NewTabItem(category, grid))
	}

	dialog.ShowCustom("Emojis", "Close", tabs, a.mainWindow)
}

func GetEmojiList() []string {
	emojis := []string{}
	for _, group := range emojiMap {
		for _, emoji := range group {
			emojis = append(emojis, emoji)
		}
	}
	return emojis
}

func (a *App) dispatchMessages() {
	for msg := range a.incoming {
		log.Println("Received message:", msg)
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "New Message",
			Content: msg,
		})

		a.processMessage(msg)
		log.Println("Processed message in UI thread")
	}
}

func (a *App) processMessage(msg string) {
	if strings.HasPrefix(msg, "Active users: ") {
		users := strings.TrimPrefix(msg, "Active users: ")
		a.users = strings.Split(users, ", ")
		a.userList.Refresh()
		return
	}

	segment := &widget.TextSegment{
		Style: widget.RichTextStyle{
			ColorName: theme.ColorNameForeground,
		},
		Text: msg + "\n",
	}
	a.messages.Segments = append(a.messages.Segments, segment)
	a.messages.Refresh()
	a.messagesScroll.ScrollToBottom()
}

func (a *App) sendMessage(content string) {
	var err error
	raw := content
	if strings.HasPrefix(raw, "/w ") {
		text := strings.TrimSpace(raw)
		parts := strings.SplitN(text, " ", 3)
		if len(parts) < 3 {
			dialog.ShowError(fmt.Errorf("invalid whisper format. use: /w username message"), a.mainWindow)
			return
		}
		err = a.client.SendPrivateMessage(parts[1], parts[2])
	} else {
		text := strings.TrimSpace(raw)
		err = a.client.SendMessage(text)
	}

	if err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}

	a.input.SetText("")
}
