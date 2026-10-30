package bot

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	tgbotapi "github.com/kirugan/telegram-bot-api/v5"
)

func Register(bot *tgbotapi.BotAPI, userID, chatID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		status string
		fl     bool = false
	)

	if userID == conf.ROOT_ID {
		status = "root"
		fl = true
	} else {
		status = "user"
	}
	err := us_db.InsertUser(ctx, userID, status)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			sendSuccessMessage(bot, chatID, "Вы уже зарегестрированы", "menu")
			return
		} else {
			log.Printf("error when insert user: %v", err)
			sendErrorMessage(bot, chatID, "Internal service error2: попробуйте позже", "")
			return
		}
	}

	if fl {
		sendSuccessMessage(bot, chatID, "Вы root", "menu")
	}

	result := fmt.Sprintf(
		"Добро пожаловать в BH_Transcriber!\n\n"+
			"Ознакомьтесь с документацией и пользовательским соглашением на "+
			"<a href=\"%s\">GitHub</a>\n\n"+
			"Чтобы продолжить, примите соглашение и подпишитесь на канал проекта.",
		html.EscapeString(conf.REPO_URL),
	)
	msg := tgbotapi.NewMessage(chatID, result)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = createRegistrKeybord()
	if _, err := bot.Send(msg); err != nil {
		log.Printf("error when sending register message: %v", err)
		sendErrorMessage(bot, chatID, "Internal service error0:", "")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := us_db.DeleteUserData(ctx, userID)
		if err != nil {
			log.Printf("error when del user in db: %v\nUser: %d", err, userID)
			sendSuccessMessage(
				bot,
				conf.ROOT_ID,
				fmt.Sprintf(
					"Не смогли удалить пользователя: %d\n"+
						"Ошибка: %v",
					userID,
					err,
				),
				"menu_for_err",
			)
		}
	}
}

func AcceptHandler(bot *tgbotapi.BotAPI, userID, chatID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	accept, _, _, _, _, err := us_db.GetUserData(ctx, userID)
	if err != nil {
		log.Printf("error when get users data: %v", err)
		sendErrorMessage(bot, chatID, "Internal service error3: попробуйте снова позже", "")
		return
	}

	if !accept {
		ctxChange, cancelC := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelC()

		err := us_db.UpUserAccept(ctxChange, userID, true)
		if err != nil {
			log.Printf("error when up user accept: %v", err)
			sendErrorMessage(bot, chatID, "Internal service error4: попробуйте снова позже", "")
			return
		}
		sendSuccessMessage(bot, chatID, "Соглашение принято", "")
		return
	} else {
		sendSuccessMessage(bot, chatID, "Вы уже приняли пользовательское соглашение", "menu")
		return
	}

}

func ChekSub(bot *tgbotapi.BotAPI, userID, chatID int64) {
	var (
		text string
		fl   bool
	)

	member, err := bot.GetChatMember(
		tgbotapi.GetChatMemberConfig{
			ChatID: conf.TG_CHANNEL_ID,
			UserID: userID,
		},
	)
	if err != nil {
		log.Printf("error when chek sub: %v", err)
		sendErrorMessage(bot, chatID, "Internal service error5: поробуйте снова позже", "")
		return
	} else {
		switch member.Status {
		case "creator", "administrator", "member":
			text = "✅ Ты подписан на канал!"
			fl = true
		case "restricted":
			if member.IsMember {
				text = "✅ Ты состоишь в канале."
				fl = true
			} else {
				text = "❌ Сначала подпишись на канал, потом проверь подписку."
				fl = false
			}
		default:
			text = "❌ Сначала подпишись на канал потом проверь подписку."
			fl = false
		}

		if fl {
			sendSuccessMessage(bot, chatID, text, "menu")
			return
		}

		if !fl {
			sendErrorMessage(bot, chatID, text, "")
			return
		}
	}
}

func Mark(bot *tgbotapi.BotAPI, userID, chatID int64, like, dis_like int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := us_db.UpUserData(ctx, userID, 0, like, dis_like)
	if err != nil {
		log.Printf("error when up user data: %v", err)
		sendErrorMessage(bot, chatID, "Internal service error6: попробуйте снова позже", "menu")
		return false
	}

	if like != 0 {
		if msgID, ok := getStateMessage(chatID); ok {
			keyboard := createGoodKeybord()
			editMessageMarkup(bot, chatID, msgID, keyboard)
		}
	} else {
		if msgID, ok := getStateMessage(chatID); ok {
			keyboard := createBadKeybord()
			editMessageMarkup(bot, chatID, msgID, keyboard)
		}
	}
	// показывает что всё прошло хорошо и можно блокировать клаву
	return true
}

func Order(bot *tgbotapi.BotAPI, userID, chatID int64, text string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	error_id, err := err_db.InsertError(ctx, userID, text)
	if err != nil {
		log.Printf("error when inser error: %v", err)
		sendErrorMessage(bot, chatID, "Internal service error7: попробуйте снова позже", "menu")
		return
	}

	otvet := fmt.Sprintf(
		"Ваше коментарий добавлен и будет рассмотрен.\n"+
			"ID вашего обращения: %d.\n"+
			"Спасибо за вклад в сообщество.\n"+
			"Вы можете продолжить трансрибацию или перейти в Меню.",
		error_id)
	sendSuccessMessage(bot, chatID, otvet, "menu")
	setUserState(chatID, map[string]string{"transcride": ""})
}

func Stat(bot *tgbotapi.BotAPI, userID, chatID int64) {
	ctxUS, cancelUS := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelUS()

	_, num_t, like, dis_like, reg_time, err := us_db.GetUserData(ctxUS, userID)
	if err != nil {
		log.Printf("error when getting user date: %v", err)
		sendErrorMessage(bot, chatID, "Internal service error8: попробуйте снова позже", "menu")
		return
	}

	ctxERR, cancelERR := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelERR()
	var (
		res_err  strings.Builder
		dick_err = make(map[int64]string)
		keys     = make([]int64, 0, len(dick_err))
	)

	dick_err, err = err_db.GetUserError(ctxERR, userID)
	if err != nil {
		log.Printf("error when get user error: %v", err)
		sendErrorMessage(bot, chatID, "Internal service error9: попробуйте снова позже", "menu")
		return
	}

	if len(dick_err) != 0 {
		//сортировка ключей map
		for k := range dick_err {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})

		// формирования сообщения
		for _, key := range keys {
			var smail string
			if dick_err[key] == "resolved" {
				smail = "✅"
			} else {
				smail = "🕓"
			}

			format := fmt.Sprintf("#%d: %s%s", key, smail, dick_err[key])
			res_err.WriteString(format)
			res_err.WriteString("\n")
		}
	} else {
		res_err.WriteString("Обращений пока нет")
	}

	result := fmt.Sprintf(
		"📊 Ваша статистика\n\n"+
			"🆔 ID: %d\n"+
			"📅 Регистрация: %s\n\n"+
			"🎧 Количество транскрибаций: %d\n"+
			"👍 Понравилось: %d\n"+
			"👎 Не понравилось: %d\n"+
			"📨 Ваши обращения:\n", userID, reg_time.Format("02.01.2006 15:04:05"), num_t, like, dis_like,
	) + res_err.String()

	sendSuccessMessage(bot, chatID, result, "menu")
}

func FixError(bot *tgbotapi.BotAPI, chatID int64, errorStr string) {
	var (
		coolList   = make([]string, 0)
		set_status string
	)

	parts := strings.Fields(errorStr)
	if len(parts) < 2 {
		log.Printf("parts == 0")
		sendErrorMessage(bot, chatID, "Укажите хотя бы один ID и селектор", "menu")
		return
	}

	// проверка селектора
	if parts[len(parts)-1] == "r" {
		set_status = "resolved"
	} else if parts[len(parts)-1] == "nvc" {
		set_status = "no_valid_comment"
	} else {
		log.Printf("no valid selector")
		sendErrorMessage(bot, chatID, "Вы указали не правильный селектор или не указали его вовсе", "")
		return
	}

	// удаляем селектор перед обработкой
	parts = parts[:len(parts)-1]

	for _, id := range parts {
		nID, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
		if err != nil {
			log.Printf("error when convert id: %v", err)
			sendErrorMessage(
				bot,
				chatID,
				fmt.Sprintf(
					"ErrorID должно быть целым числом, но сейчас: %s,\nисправте ввод и повторите попытку",
					id,
				),
				"",
			)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		status, err := err_db.FindError(ctx, nID)
		cancel()
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("there is no error with this id: %d", nID)
				sendErrorMessage(
					bot,
					chatID,
					fmt.Sprintf(
						"Ошибки с таким ID: %d не существует,\nисправте ввод и повторите попытку",
						nID,
					),
					"",
				)
				continue
			} else {
				log.Printf("error when finding id")
				sendErrorMessage(bot, chatID, "Internal service error12: попробуйте попытку позже", "menu")
				return
			}
		}

		switch status {
		case "in_progress", "resolved", "no_valid_comment":
			log.Printf("полученно id уже решённой проблемы")
			sendErrorMessage(bot, chatID, "Эта проблема была решена", "")
			continue
		case "new":
			ctxRE, cancelRE := context.WithTimeout(context.Background(), 5*time.Second)
			status_after, err := err_db.ResolveError(ctxRE, nID, set_status)
			cancelRE()
			if err != nil {
				if err == sql.ErrNoRows {
					log.Printf("полученно id уже решённой проблемы")
					sendErrorMessage(bot, chatID, "Эта проблема была решена", "")
					continue
				}
				log.Printf("error when set resolve status")
				sendErrorMessage(bot, chatID, "Internal service error13: повторите попытку позже", "menu")
				return
			}

			sendSuccessMessage(
				bot,
				chatID,
				fmt.Sprintf(
					"Ошибка с ID: %d обработанна, статус изменёт на %s",
					nID,
					status_after,
				),
				"",
			)
			coolList = append(coolList, strconv.FormatInt(nID, 10))
		default:
			log.Printf("status incorrect")
			sendErrorMessage(bot, chatID, fmt.Sprintf("Статус не корректный: %s", status), "")
			continue
		}
	}

	if len(coolList) != 0 {
		var res strings.Builder

		res.WriteString("Исправленный задачи:")

		for i, sID := range coolList {
			if i > 0 {
				if i%5 == 0 {
					res.WriteString("\n")
				} else {
					res.WriteString(" ")
				}
			}
			res.WriteString(sID)
		}

		sendSuccessMessage(
			bot,
			chatID,
			res.String(),
			"menu",
		)
	} else {
		log.Printf("0 исправленных проблем")
		sendErrorMessage(bot, chatID, "Ни одна проблема не исправленна", "menu")
		return
	}
}

func GetErrorList(bot *tgbotapi.BotAPI, rootID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errorMap, err := err_db.GetUnresolvedErrors(ctx)
	if err != nil {
		log.Printf("error when getting error list: %v", err)
		sendErrorMessage(bot, rootID, fmt.Sprintf("ОШИБКА: %v", err), "menu")
		return
	}

	var (
		res     strings.Builder
		userIDs = make([]int64, 0, len(errorMap))
	)

	if len(errorMap) != 0 {
		for userID := range errorMap {
			userIDs = append(userIDs, userID)
		}
		sort.Slice(userIDs, func(i, j int) bool {
			return userIDs[i] < userIDs[j]
		})
		res.WriteString("Список всех не обработаный ошибок, обозначенных пользователями\n\n")

		for _, userID := range userIDs {
			errorIDs := make([]int64, 0, len(errorMap[userID]))
			for errorID := range errorMap[userID] {
				errorIDs = append(errorIDs, errorID)
			}
			sort.Slice(errorIDs, func(i, j int) bool {
				return errorIDs[i] < errorIDs[j]
			})

			res.WriteString(fmt.Sprintf("UserID: %d\n", userID))

			for _, errorID := range errorIDs {
				comment := errorMap[userID][errorID]
				res.WriteString(fmt.Sprintf(
					"ErrID: %d\nComment: %s\n",
					errorID,
					comment,
				))
			}
			res.WriteString("\n")
		}
	} else {
		res.WriteString("Обращений пока нет")
	}

	sendSuccessMessage(bot, rootID, res.String(), "menu_for_error")
}

type DataForSend struct {
	Body        bytes.Buffer
	ContentType string
}

type DataFromParser struct {
	Text        string  `json:"text"`
	Language    string  `json:"language"`
	Probability float64 `json:"language_probability"`
}

func handlerTranscrib(bot *tgbotapi.BotAPI, userID, chatID int64, message *tgbotapi.Message) {
	// TODO:
	// надо реализовать функцию помощник которая будет доставать аудио дорожку из кружкоф
	// надо оставить боту возможноть работать не только с голосовыми и кружками но и
	// с обычными ауди файлами с записями голоса, например песни и т.д.
	// отправляем сообщение о начале работ
	text := "Обрабатываем, пожалуйтса подождите."
	msg := tgbotapi.NewMessage(chatID, text)
	// отправляем сообщение о начале работ
	sendMsg, sendErr := bot.Send(msg)
	if sendErr != nil {
		log.Printf("error sending message: %s, %v", text, sendErr)
		sendErrorMessage(bot, chatID, "Ошибка со стороны Telegram: уже решаем", "menu")
		return
	}

	// отправляем стикер
	stic := tgbotapi.NewSticker(chatID, tgbotapi.FileID(conf.STICKER_PROCESSING_ID))
	sendStic, sendErrStic := bot.Send(stic)
	if sendErrStic != nil {
		log.Printf("error sending stiker: %v", sendErrStic)
		// удаляем сообщение о начале работ
		delMsg := tgbotapi.NewDeleteMessage(chatID, sendMsg.MessageID)
		_, errDel := bot.Request(delMsg)
		if errDel != nil {
			log.Printf("error when deleting message: %v", errDel)
		}
		return
	}

	var fileID, fileName string
	msgID := sendMsg.MessageID
	sticID := sendStic.MessageID

	switch {
	case message.Voice != nil:
		fileID = message.Voice.FileID
		fileName = "voice.ogg"
		log.Printf("Voice: ID:%s, name: %s", fileID, fileName)

		dFS, err := handlerDounloadForVoice(bot, fileID, fileName, chatID)
		if err != nil {
			log.Printf("error when dounload file: %v", err)
			sendErrorMessage(bot, chatID, "Internal service error17: попробуйте снова позже", "menu")
			delMessageAndSticer(bot, msgID, sticID, chatID)
			return
		}
		if dFS == nil {
			delMessageAndSticer(bot, msgID, sticID, chatID)
			return
		}

		errAs := headAssistent(dFS, bot, userID, chatID)
		if errAs != nil {
			log.Printf("error in headAssistent: %v", err)
			sendErrorMessage(bot, chatID, "Internal service error10: поробуйте снова позже", "menu")
			delMessageAndSticer(bot, msgID, sticID, chatID)
			return
		}
		delMessageAndSticer(bot, msgID, sticID, chatID)
		return

	case message.VideoNote != nil:
		fileID = message.VideoNote.FileID
		fileName = "circle.mp3"
		log.Printf("Circle: ID: %s, name: %s", fileID, fileName)

		inputPath, err := handlerDounloadForCircle(bot, fileID)
		if err != nil {
			log.Printf("error when dounload circle: %v", err)
			sendErrorMessage(bot, chatID, "Internal service error18: попробуйте снова позже", "menu")
			delMessageAndSticer(bot, msgID, sticID, chatID)
			return
		}

		// отложено удаляем файл
		defer func() {
			if err := os.Remove(inputPath); err != nil {
				log.Printf("remove temporary circle file: %v", err)
			}
		}()

		// достаём аудио и формируем файл
		dFS, err := extractAudio(inputPath, fileName)
		if err != nil {
			log.Printf("error when extract audio or create form: %v", err)
			sendErrorMessage(bot, chatID, "Internal service error19: попробуйте снова позже", "menu")
			delMessageAndSticer(bot, msgID, sticID, chatID)
			return
		}

		errAs := headAssistent(dFS, bot, userID, chatID)
		if errAs != nil {
			log.Printf("error in headAssistent: %v", err)
			sendErrorMessage(bot, chatID, "Internal service error10: поробуйте снова позже", "menu")
			delMessageAndSticer(bot, msgID, sticID, chatID)
			return
		}
		delMessageAndSticer(bot, msgID, sticID, chatID)
		return

	case message.Audio != nil:
		fileID = message.Audio.FileID
		fileName = "audio.mp4"
		log.Printf("Audio file: ID: %s, name: %s", fileID, fileName)
	// тут будет функция обработки
	default:
		log.Printf("wrong file type")
		sendErrorMessage(bot, chatID, "Wrong file type", "menu")
		return
	}
}

func delMessageAndSticer(bot *tgbotapi.BotAPI, msgID int, sticID int, chatID int64) {
	// удаляем сообщение о начале работ
	delMsg := tgbotapi.NewDeleteMessage(chatID, msgID)
	_, errDel := bot.Request(delMsg)
	if errDel != nil {
		log.Printf("error when deleting message: %v", errDel)
	}
	// удаляем стикер
	delStic := tgbotapi.NewDeleteMessage(chatID, sticID)
	_, errDelStic := bot.Request(delStic)
	if errDelStic != nil {
		log.Printf("error when gel sticer: %v", errDelStic)
	}
}

func headAssistent(dFS *DataForSend, bot *tgbotapi.BotAPI, userID, chatID int64) error {
	fromTail, err := handlerSendPy(dFS)
	if err != nil {
		return fmt.Errorf("error when send file: %w", err)
	}
	// если всё прошло успешно отдаём текст пользователю
	data := &DataFromParser{
		Text: fromTail.Text,
		//Language: fromTail.Language,
		//Probability: fromTail.Probability,
	}
	if strings.TrimSpace(data.Text) == "" {
		return fmt.Errorf("text == 0 что-то пошло не так на стороне python")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = us_db.UpUserData(ctx, userID, 1, 0, 0)
	if err != nil {
		log.Printf("error when up user data after transcribation: %v", err)
		sendSuccessMessage(
			bot,
			chatID,
			"Ваша статистика не обновилась по вине внутренней ошибки базы,\n но трансрибация готова",
			"",
		)
	}
	sendSuccessMessage(bot, chatID, data.Text, "ord")
	return nil
}

func handlerDounloadForVoice(bot *tgbotapi.BotAPI, fileID, fileName string, chatID int64) (*DataForSend, error) {
	telegramFile, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("error get file:%v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	downloadReq, err := http.NewRequestWithContext(
		ctx, "GET", telegramFile.Link(conf.BOT_TOKEN), nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error when get link on file:%v", err)
	}

	downloadResp, err := http.DefaultClient.Do(downloadReq)
	if err != nil {
		return nil, fmt.Errorf("error when douloading file: %v", err)
	}
	defer downloadResp.Body.Close()

	// проверка, что скачали
	if downloadResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"Telegram file download returned HTTP %d",
			downloadResp.StatusCode,
		)
	}

	var body bytes.Buffer
	form := multipart.NewWriter(&body)

	part, err := form.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("error when creating form: %v", err)
	}

	maxBytes := conf.MAX_UPLOAD_SIZE * 1024 * 1024

	copied, err := io.Copy(part, io.LimitReader(downloadResp.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("error copying downloaded file: %w", err)
	}
	if copied > maxBytes {
		log.Printf("Файл превышает допустимый размер в %d MB", conf.MAX_UPLOAD_SIZE)
		sendErrorMessage(
			bot,
			chatID,
			fmt.Sprintf("Файл превышает допустимый размер в %d MB", conf.MAX_UPLOAD_SIZE),
			"menu",
		)
		return nil, nil
	}
	if err := form.Close(); err != nil {
		return nil, fmt.Errorf("error closing multipart form: %w", err)
	}

	data := &DataForSend{
		Body:        body,
		ContentType: form.FormDataContentType(),
	}

	return data, nil
}

func handlerDounloadForCircle(bot *tgbotapi.BotAPI, fileID string) (string, error) {
	// получаем файл от TG
	file, err := bot.GetFile(tgbotapi.FileConfig{
		FileID: fileID,
	})
	if err != nil {
		return "", fmt.Errorf("error when get file: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	req, errReq := http.NewRequestWithContext(
		ctx,
		"GET",
		file.Link(conf.BOT_TOKEN),
		nil,
	)
	if errReq != nil {
		return "", fmt.Errorf("error when create dounload req: %w", errReq)
	}

	resp, errResp := http.DefaultClient.Do(req)
	if errResp != nil {
		return "", fmt.Errorf("error when do req: %w", errResp)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TG return status cod: %d", resp.StatusCode)
	}

	// создаём временный файл в docker tmp
	tmp, errTmp := os.CreateTemp("/tmp/transcriber", "circle-*.mp4")
	if errTmp != nil {
		return "", fmt.Errorf("error when create file: %w", errTmp)
	}

	inputPath := tmp.Name()
	keepFile := false

	defer func() {
		_ = tmp.Close()
		if !keepFile {
			err := os.Remove(inputPath)
			if err != nil {
				log.Printf("error when remove file: %v", err)
			}
		}
	}()

	// заполняем временный файл
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", fmt.Errorf("error when save file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("error when close file: %w", err)
	}

	// если всё прошло успешно сообщяем удалятелю не удалять и позвращаетм inputPath
	keepFile = true
	return inputPath, nil
}

func extractAudio(inputPath, fileName string) (*DataForSend, error) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)

	part, err := form.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("error when create form file: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-i", inputPath,
		"-vn",
		"-ac", "1",
		"-ar", "16000",
		"-c:a", "libmp3lame",
		"-q:a", "4",
		"-f", "mp3",
		"pipe:1",
	)

	var ffmpegOutput bytes.Buffer
	cmd.Stdout = part
	cmd.Stderr = &ffmpegOutput

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error when run ffmpeg: %w\n%s", err, ffmpegOutput.String())
	}

	if err := form.Close(); err != nil {
		return nil, fmt.Errorf("error when close form: %w", err)
	}

	data := &DataForSend{
		Body:        body,
		ContentType: form.FormDataContentType(),
	}

	return data, nil
}

func handlerSendPy(data *DataForSend) (*DataFromParser, error) {
	ctxPY, cancelPY := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelPY()

	req, err := http.NewRequestWithContext(
		ctxPY, "POST", conf.WHISPER_PORT, &data.Body,
	)
	if err != nil {
		return nil, fmt.Errorf("error when sending req: %v", err)
	}
	req.Header.Set("Content-Type", data.ContentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error when get resp: %v", err)
	}
	defer resp.Body.Close()
	// получение ответа и отправка в голову
	var forHead DataFromParser

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"transcriber returned HTTP %d",
			resp.StatusCode,
		)
	} else {
		log.Printf("transcription was successful")
		if err := json.NewDecoder(resp.Body).Decode(&forHead); err != nil {
			return nil, fmt.Errorf("error when decoding resp: %v", err)
		}
	}
	return &forHead, nil
}
