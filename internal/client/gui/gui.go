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

// Yahoo Messenger inspired colors
var (
	yahooYellow    = color.NRGBA{R: 255, G: 204, B: 0, A: 255}   // Yahoo yellow
	yahooBlue      = color.NRGBA{R: 64, G: 103, B: 178, A: 255}  // Yahoo blue
	lightGray      = color.NRGBA{R: 240, G: 240, B: 240, A: 255} // Light gray background
	borderGray     = color.NRGBA{R: 180, G: 180, B: 180, A: 255} // Border
	userPanelColor = color.NRGBA{R: 230, G: 235, B: 250, A: 255} // Light blue for users
)

type App struct {
	client         *client.Client
	mainWindow     fyne.Window
	messagesScroll *container.Scroll
	userList       *widget.List
	input          *customEntry
	users          []string
	connected      bool
	msgHistory     []string
	incoming       chan string
	messageList    *fyne.Container
	currentMsg     string
}

// Custom entry widget to handle Enter key properly
type customEntry struct {
	widget.Entry
	onEnterPressed func()
}

func newCustomEntry() *customEntry {
	entry := &customEntry{}
	entry.ExtendBaseWidget(entry)
	entry.MultiLine = true
	entry.Wrapping = fyne.TextWrapWord
	return entry
}

func (e *customEntry) TypedKey(key *fyne.KeyEvent) {
	if key.Name == fyne.KeyReturn && e.onEnterPressed != nil {
		e.onEnterPressed()
		return
	}
	e.Entry.TypedKey(key)
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

func createYahooBox(content fyne.CanvasObject, title string, bgColor color.Color) *fyne.Container {
	bg := canvas.NewRectangle(bgColor)

	border := canvas.NewRectangle(color.Transparent)
	border.StrokeWidth = 1
	border.StrokeColor = borderGray
	border.FillColor = color.Transparent

	if title != "" {
		// Yahoo-style header with gradient-like effect
		headerBg := canvas.NewRectangle(yahooBlue)

		titleLabel := canvas.NewText(title, color.White)
		titleLabel.TextSize = 13
		titleLabel.TextStyle = fyne.TextStyle{Bold: true}

		header := container.NewStack(
			headerBg,
			container.NewPadded(titleLabel),
		)

		return container.NewStack(
			bg,
			border,
			container.NewBorder(header, nil, nil, nil, container.NewPadded(content)),
		)
	}

	return container.NewStack(bg, border, container.NewPadded(content))
}

func (a *App) Run() error {
	fyneApp := app.NewWithID("com.chatroom.app")
	fyneApp.Settings().SetTheme(theme.LightTheme())
	a.mainWindow = fyneApp.NewWindow("Talkie Messenger")
	a.mainWindow.Resize(fyne.NewSize(750, 550))
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
	messageList := container.NewVBox()
	a.messageList = messageList
	a.messagesScroll = container.NewVScroll(messageList)

	messagesContainer := createYahooBox(a.messagesScroll, "Conversation", color.White)

	a.userList = widget.NewList(
		func() int { return len(a.users) },
		func() fyne.CanvasObject {
			icon := canvas.NewCircle(color.NRGBA{R: 0, G: 200, B: 0, A: 255})
			icon.Resize(fyne.NewSize(8, 8))
			label := widget.NewLabel("Template User")
			return container.NewHBox(icon, label)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			box := obj.(*fyne.Container)
			icon := box.Objects[0].(*canvas.Circle)
			label := box.Objects[1].(*widget.Label)

			username := a.users[id]

			icon.FillColor = color.NRGBA{R: 0, G: 200, B: 0, A: 255}

			if strings.Contains(username, "(you)") {
				label.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				label.TextStyle = fyne.TextStyle{}
			}
			label.SetText(username)
		},
	)

	userScroll := container.NewScroll(a.userList)
	userContainer := createYahooBox(userScroll, "Buddies Online", userPanelColor)

	a.input = newCustomEntry()
	a.input.SetPlaceHolder("Type a message here...")

	a.input.onEnterPressed = func() {
		content := a.input.Text
		content = ConvertEmojis(content)
		a.sendMessage(content)
	}

	emojiBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), a.showEmojiPicker)
	emojiBtn.SetText("")

	fileBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		a.showFilePicker()
	})
	fileBtn.SetText("")

	sendBtn := widget.NewButton("Send", func() {
		content := a.input.Text
		content = ConvertEmojis(content)
		a.sendMessage(content)
	})
	sendBtn.Importance = widget.HighImportance

	leftButtons := container.NewHBox(emojiBtn, fileBtn)

	inputBox := container.NewBorder(
		nil, nil,
		leftButtons, // Left side
		sendBtn,     // Right side
		a.input,
	)

	inputContainer := createYahooBox(inputBox, "", lightGray)

	// Main layout
	chatArea := container.NewBorder(
		nil, inputContainer, nil, nil,
		messagesContainer,
	)

	split := container.NewHSplit(chatArea, userContainer)
	split.SetOffset(0.72)

	// Yahoo-style top bar
	topBar := canvas.NewRectangle(yahooYellow)
	topBar.SetMinSize(fyne.NewSize(0, 3))

	mainContent := container.NewBorder(
		topBar, nil, nil, nil,
		container.NewPadded(split),
	)

	a.mainWindow.SetContent(mainContent)
}

func (a *App) showLoginDialog() {
	username := widget.NewEntry()
	username.SetPlaceHolder("Enter your username")

	welcomeText := widget.NewLabel("Welcome to Talkie Messenger")
	welcomeText.TextStyle = fyne.TextStyle{Bold: true}
	welcomeText.Alignment = fyne.TextAlignCenter

	content := container.NewVBox(
		welcomeText,
		widget.NewSeparator(),
		widget.NewLabel("Username:"),
		username,
	)

	var dlg dialog.Dialog

	connectFunc := func() {
		if strings.TrimSpace(username.Text) == "" {
			dialog.ShowInformation("Invalid Input", "Please enter a username.", a.mainWindow)
			go func() {
				time.Sleep(500 * time.Millisecond)
				a.showLoginDialog()
			}()
			return
		}

		if err := a.client.Login(username.Text); err != nil {
			dialog.ShowError(err, a.mainWindow)
			go func() {
				time.Sleep(500 * time.Millisecond)
				a.showLoginDialog()
			}()
			return
		}

		if err := a.client.Connect(":9000"); err != nil {
			if strings.Contains(err.Error(), "username") {
				dialog.ShowError(fmt.Errorf("login failed: %s", err.Error()), a.mainWindow)
				go func() {
					time.Sleep(500 * time.Millisecond)
					a.showLoginDialog()
				}()
				return
			}

			dialog.ShowConfirm("Connection failed",
				"Cannot connect to server.\nDo you want to retry?",
				func(confirm bool) {
					if confirm {
						a.showLoginDialog()
					} else {
						a.mainWindow.Close()
					}
				},
				a.mainWindow)
			return
		}

		a.connected = true
	}

	dlg = dialog.NewCustomConfirm("Login", "Connect", "Exit", content, func(connect bool) {
		if !connect {
			a.mainWindow.Close()
			return
		}
		connectFunc()
	}, a.mainWindow)

	dlg.Show()
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

	d := dialog.NewCustom("Choose Emoji", "Close", tabs, a.mainWindow)
	d.Resize(fyne.NewSize(450, 350))
	d.Show()
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
	var header string
	var headerColor color.Color

	if isPrivate {
		header = fmt.Sprintf("*** %s is sending you a file: %s", from, filename)
		headerColor = color.NRGBA{R: 150, G: 0, B: 150, A: 255}
	} else {
		header = fmt.Sprintf("*** %s is sharing a file: %s", from, filename)
		headerColor = yahooBlue
	}

	headerText := canvas.NewText(header, headerColor)
	headerText.TextSize = 12
	headerText.TextStyle = fyne.TextStyle{Bold: true}

	var downloadBtn *widget.Button
	if isPrivate {
		downloadBtn = widget.NewButton("Accept and Download", func() {
			a.downloadPrivateFile(filename, sender)
		})
	} else {
		downloadBtn = widget.NewButton("Download", func() {
			a.downloadFile(filename)
		})
	}
	downloadBtn.Importance = widget.MediumImportance

	fileBox := container.NewVBox(
		headerText,
		downloadBtn,
		widget.NewSeparator(),
	)

	a.messageList.Add(fileBox)
	a.messagesScroll.Refresh()
	a.messagesScroll.ScrollToBottom()
}

func (a *App) addTextMessage(msg string) {
	// Yahoo Messenger style message colors
	displayMsg := msg
	prefix := a.client.GetUsername() + ":"

	var msgColor color.Color

	if strings.HasPrefix(msg, prefix) {
		displayMsg = strings.Replace(msg, prefix, prefix+" (you)", 1)
		msgColor = color.NRGBA{R: 0, G: 100, B: 0, A: 255} // Dark green
	} else if strings.HasPrefix(msg, "(System)") {
		msgColor = color.NRGBA{R: 150, G: 0, B: 0, A: 255} // Red for system
	} else if strings.HasPrefix(msg, "(Global)") {
		msgColor = yahooBlue // Yahoo blue
	} else if strings.HasPrefix(msg, "(Private)") {
		msgColor = color.NRGBA{R: 150, G: 0, B: 150, A: 255} // Purple
	} else {
		msgColor = color.Black
	}

	msgText := canvas.NewText(displayMsg, msgColor)
	msgText.TextSize = 13
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
	options := []string{"Everyone (Public)", "One Person (Private)"}
	selected := widget.NewSelect(options, nil)
	selected.SetSelected("Everyone (Public)")

	recipient := widget.NewEntry()
	recipient.SetPlaceHolder("Enter username")
	recipient.Disable()

	selected.OnChanged = func(value string) {
		if value == "One Person (Private)" {
			recipient.Enable()
		} else {
			recipient.Disable()
		}
	}

	content := container.NewVBox(
		widget.NewLabel("Send file to:"),
		selected,
		widget.NewLabel("Recipient (for private only):"),
		recipient,
	)

	dialog.ShowCustomConfirm(
		"Send File",
		"Browse",
		"Cancel",
		content,
		func(confirm bool) {
			if !confirm {
				return
			}

			toUser := strings.TrimSpace(recipient.Text)
			isPrivate := selected.Selected == "One Person (Private)"

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
					statusText := canvas.NewText(fmt.Sprintf(">>> Uploading %s...", filepath.Base(filePath)), color.NRGBA{R: 100, G: 100, B: 100, A: 255})
					statusText.TextSize = 12
					statusText.TextStyle = fyne.TextStyle{Italic: true}
					a.messageList.Add(statusText)
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
		},
		a.mainWindow,
	)
}
