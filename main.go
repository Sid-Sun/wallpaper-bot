package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var photoIDMap = make(map[string]string)
var photoList []string

func main() {
	// Read available wallpapers list
	refreshWallpaperList()

	// Read Photo IDs from JSON
	readPhotoIDs()

	// Populate photoList with already uploaded file IDs
	populateWallpapersFromIDs()

	// Actual BOT stuff
	bot, err := tgbotapi.NewBotAPI(os.Getenv("API_TOKEN"))
	if err != nil {
		panic(err.Error())
	}

	bot.Debug = false
	fmt.Printf("Hello, I am %s\n", bot.Self.FirstName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		panic(err.Error())
	}

	adminChatID, err := strconv.ParseInt(os.Getenv("ADMIN_CHAT_ID"), 10, 64)
	if err != nil {
		panic(err.Error())
	}

	getUpdates(bot, updates, adminChatID)
}

func getUpdates(bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel, adminChatID int64) {
	for update := range updates {
		if update.Message == nil {
			continue
		}
		go handleUpdate(bot, update, adminChatID)
	}
}

func handleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update, adminChatID int64) {
	if update.Message.Chat.IsPrivate() {
		if messageSlice := strings.Fields(update.Message.Text); len(messageSlice) > 0 {
			switch messageSlice[0] {
			case "/wallpaper", "/wallpapers":
				var wg sync.WaitGroup
				var numberOfWallpapers int

				if len(messageSlice) == 1 {
					numberOfWallpapers = 1
				} else {
					var err error
					numberOfWallpapers, err = strconv.Atoi(messageSlice[1])
					if err != nil {
						numberOfWallpapers = 1
					}
					if numberOfWallpapers > 10 && update.Message.Chat.ID != adminChatID {
						numberOfWallpapers = 10
					} else if numberOfWallpapers > len(photoList) {
						// Cap number of wallpapers to length of photolist
						// If less than 10 walls are requested and even fewer are available
						numberOfWallpapers = len(photoList)
					}
				}

				var wallpapersSent []int
				for i := 0; i < numberOfWallpapers; i++ {
					rand.Seed(time.Now().UnixNano())
					randomInt := rand.Intn(len(photoList))
					temp := append(wallpapersSent, randomInt)
					if hasDuplicates(temp) {
						i--
						continue
					}
					wallpapersSent = temp
					wg.Add(1)
					go sendWallpaper(bot, update.Message.Chat.ID, &wg, photoIDMap, photoList, randomInt)
				}

				wg.Wait()
			case "/start":
				helloMessage := tgbotapi.NewMessage(update.Message.Chat.ID, "Hello! I am Wallpaper Bot, to request one wallpaper, send /wallpaper, to get multiple, send /wallpapers <count (limited to 10)>!")
				helloMessage.ReplyToMessageID = update.Message.MessageID
				if _, err := bot.Send(helloMessage); err != nil {
					handleError(bot, err, update.Message.Chat.ID)
				}
			case "/refresh":
				if update.Message.Chat.ID == adminChatID {
					refreshWallpaperList()
					readPhotoIDs()
					populateWallpapersFromIDs()
					sendToAdmin(bot, "Refreshed!")
				}
			case "/all":
				if update.Message.Chat.ID == adminChatID {
					var wg sync.WaitGroup
					for i := 0; i < len(photoList); i++ {
						wg.Add(1)
						go sendWallpaper(bot, update.Message.Chat.ID, &wg, photoIDMap, photoList, i)
					}
					wg.Wait()
					sendToAdmin(bot, "That would be all!")
				}
			}
		} else if update.Message.Document.FileID != "" {
			if update.Message.Chat.ID == adminChatID {
				fileName := update.Message.Document.FileName
				fileID := update.Message.Document.FileID
				if photoIDMap[fileName] == "" {
					photoIDMap[fileName] = fileID
					data, err := json.Marshal(photoIDMap)
					if err != nil {
						sendToAdmin(bot, err.Error())
					}
					err = writeContentToFile(os.Getenv("WALLPAPERS_DIR")+"/photoIDs.json", data)
					if err != nil {
						sendToAdmin(bot, err.Error())
					}
					populateWallpapersFromIDs()
					sendToAdmin(bot, "Added: "+fileName+" and refreshed the IDs")
				} else {
					sendToAdmin(bot, "Wallpaper already in database")
				}
			}
		}
	}
}

func sendWallpaper(bot *tgbotapi.BotAPI, chatID int64, wg *sync.WaitGroup, photoIDMap map[string]string, photoList []string, randomInt int) {
	randomPhotoName := photoList[randomInt]
	if photoIDMap[randomPhotoName] == "" {
		filePath := os.Getenv("WALLPAPERS_DIR") + "/" + randomPhotoName
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			handleError(bot, err, chatID)
		}
		file := tgbotapi.FileBytes{
			Name:  randomPhotoName,
			Bytes: data,
		}
		document := tgbotapi.NewDocumentUpload(chatID, file)
		// photo := tgbotapi.NewPhotoUpload(update.Message.Chat.ID, file)
		document.Caption = randomPhotoName
		res, err := bot.Send(document)
		if err != nil {
			handleError(bot, err, chatID)
		}
		DocumentID := res.Document.FileID
		// PhotoID := (*res.Photo)[len(*res.Photo)-1].FileID
		photoIDMap[randomPhotoName] = DocumentID
		data, err = json.Marshal(photoIDMap)
		if err != nil {
			sendToAdmin(bot, err.Error())
		}
		err = writeContentToFile(os.Getenv("WALLPAPERS_DIR")+"/photoIDs.json", data)
		if err != nil {
			sendToAdmin(bot, err.Error())
		}
		// Once uploaded, delete the file
		err = deleteFile(filePath)
		if err != nil {
			sendToAdmin(bot, err.Error())
		}
	} else {
		// photo := tgbotapi.NewPhotoShare(update.Message.Chat.ID, photoIDMap[randomPhotoName])
		document := tgbotapi.NewDocumentShare(chatID, photoIDMap[randomPhotoName])
		document.Caption = randomPhotoName
		_, err := bot.Send(document)
		if err != nil {
			handleError(bot, err, chatID)
		}
	}
	wg.Done()
}
