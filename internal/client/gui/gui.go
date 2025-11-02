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
	currentMsg     string // Keep track of the current message context
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
	messageList := container.NewVBox()
	a.messageList = messageList
	a.messagesScroll = container.NewVScroll(messageList)
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
	sendFileBtn := widget.NewButtonWithIcon("File", theme.FileIcon(), func() {
		a.showFilePicker()
	})

	emojiBtn := widget.NewButton("☺", a.showEmojiPicker)

	inputBox := container.NewBorder(
		nil, nil,
		container.NewHBox(emojiBtn, sendFileBtn),
		sendBtn,
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
		if !connect {
			a.mainWindow.Close()
			return
		}

		if username.Text == "" {
			dialog.ShowInformation("Invalid Input", "Please enter a username.", a.mainWindow)
			a.reopenLogin()
			return
		}

		// Try to login and connect
		if err := a.client.Login(username.Text); err != nil {
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
		rawUsers := strings.Split(users, ",")
		a.users = make([]string, len(rawUsers))
		for i, u := range rawUsers {
			u = strings.TrimSpace(u)
			if u == a.client.GetUsername() {
				a.users[i] = u + " (me)"
			} else {
				a.users[i] = u
			}
		}
		a.userList.Refresh()
		return
	}

	if idx := strings.Index(msg, "[FILE]"); idx != -1 {
		s := strings.TrimSpace(msg[idx+len("[FILE]"):])
		parts := strings.SplitN(s, ":", 2)
		if len(parts) == 2 {
			from := strings.TrimSpace(parts[0])
			filename := strings.TrimSpace(parts[1])
			btn := widget.NewButton(fmt.Sprintf("⬇ Download %s", filename), func() {
				a.downloadFile(filename)
			})
			box := container.NewVBox(
				widget.NewLabel(fmt.Sprintf("%s sent a file:", from)),
				btn,
			)
			a.messageList.Add(box)
			a.messagesScroll.Refresh()
			a.messagesScroll.ScrollToBottom()
			return
		}
	}

	if idx := strings.Index(msg, "[PRIVATE FILE]"); idx != -1 {
		s := strings.TrimSpace(msg[idx+len("[PRIVATE FILE]"):])
		if strings.Contains(s, " sent you: ") {
			parts := strings.SplitN(s, " sent you: ", 2)
			if len(parts) == 2 {
				from := strings.TrimSpace(parts[0])
				filename := strings.TrimSpace(parts[1])

				btn := widget.NewButton(fmt.Sprintf("Download %s", filename), func() {
					a.downloadPrivateFile(filename, from)
				})

				box := container.NewVBox(
					widget.NewLabel(fmt.Sprintf("%s sent you a private file:", from)),
					btn,
				)
				box.Objects[0].(*widget.Label).Importance = widget.HighImportance

				a.messageList.Add(box)
				a.messagesScroll.Refresh()
				a.messagesScroll.ScrollToBottom()
				return
			}
		}
	}

	var colorName fyne.ThemeColorName

	switch {
	case strings.HasPrefix(msg, "(System)"):
		colorName = theme.ColorNamePlaceHolder
	case strings.HasPrefix(msg, "(Global)"):
		colorName = theme.ColorNameWarning
	case strings.HasPrefix(msg, "(Private)"):
		colorName = theme.ColorNamePrimary
	default:
		colorName = theme.ColorNameForeground
	}

	displayMsg := msg
	prefix := a.client.GetUsername() + ":"
	if strings.HasPrefix(msg, prefix) {
		displayMsg = strings.Replace(msg, prefix, prefix+" (me):", 1)
	}

	segment := &widget.TextSegment{
		Style: widget.RichTextStyle{
			ColorName: colorName,
			Inline:    true,
		},
		Text: displayMsg + "\n",
	}

	msgRichText := widget.NewRichText(segment)
	msgRichText.Wrapping = fyne.TextWrapWord

	a.messageList.Add(msgRichText)
	a.messagesScroll.Refresh()
	a.messagesScroll.ScrollToBottom()
}

func (a *App) downloadFile(filename string) {
	go func() {
		if err := a.client.RequestFile(filename); err != nil {
			dialog.ShowError(fmt.Errorf("failed to request file: %v", err), a.mainWindow)
			return
		}
		dialog.ShowInformation("Downloading", fmt.Sprintf("Downloading %s...", filename), a.mainWindow)
	}()
}

func (a *App) downloadPrivateFile(filename, sender string) {
	go func() {
		if err := a.client.RequestPrivateFile(filename, sender); err != nil {
			dialog.ShowError(fmt.Errorf("failed to request private file: %v", err), a.mainWindow)
			return
		}
		dialog.ShowInformation("Downloading", fmt.Sprintf("Downloading private file %s from %s...", filename, sender), a.mainWindow)
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
	options := []string{"Public", "Private"}
	selected := widget.NewSelect(options, nil)
	selected.SetSelected("Public")

	recipient := widget.NewEntry()
	recipient.SetPlaceHolder("Enter recipient username (for Private only)")
	recipient.Disable()

	selected.OnChanged = func(value string) {
		if value == "Private" {
			recipient.Enable()
		} else {
			recipient.Disable()
		}
	}

	dialog.ShowCustomConfirm(
		"Send File",
		"Next",
		"Cancel",
		container.NewVBox(
			widget.NewLabel("Choose how to send your file:"),
			selected,
			recipient,
		),
		func(confirm bool) {
			if !confirm {
				return
			}

			toUser := strings.TrimSpace(recipient.Text)

			if selected.Selected == "Private" {
				if toUser == "" {
					dialog.ShowError(fmt.Errorf("recipient username is required"), a.mainWindow)
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
					uploadMsg := widget.NewLabel(fmt.Sprintf("Uploading file: %s...", filepath.Base(filePath)))
					a.messageList.Add(uploadMsg)
					a.messageList.Refresh()

					var sendErr error
					if selected.Selected == "Private" {
						toUser := strings.TrimSpace(recipient.Text)
						if toUser == "" {
							dialog.ShowError(fmt.Errorf("recipient username is required for private file"), a.mainWindow)
							return
						}
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
						Content: fmt.Sprintf("%s uploaded successfully", filepath.Base(filePath)),
					})

					a.messageList.Refresh()
				}()
			}, a.mainWindow)
		},
		a.mainWindow,
	)
}
