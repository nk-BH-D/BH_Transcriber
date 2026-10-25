package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/kirugan/telegram-bot-api/v5"
	tg "github.com/nk-BH-D/BH_Transcribator/tg/bot"
	config "github.com/nk-BH-D/BH_Transcribator/tg/internal/config"
	db_dv "github.com/nk-BH-D/BH_Transcribator/tg/internal/db"
)

func main() {
	conf, err := config.Loader()
	if err != nil {
		log.Fatalf("error whem loading config: %v", err)
	}

	//connect to PostgresUs
	pg_us_db, err := db_dv.NewUsPostgres(conf.DB_URL, conf.POSTGRES_SMOC, conf.POSTGRES_SMIC)
	if err != nil {
		log.Fatalf("failed to connect postgres_us_db: %v", err)
	}
	defer pg_us_db.CloseUs()

	//connect to PostgresError
	pg_err_db, err := db_dv.NewErrPostgres(conf.DB_URL, conf.POSTGRES_SMOC, conf.POSTGRES_SMIC)
	if err != nil {
		log.Fatalf("failed to connect postgres_mus_db: %v", err)
	}
	defer pg_err_db.CloseErr()

	// func for initialization of variables
	tg.Init(conf, pg_us_db, pg_err_db)

	// handler for health-check
	muxHealth := http.NewServeMux()
	muxHealth.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()
		if err := pg_err_db.DB_err.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("db_lib not ready, status: %d, error: %v",
				http.StatusInternalServerError,
				err,
			)
			return
		}
		if err := pg_us_db.DB_us.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("db_us not ready, status: %d, error: %v",
				http.StatusInternalServerError,
				err,
			)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// struct for start health-check handler and greceful shutdown this handler
	HC := &http.Server{
		Addr:    fmt.Sprintf(":%s", conf.HEALTH_CHECK_PORT),
		Handler: muxHealth,
	}

	// start health-check handler inside gorutine
	go func() {
		log.Printf("health-check handler listening port :%s", HC.Addr)
		if err := HC.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve health-check handler: %v", err)
		}
	}()

	// start Tg bot with graceful shutdown
	bot, err := tgbotapi.NewBotAPI(conf.BOT_TOKEN)
	if err != nil {
		log.Panicf("error create bot: %v", err)
	}
	log.Printf("Logged in as %s", bot.Self.UserName)

	// планировщик для отпраки списка репортов
	schedulerCtx, cancelScheduler := context.WithCancel(context.Background())
	schedulerDone := make(chan struct{})
	go func() {
		defer close(schedulerDone)
		runTuesdayScheduler(schedulerCtx, bot, conf.ROOT_ID)
	}()

	up := tgbotapi.NewUpdate(0)
	up.Timeout = 60
	updates := bot.GetUpdatesChan(up)
	botDone := make(chan struct{})

	go func() {
		defer close(botDone)
		for update := range updates {
			if update.Message != nil {
				log.Printf("Entities: %+v", update.Message.Entities)
				go tg.HandleMessage(bot, update.Message)
			} else if update.CallbackQuery != nil {
				go tg.HandleCallback(bot, update.CallbackQuery)
			}
		}
	}()

	// graceful shutdown global
	signa_chan := make(chan os.Signal, 1)
	signal.Notify(signa_chan, syscall.SIGINT, syscall.SIGTERM)

	<-signa_chan

	cancelScheduler()
	<-schedulerDone

	// TG bot
	log.Printf("Graceful shutdown: stopping %s", bot.Self.UserName)
	// oficial method for stopping receiving updates
	bot.StopReceivingUpdates()
	<-botDone
	log.Printf("%s stopped", bot.Self.UserName)
}

func runTuesdayScheduler(ctx context.Context, bot *tgbotapi.BotAPI, rootID int64) {
	for {
		nextRun := nextTuesdayAtTen(time.Now())

		log.Printf(
			"Следующая отправка списка ошибок: %s (через %s)",
			nextRun.Format("2006-01-02 15:04:05 MST"),
			time.Until(nextRun).Round(time.Second),
		)

		timer := time.NewTimer(time.Until(nextRun))

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if ctx.Err() != nil {
				return
			}
			log.Printf("Запускается запланированная отправка списка ошибок")
			tg.GetErrorList(bot, rootID)
		}
	}
}

func nextTuesdayAtTen(now time.Time) time.Time {
	daysUntilTuesday := (int(time.Tuesday) - int(now.Weekday()) + 7) % 7
	nextRun := time.Date(
		now.Year(),
		now.Month(),
		now.Day()+daysUntilTuesday,
		10,
		0,
		0,
		0,
		now.Location(),
	)
	if !nextRun.After(now) {
		nextRun = nextRun.AddDate(0, 0, 7)
	}
	return nextRun
}
