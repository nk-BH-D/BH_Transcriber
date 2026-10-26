package bot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/kirugan/telegram-bot-api/v5"
	config "github.com/nk-BH-D/BH_Transcribator/tg/internal/config"
	db "github.com/nk-BH-D/BH_Transcribator/tg/internal/db"
)

var (
	userStates         = make(map[int64]map[string]string)
	stateMessage       = make(map[int64]int)
	bloсkOrderKeyboard = make(map[int64]string)
	us_db              *db.PostgresUs
	err_db             *db.PostgresErrors
	uSM                sync.RWMutex
	sMM                sync.RWMutex
	bKM                sync.RWMutex
	conf               *config.Config
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
func setStateMessage(chatID int64, messageID int) {
	sMM.Lock()
	defer sMM.Unlock()
	stateMessage[chatID] = messageID
}
func getStateMessage(chatID int64) (int, bool) {
	sMM.RLock()
	defer sMM.RUnlock()
	messageID, ok := stateMessage[chatID]
	return messageID, ok
}
func deleteStateMessage(chatID int64) {
	sMM.Lock()
	defer sMM.Unlock()
	delete(stateMessage, chatID)
}

// funcs for synciong blokOrderKeyboart map
func setBloсkOrderKeyboard(chatID int64, value string) {
	bKM.Lock()
	defer bKM.Unlock()
	bloсkOrderKeyboard[chatID] = value
}
func getBloсkOrderKeyboard(chatID int64) string {
	bKM.RLock()
	defer bKM.RUnlock()
	return bloсkOrderKeyboard[chatID]
}
func deleteBloсkOrderKeyboard(chatID int64) {
	bKM.Lock()
	defer bKM.Unlock()
	delete(bloсkOrderKeyboard, chatID)
}

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
	var (
		fl          bool = false
		globSendMsg tgbotapi.Message
	)

	for i, message := range message_list {
		msg := tgbotapi.NewMessage(chatID, message)
		// добавляем кнопку только к последнему куску сообщения
		if i == len(message_list)-1 {
			switch buttom {
			case "ord":
				msg.ReplyMarkup = createOrderKeybord()
				fl = true
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
		}

		sendMsg, sendErr := bot.Send(msg)
		if sendErr != nil {
			log.Printf("error sending message: %s, %v", message, sendErr)
			sendErrorMessage(bot, chatID, fmt.Sprintf("произошла ошибка при отправке сообщения: %v", sendErr), "menu")
			return
		}
		// так как кнопка только у последнего, сохраняем в map только его id
		if i == len(message_list)-1 {
			globSendMsg = sendMsg
		}
	}
	if fl {
		setStateMessage(chatID, globSendMsg.MessageID)
		setBloсkOrderKeyboard(chatID, "unlock")
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

// функция изменения клавиатуры оценки трансрибации
func editMessageMarkup(bot *tgbotapi.BotAPI, chatID int64, messageID int, newKeyboard tgbotapi.InlineKeyboardMarkup) {
	edit := tgbotapi.NewEditMessageReplyMarkup(
		chatID,
		messageID,
		newKeyboard,
	)

	_, err := bot.Request(edit)
	if err != nil {
		log.Printf("error when edit reply markup: %v", err)
		sendErrorMessage(bot, chatID, "Оценка поставлена", "")
		return
	}
}

// функция валидации пользователлей
func userValid(bot *tgbotapi.BotAPI, userID, chatID int64) bool {
	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()
	accept, _, _, _, _, err := us_db.GetUserData(ctx, userID)
	if err != nil {
		log.Printf("error when valid user: %v", err)
		return false
	}
	if !accept {
		sendErrorMessage(bot, chatID, "Вам отказано в доступе.\nВозможно вы не подписаны на канал проекта или не приняли соглашение", "")
		return false
	}

	var (
		fl bool = false
	)

	member, err := bot.GetChatMember(
		tgbotapi.GetChatMemberConfig{
			ChatID: conf.TG_CHANNEL_ID,
			UserID: userID,
		},
	)
	if err != nil {
		log.Printf("error when chek sub: %v", err)
		sendErrorMessage(bot, chatID, "Internal service error16: поробуйте снова позже", "")
		return false
	} else {
		switch member.Status {
		case "creator", "administrator", "member":
			fl = true
		case "restricted":
			if member.IsMember {
				fl = true
			} else {
				sendErrorMessage(bot, chatID, "Вам отказано в доступе.\nВозможно вы не подписаны на канал проекта или не приняли соглашение", "")
				fl = false
			}
		default:
			sendErrorMessage(bot, chatID, "Вам отказано в доступе.\nВозможно вы не подписаны на канал проекта или не приняли соглашение", "")
			fl = false
		}
	}
	return fl
}

// функции валидации root
func ValidRoot(bot *tgbotapi.BotAPI, userID, chatID int64, login string) bool {
	if userID != conf.ROOT_ID {
		log.Printf("не root работает с admin панелью")
		sendSuccessMessage(
			bot,
			conf.ROOT_ID,
			fmt.Sprintf(
				"Не root работает с admin панелью!!!\n"+
					"Данные нарушителя\n"+
					"UserID: %d\n"+
					"Login: %s",
				userID,
				login,
			),
			"menu_for_error",
		)
		sendErrorMessage(bot, chatID, "У вас нет прав доступа к этому функционалу", "")
		return false
	}
	return true
}

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

	if message.Text != "" {
		if message.Text == "/start" {
			runSafely(bot, chatID, "start", func() {
				Register(bot, userID, chatID)
			})
			return
		}

		if message.Text == "/menu" {
			deleteUserState(chatID)
			deleteStateMessage(chatID)
			deleteBloсkOrderKeyboard(chatID)
			sendSuccessMessage(bot, chatID, "Выберете действие", "command")
			return
		}
	}

	if state, ok := getUserState(chatID); ok {
		for key := range state {
			switch key {
			case "order":
				if !userValid(bot, userID, chatID) {
					return
				}
				text := message.Text
				runSafely(bot, chatID, "order", func() {
					Order(bot, userID, chatID, text)
				})
				return
			case "transcride":
				if !userValid(bot, userID, chatID) {
					return
				}
				setBloсkOrderKeyboard(chatID, "unlock")
				runSafely(bot, chatID, "transcride", func() {
					handlerTranscrib(bot, userID, chatID, message)
				})
				return
			case "fix_error":
				if !ValidRoot(bot, userID, chatID, message.From.UserName) {
					return
				}
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
		deleteStateMessage(chatID)
		deleteBloсkOrderKeyboard(chatID)
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
		state := getBloсkOrderKeyboard(chatID)
		lastMessageID, ok := getStateMessage(chatID)
		callbackMessageID := callbackQuery.Message.MessageID
		if !ok || lastMessageID != callbackMessageID {
			log.Printf("не существует или не совпадает ID")
			return
		}
		if state == "" || state == "unlock" {
			var (
				like     int = 0
				dis_like int = 0
			)
			if data == "good" {
				like = 1
			} else {
				dis_like = 1
			}
			fl := Mark(bot, userID, chatID, like, dis_like)
			if fl {
				setBloсkOrderKeyboard(chatID, "lock")
			}
		} else {
			log.Printf("User: %d tried to evaluate the transcription several times", chatID)
			sendErrorMessage(bot, chatID, "Вы можете оценить транскрибацию только один раз", "")
		}
		return
	case "order":
		setUserState(chatID, map[string]string{"order": ""})
		sendSuccessMessage(bot, chatID, "Отправьте текст c комментарием ошибки", "")
		return
	case "stat":
		runSafely(bot, chatID, "stat", func() {
			Stat(bot, userID, chatID)
		})
		return
	case "transcride":
		setUserState(chatID, map[string]string{"transcride": ""})
		sendSuccessMessage(bot, chatID, "Отправьте ваше аудио", "")
		return
	case "fix_error":
		setUserState(chatID, map[string]string{"fix_error": ""})
		sendSuccessMessage(bot, chatID, "Отправьте список ID и селектор", "")
		return
	default:
		log.Printf("неизвестная команда: %s", data)
		sendErrorMessage(bot, chatID, "Неизвестная команда", "menu")
		return
	}
}
