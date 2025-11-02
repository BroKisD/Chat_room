package gui

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	messageList    *fyne.Container
	currentMsg     string
}

func NewApp(client *client.Client) *App {
	a := &App{
		client:     client,
		msgHistory: make([]string, 0),
		users:      make([]string, 0),
		incoming:   make(chan string, 100),
		currentMsg: "",
	}

	client.SetMessageHandler(func(msg string) {
		fmt.Println("[DEBUG] Raw to server:", msg)
		a.incoming <- msg
	})

	go a.dispatchMessages()

	return a
}

func createSimpleBox(content fyne.CanvasObject, title string, bgColor color.Color) *fyne.Container {
	bg := canvas.NewRectangle(bgColor)

	if title != "" {
		titleLabel := widget.NewLabel(title)
		titleLabel.TextStyle = fyne.TextStyle{Bold: true}
		header := container.NewVBox(titleLabel, widget.NewSeparator())
		return container.NewStack(
			bg,
			container.NewBorder(header, nil, nil, nil, container.NewPadded(content)),
		)
	}
	return container.NewStack(bg, container.NewPadded(content))
}

func (a *App) Run() error {
	fyneApp := app.NewWithID("com.chatroom.app")
	fyneApp.Settings().SetTheme(theme.LightTheme())
	a.mainWindow = fyneApp.NewWindow("Talkie Chat")
	a.mainWindow.Resize(fyne.NewSize(800, 600))
	a.mainWindow.CenterOnScreen()

	iconPath := "assets/app_icon.png"
	iconData, err := os.ReadFile(iconPath)
	if err != nil {
		fyne.LogError("Failed to load icon:", err)
	} else {
		iconResource := fyne.NewStaticResource("app_icon.png", iconData)
		a.mainWindow.SetIcon(iconResource)
	}

	a.setupUI()
	a.showLoginDialog()

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
	// Messages area
	messageList := container.NewVBox()
	a.messageList = messageList
	a.messagesScroll = container.NewVScroll(messageList)

	messagesContainer := createSimpleBox(a.messagesScroll, "Chat Room", color.White)

	// User list - simple and clean with light blue background
	a.userList = widget.NewList(
		func() int { return len(a.users) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("Template User")
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			username := a.users[id]

			// Highlight "me" in bold
			if strings.HasSuffix(username, " (you)") {
				label.SetText("• " + username)
				label.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				label.SetText("• " + username)
				label.TextStyle = fyne.TextStyle{}
			}
		},
	)

	userScroll := container.NewScroll(a.userList)
	userContainer := createSimpleBox(userScroll, "Online Users", color.NRGBA{R: 230, G: 240, B: 255, A: 255})

	// Input area - clean buttons
	a.input = widget.NewMultiLineEntry()
	a.input.SetPlaceHolder("Type your message...")
	a.input.Wrapping = fyne.TextWrapWord
	a.input.SetMinRowsVisible(3)

	// Enable Enter key to send message
	a.input.OnSubmitted = func(text string) {
		content := a.input.Text
		content = ConvertEmojis(content)
		a.sendMessage(content)
	}

	sendBtn := widget.NewButton("Send", func() {
		content := a.input.Text
		content = ConvertEmojis(content)
		a.sendMessage(content)
	})
	sendBtn.Importance = widget.HighImportance

	fileBtn := widget.NewButton("Send File", func() {
		a.showFilePicker()
	})

	emojiBtn := widget.NewButton("😊 Emoji", a.showEmojiPicker)

	// Put send button beside the text box
	inputWithSend := container.NewBorder(
		nil, nil, nil, sendBtn,
		a.input,
	)

	buttonBox := container.NewHBox(
		emojiBtn,
		fileBtn,
	)

	inputBox := container.NewBorder(
		nil, buttonBox, nil, nil,
		inputWithSend,
	)

	// Main layout with clear separation
	chatArea := container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), inputBox),
		nil, nil,
		messagesContainer,
	)

	split := container.NewHSplit(chatArea, userContainer)
	split.SetOffset(0.75)

	a.mainWindow.SetContent(split)
}

func (a *App) showLoginDialog() {
	username := widget.NewEntry()
	username.SetPlaceHolder("Enter your username")

	// Enable Enter key to connect
	username.OnSubmitted = func(text string) {
		if text != "" {
			a.attemptLogin(text)
		}
	}

	content := container.NewVBox(
		widget.NewLabel("Welcome to Talkie Chat Room"),
		widget.NewSeparator(),
		widget.NewLabel("Username:"),
		username,
	)

	dialog.ShowCustomConfirm("Login", "Connect", "Cancel", content, func(connect bool) {
		if !connect {
			a.mainWindow.Close()
			return
		}

		if username.Text == "" {
			dialog.ShowInformation("Invalid Input", "Please enter a username.", a.mainWindow)
			a.reopenLogin()
			return
		}

		a.attemptLogin(username.Text)
	}, a.mainWindow)
}

func (a *App) attemptLogin(username string) {
	if err := a.client.Login(username); err != nil {
		dialog.ShowError(err, a.mainWindow)
		a.reopenLogin()
		return
	}

	if err := a.client.Connect(":9000"); err != nil {
		if strings.Contains(err.Error(), "username") {
			dialog.ShowError(fmt.Errorf("login failed: %s", err.Error()), a.mainWindow)
			a.reopenLogin()
			return
		}

		dialog.ShowConfirm("Connection failed",
			"Cannot connect to server.\nDo you want to retry?",
			func(confirm bool) {
				if confirm {
					a.reopenLogin()
				} else {
					a.mainWindow.Close()
				}
			},
			a.mainWindow)
		return
	}

	a.connected = true
}

func (a *App) showEmojiPicker() {
	tabs := container.NewAppTabs()

	for category, group := range emojiMap {
		buttons := []fyne.CanvasObject{}
		for _, emoji := range group {
			e := emoji
			btn := widget.NewButton(e, func() {
				a.input.SetText(a.input.Text + e)
			})
			buttons = append(buttons, btn)
		}
		grid := container.NewGridWithColumns(8, buttons...)
		scroll := container.NewVScroll(grid)
		tabs.Append(container.NewTabItem(category, scroll))
	}

	dialog.ShowCustom("Choose Emoji", "Close", tabs, a.mainWindow)
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
		rawUsers := strings.Split(users, ",")
		a.users = make([]string, len(rawUsers))
		for i, u := range rawUsers {
			u = strings.TrimSpace(u)
			if u == a.client.GetUsername() {
				a.users[i] = u + " (you)"
			} else {
				a.users[i] = u
			}
		}
		a.userList.Refresh()
		return
	}

	// Handle public file transfers
	if idx := strings.Index(msg, "[FILE]"); idx != -1 {
		s := strings.TrimSpace(msg[idx+len("[FILE]"):])
		parts := strings.SplitN(s, ":", 2)
		if len(parts) == 2 {
			from := strings.TrimSpace(parts[0])
			filename := strings.TrimSpace(parts[1])

			a.addFileMessage(from, filename, false, "")
			return
		}
	}

	// Handle private file transfers
	if idx := strings.Index(msg, "[PRIVATE FILE]"); idx != -1 {
		s := strings.TrimSpace(msg[idx+len("[PRIVATE FILE]"):])
		if strings.Contains(s, " sent you: ") {
			parts := strings.SplitN(s, " sent you: ", 2)
			if len(parts) == 2 {
				from := strings.TrimSpace(parts[0])
				filename := strings.TrimSpace(parts[1])

				a.addFileMessage(from, filename, true, from)
				return
			}
		}
	}

	// Regular text messages
	a.addTextMessage(msg)
}

func (a *App) addFileMessage(from, filename string, isPrivate bool, sender string) {
	// Simple file message with download button
	var header string
	if isPrivate {
		header = fmt.Sprintf("📁 Private file from %s: %s", from, filename)
	} else {
		header = fmt.Sprintf("📁 %s shared: %s", from, filename)
	}

	headerLabel := widget.NewLabel(header)
	headerLabel.Wrapping = fyne.TextWrapWord

	var downloadBtn *widget.Button
	if isPrivate {
		downloadBtn = widget.NewButton("⬇ Download File", func() {
			a.downloadPrivateFile(filename, sender)
		})
	} else {
		downloadBtn = widget.NewButton("⬇ Download File", func() {
			a.downloadFile(filename)
		})
	}
	downloadBtn.Importance = widget.HighImportance

	fileBox := container.NewVBox(
		headerLabel,
		downloadBtn,
		widget.NewSeparator(),
	)

	a.messageList.Add(fileBox)
	a.messagesScroll.Refresh()
	a.messagesScroll.ScrollToBottom()
}

func (a *App) addTextMessage(msg string) {
	// Determine message color based on type
	displayMsg := msg
	prefix := a.client.GetUsername() + ":"

	var msgColor color.Color

	if strings.HasPrefix(msg, prefix) {
		displayMsg = strings.Replace(msg, prefix, prefix+" (you)", 1)
		msgColor = color.NRGBA{R: 0, G: 100, B: 0, A: 255} // Dark green for own messages
	} else if strings.HasPrefix(msg, "(System)") {
		msgColor = color.NRGBA{R: 100, G: 100, B: 100, A: 255} // Gray for system
	} else if strings.HasPrefix(msg, "(Global)") {
		msgColor = color.NRGBA{R: 0, G: 0, B: 200, A: 255} // Blue for global
	} else if strings.HasPrefix(msg, "(Private)") {
		msgColor = color.NRGBA{R: 150, G: 0, B: 150, A: 255} // Purple for private
	} else {
		msgColor = color.Black // Black for normal messages
	}

	// Create colored text
	msgText := canvas.NewText(displayMsg, msgColor)
	msgText.TextSize = 14
	msgText.Alignment = fyne.TextAlignLeading

	a.messageList.Add(msgText)
	a.messagesScroll.Refresh()
	a.messagesScroll.ScrollToBottom()
}

func (a *App) downloadFile(filename string) {
	go func() {
		if err := a.client.RequestFile(filename); err != nil {
			dialog.ShowError(fmt.Errorf("failed to request file: %v", err), a.mainWindow)
			return
		}
		dialog.ShowInformation("Download", fmt.Sprintf("Downloading %s...", filename), a.mainWindow)
	}()
}

func (a *App) downloadPrivateFile(filename, sender string) {
	go func() {
		if err := a.client.RequestPrivateFile(filename, sender); err != nil {
			dialog.ShowError(fmt.Errorf("failed to request private file: %v", err), a.mainWindow)
			return
		}
		dialog.ShowInformation("Download", fmt.Sprintf("Downloading %s from %s...", filename, sender), a.mainWindow)
	}()
}

func (a *App) sendMessage(content string) {
	var err error
	raw := content
	if strings.TrimSpace(raw) == "" {
		return
	}

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

func (a *App) reopenLogin() {
	go func() {
		time.Sleep(2000 * time.Millisecond)
		a.showLoginDialog()
	}()
}

func (a *App) showFilePicker() {
	options := []string{"Send to Everyone (Public)", "Send to One Person (Private)"}
	selected := widget.NewSelect(options, nil)
	selected.SetSelected("Send to Everyone (Public)")

	recipient := widget.NewEntry()
	recipient.SetPlaceHolder("Enter username")
	recipient.Disable()

	// Enable Enter key to proceed
	recipient.OnSubmitted = func(text string) {
		if selected.Selected == "Send to One Person (Private)" && text != "" {
			a.proceedWithFilePicker(selected.Selected, text)
		}
	}

	selected.OnChanged = func(value string) {
		if value == "Send to One Person (Private)" {
			recipient.Enable()
		} else {
			recipient.Disable()
		}
	}

	content := container.NewVBox(
		widget.NewLabel("Choose who can see this file:"),
		selected,
		widget.NewLabel("Recipient username (for private only):"),
		recipient,
	)

	dialog.ShowCustomConfirm(
		"Send File",
		"Choose File",
		"Cancel",
		content,
		func(confirm bool) {
			if !confirm {
				return
			}

			toUser := strings.TrimSpace(recipient.Text)
			a.proceedWithFilePicker(selected.Selected, toUser)
		},
		a.mainWindow,
	)
}

func (a *App) proceedWithFilePicker(selectedOption, toUser string) {
	isPrivate := selectedOption == "Send to One Person (Private)"

	if isPrivate {
		if toUser == "" {
			dialog.ShowError(fmt.Errorf("please enter recipient username"), a.mainWindow)
			return
		}
		if !a.client.UserExists(toUser) {
			dialog.ShowError(fmt.Errorf("user '%s' not found", toUser), a.mainWindow)
			return
		}
	}

	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, a.mainWindow)
			return
		}
		if reader == nil {
			return
		}
		filePath := reader.URI().Path()
		reader.Close()

		go func() {
			statusLabel := widget.NewLabel(fmt.Sprintf("⏳ Uploading %s...", filepath.Base(filePath)))
			a.messageList.Add(statusLabel)
			a.messageList.Refresh()

			var sendErr error
			if isPrivate {
				sendErr = a.client.SendPrivateFile(filePath, toUser)
			} else {
				sendErr = a.client.SendFile(filePath)
			}

			if sendErr != nil {
				dialog.ShowError(fmt.Errorf("failed to send file: %v", sendErr), a.mainWindow)
				return
			}

			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "File Sent",
				Content: fmt.Sprintf("%s sent successfully", filepath.Base(filePath)),
			})

			a.messageList.Refresh()
		}()
	}, a.mainWindow)
}
