package bot

import (
	"fmt"
	"log"
	//"runtime"

	//"strings"
	"sync"

	tgbotapi "github.com/kirugan/telegram-bot-api/v5"
	config "github.com/nk-BH-D/BH_Transcribator/tg/internal/config"
	db "github.com/nk-BH-D/BH_Transcribator/tg/internal/db"
)

var (
	userStates = make(map[int64]map[string]string)
	//forDelMessage = make(map[int64]int)
	us_db  *db.PostgresUs
	err_db *db.PostgresErrors
	uSM    sync.RWMutex
	//fDM           sync.RWMutex
	conf *config.Config
)

// func fot syncing userState map
func setUserState(chatID int64, state map[string]string) {
	uSM.Lock()
	defer uSM.Unlock()
	userStates[chatID] = state
}
func getUserState(chatID int64) (map[string]string, bool) {
	uSM.RLock()
	defer uSM.RUnlock()
	state, ok := userStates[chatID]
	return state, ok
}
func deleteUserState(chatID int64) {
	uSM.Lock()
	defer uSM.Unlock()
	delete(userStates, chatID)
}

// funcs for synciong forDelMessage map
//func setDelID(chatID int64, delID int) {
//	fDM.Lock()
//	defer fDM.Unlock()
//	forDelMessage[chatID] = delID
//}
//func getDelID(chatID int64) (int, bool) {
//	fDM.RLock()
//	defer fDM.RUnlock()
//	delID, ok := forDelMessage[chatID]
//	return delID, ok
//}
//func deleteDelID(chatID int64) {
//	fDM.Lock()
//	defer fDM.Unlock()
//	delete(forDelMessage, chatID)
//}

func Init(cfg *config.Config, pg_us *db.PostgresUs, pg_err *db.PostgresErrors) {
	conf = cfg
	us_db = pg_us
	err_db = pg_err
}

func sendErrorMessage(bot *tgbotapi.BotAPI, chatID int64, message, buttom string) {
	msg := tgbotapi.NewMessage(chatID, message)
	switch buttom {
	case "menu":
		msg.ReplyMarkup = createMenuKeybord()
	case "":
		msg.ReplyMarkup = nil
	default:
		log.Printf("unknown button type: %s", buttom)
	}

	deleteUserState(chatID)
	_, sendErr := bot.Send(msg)
	if sendErr != nil {
		log.Printf("errro sending message: %s, %v", message, sendErr)
	}
}

func sendSuccessMessage(bot *tgbotapi.BotAPI, chatID int64, message, buttom string) {
	message_list := []string{}
	if len(message) > 4096 {
		runes := []rune(message)
		for i := 0; i < len(runes); i += 4096 {
			end := i + 4096

			if end > len(runes) {
				end = len(runes) // хвост
			}

			message_list = append(message_list, string(runes[i:end]))
		}
		assistedSendSuccessMessage(bot, chatID, message_list, buttom)
	} else {
		message_list = append(message_list, message)
		assistedSendSuccessMessage(bot, chatID, message_list, buttom)
	}
}

func assistedSendSuccessMessage(bot *tgbotapi.BotAPI, chatID int64, message_list []string, buttom string) {
	for _, message := range message_list {
		msg := tgbotapi.NewMessage(chatID, message)

		switch buttom {
		case "ord":
			msg.ReplyMarkup = createOrderKetbord()
		case "reg":
			msg.ReplyMarkup = createRegistrKeybord()
		case "menu":
			msg.ReplyMarkup = createMenuKeybord()
		case "command":
			msg.ReplyMarkup = createCommandKeybord()
		case "menu_for_error":
			msg.ReplyMarkup = createErrorWorkKeybord()
		case "":
			msg.ReplyMarkup = nil
		default:
			log.Printf("unknown button type: %s", buttom)
			sendErrorMessage(bot, chatID, "Internal service error1: повторите попытку позже", "menu")
			return
		}
		_, sendErr := bot.Send(msg)
		if sendErr != nil {
			log.Printf("error sending message: %s, %v", message, sendErr)
			sendErrorMessage(bot, chatID, fmt.Sprintf("произошла ошибка при отправке сообщения: %v", sendErr), "menu")
			return
		}
		//setDelID(chatID, sendMsg.MessageID)
	}
}

// удаление сообщений бота
//func delSendedMessage(bot *tgbotapi.BotAPI, chatID int64, delID int) {
//	delBot := tgbotapi.NewDeleteMessage(chatID, delID)
//	_, err := bot.Request(delBot)
//	if err != nil {
//		log.Printf("error when deleting message from bot: %v", err)
//		return
//	}
//}

// спасатель
func runSafely(bot *tgbotapi.BotAPI, chatID int64, name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in %s: %v", name, r)
				sendErrorMessage(bot, chatID, "Что-то пошло не так, попробуйте снова", "")
			}
		}()

		fn()
	}()
}

func HandleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userID := message.From.ID

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Intercepted panic: %v", r)
			sendErrorMessage(bot, chatID, "Что-то пошло не так, попробуйте снова", "")
			return
		}
	}()

	if message.Text != "" && message.Text == "/start" {
		deleteUserState(chatID)
		runSafely(bot, chatID, "start", func() {
			Register(bot, userID, chatID)
		})
		//log.Printf("Активныч горутин: %d\n", runtime.NumGoroutine())
		return
	}

	if state, ok := getUserState(chatID); ok {
		for key := range state {
			switch key {
			case "order":
				text := message.Text
				runSafely(bot, chatID, "order", func() {
					Order(bot, userID, chatID, text)
				})
				return
			case "transcride":
				runSafely(bot, chatID, "transcride", func() {
					handlerTranscrib(bot, userID, chatID, message)
				})
				return
			case "fix_error":
				runSafely(bot, chatID, "fix_error", func() {
					FixError(bot, chatID, message.Text)
				})
				return
			default:
				log.Printf("неизвестная команда: %s", key)
				sendErrorMessage(bot, chatID, "Неизвестная команда", "menu")
				return

			}
		}
	}

}

func HandleCallback(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery) {
	chatID := callbackQuery.Message.Chat.ID
	userID := callbackQuery.From.ID
	data := callbackQuery.Data

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Перехвачена паника: %v", r)
			sendErrorMessage(bot, chatID, "Что-то пошло не так, попробуйте снова", "")
			return
		}
	}()

	switch data {
	case "show_menu":
		deleteUserState(chatID)
		sendSuccessMessage(bot, chatID, "Выберете действие", "command")
		return
	case "accept":
		runSafely(bot, chatID, "accept", func() {
			AcceptHandler(bot, userID, chatID)
		})
		return
	case "check subscription":
		runSafely(bot, chatID, "chek sub", func() {
			ChekSub(bot, userID, chatID)
		})
		return
	case "good", "bad":
		var (
			like     int = 0
			dis_like int = 0
			name     string
		)
		if data == "good" {
			like = 1
			name = "good"
		} else {
			dis_like = 1
			name = "bad"
		}
		runSafely(bot, chatID, name, func() {
			Mark(bot, userID, chatID, 0, like, dis_like)
		})
		return
	case "order":
		setUserState(chatID, map[string]string{"order": ""})
		sendSuccessMessage(bot, chatID, "Отправьте текст c комментарием ошибки", "")
		return
	case "stat":
		go Stat(bot, userID, chatID)
		return
	case "transcride":
		setUserState(chatID, map[string]string{"transcride": ""})
		sendSuccessMessage(bot, chatID, "Отправьте ваше аудио", "")
		return
	case "fix_error":
		setUserState(chatID, map[string]string{"fix_error": ""})
		sendSuccessMessage(bot, chatID, "Отправте errorID решённой(ых) проблема через пробел.", "")
		return
	default:
		log.Printf("неизвестная команда: %s", data)
		sendErrorMessage(bot, chatID, "Неизвестная команда", "menu")
		return

	}

}
