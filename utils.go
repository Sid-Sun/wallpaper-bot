package main

import (
	"errors"
	"io/ioutil"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func strSliceHasDuplicates(x []string) bool {
	encountered := map[string]bool{}
	for _, val := range x {
		if encountered[val] == true {
			return true
		}
		encountered[val] = true
	}
	return false
}

func hasDuplicates(x []int) bool {
	encountered := map[int]bool{}
	for _, val := range x {
		if encountered[val] == true {
			return true
		}
		encountered[val] = true
	}
	return false
}

func refreshWallpaperList() {
	// Read available wallpapers list
	files, err := ioutil.ReadDir(os.Getenv("WALLPAPERS_DIR"))
	if err != nil {
		panic(err.Error())
	}
	// Empty List first
	photoList = []string{}
	for _, file := range files {
		photoList = append(photoList, file.Name())
	}
}

func handleError(bot *tgbotapi.BotAPI, err error, chatID int64) {
	errorMessage := tgbotapi.NewMessage(chatID, "Could not process request, this incident has been reported")
	_, _ = bot.Send(errorMessage)
	sendToAdmin(bot, err.Error())
}

func sendToAdmin(bot *tgbotapi.BotAPI, message string) {
	adminChatID, _ := strconv.ParseInt(os.Getenv("ADMIN_CHAT_ID"), 10, 64)
	msg := tgbotapi.NewMessage(adminChatID, message)
	_, _ = bot.Send(msg)
}

func writeContentToFile(fileName string, fileContents []byte) error {
	err := ioutil.WriteFile(fileName, fileContents, 0644)
	if err != nil {
		return err
	}
	return nil
}

func readFromFile(filePath string) ([]byte, error) {
	// Check if file exists and if not, print
	if fileExists(filePath) {
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	return nil, errors.New("FILE DOES NOT EXIST")
}

func deleteFile(fileName string) error {
	err := os.Remove(fileName)
	return err
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
