package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	storageengine "github.com/fitant/storage-engine-go/storageengine"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func strSliceHasDuplicates(x []string) bool {
	encountered := map[string]bool{}
	for _, val := range x {
		if encountered[val] {
			return true
		}
		encountered[val] = true
	}
	return false
}

func loadDataFromSE() {
	listMutex = new(sync.Mutex)
	if seClient, err := storageengine.NewClientConfig(http.DefaultClient, os.Getenv("SE_URL")); err != nil {
		panic(err)
	} else {
		photoListObject, err = storageengine.NewObject(seClient)
		if err != nil {
			panic(err)
		}
	}

	photoListObject.SetID(os.Getenv("SE_OBJ_ID"))
	photoListObject.SetPassword(os.Getenv("SE_OBJ_PASS"))
	if err := photoListObject.Refresh(); err != nil {
		log.Print(err)
	}
	if photoListObject.GetData() == "" {
		log.Print("fetch from SE failed presumably due to 404 - doing a fresh start")
	} else {
		err := json.Unmarshal([]byte(photoListObject.GetData()), &photoIDMap)
		if err != nil {
			log.Print(err)
		}
	}
}

func hasDuplicates(x []int) bool {
	encountered := map[int]bool{}
	for _, val := range x {
		if encountered[val] {
			return true
		}
		encountered[val] = true
	}
	return false
}

func populateWallpapersFromIDs() {
	// Populate photoList with already uploaded files
	// This way we can delete the files once uploaded
	// And be able to reuse them too
	for val := range photoIDMap {
		// Don't add photoIDs to list
		if val != "photoIDs.json" {
			t := append(photoList, val)
			if !strSliceHasDuplicates(t) {
				photoList = t
			}
		}
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
