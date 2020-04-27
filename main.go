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

	// Read data from photo IDs JSON
	data, err := readFromFile("photoIDs.json")
	if err != nil {
		// IF the photoIDs.json is not present, try to create one
		if err.Error() == "FILE DOES NOT EXIST" {
			data, err = json.Marshal(photoIDMap)
			if err != nil {
				panic(err.Error())
			}
			err = writeContentToFile("photoIDs.json", data)
			if err != nil {
				panic(err.Error())
			}
		} else {
			panic(err.Error())
		}
	}
	// Unmarshal JSON and store in map
	err = json.Unmarshal(data, &photoIDMap)
	if err != nil {
		panic(err.Error())
	}

	// Populate photoList with already uploaded files
	// This way we can delete the files once uploaded
	// And be able to reuse them too
	for val := range photoIDMap {
		t := append(photoList, val)
		if !strSliceHasDuplicates(t) {
			photoList = t
		}
	}

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
		messageSlice := strings.Fields(update.Message.Text)
		if messageSlice[0] == "/wallpapers" || messageSlice[0] == "/wallpaper" {
			var wg sync.WaitGroup
			var numberOfWallpapers int

			if len(messageSlice) != 1 {
				var err error
				numberOfWallpapers, err = strconv.Atoi(messageSlice[1])
				if err != nil {
					numberOfWallpapers = 1
				}
				if numberOfWallpapers > 10 && update.Message.Chat.ID != adminChatID {
					numberOfWallpapers = 10
				} else if numberOfWallpapers > len(photoList) {
					// Cap number of wallpapers to length of photolist if less than 10 walls are requested and even fewer are available
					numberOfWallpapers = len(photoList)
				}
			} else {
				numberOfWallpapers = 1
			}

			var wallpapersSent []int
			for i := 0; i < numberOfWallpapers; i++ {
				wg.Add(1)
				rand.Seed(time.Now().UnixNano())
				randomInt := rand.Intn(len(photoList))
				temp := append(wallpapersSent, randomInt)
				if hasDuplicates(temp) {
					i--
					continue
				}
				wallpapersSent = temp
				go sendWallpaper(bot, update.Message.Chat.ID, &wg, photoIDMap, photoList, randomInt)
			}

			wg.Wait()
		} else if messageSlice[0] == "/start" {
			helloMessage := tgbotapi.NewMessage(update.Message.Chat.ID, "Hello! I am Wallpaper Bot, to request one wallpaper, send /wallpaper, to get multiple, send /wallpapers <count (limited to 10)>!")
			helloMessage.ReplyToMessageID = update.Message.MessageID
			if _, err := bot.Send(helloMessage); err != nil {
				handleError(bot, err, update.Message.Chat.ID)
			}
		} else if update.Message.Chat.ID == adminChatID {
			if update.Message.Text == "/refresh" {
				refreshWallpaperList()
				sendToAdmin(bot, "Refreshed!")
			} else if update.Message.Text == "/all" {
				var wg sync.WaitGroup
				for i := 0; i < len(photoList); i++ {
					wg.Add(1)
					go sendWallpaper(bot, update.Message.Chat.ID, &wg, photoIDMap, photoList, i)
				}
				wg.Wait()
				sendToAdmin(bot, "That would be all!")
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
		err = writeContentToFile("photoIDs.json", data)
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
