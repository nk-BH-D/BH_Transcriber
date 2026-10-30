package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	BOT_TOKEN             string
	WHISPER_PORT          string
	HEALTH_CHECK_PORT     string
	REPO_URL              string
	TG_CHANNEL_URL        string
	POSTGRES_USER         string
	DB_URL                string
	POSTGRES_SMOC         int
	POSTGRES_SMIC         int
	ROOT_ID               int64
	TG_CHANNEL_ID         int64
	MAX_UPLOAD_SIZE       int64
	CUSTOM_EMOJI_ID       string
	STICKER_PROCESSING_ID string
}

func Loader() (*Config, error) {
	bot_token := os.Getenv("BOT_TOKEN")
	if bot_token == "" {
		return nil, fmt.Errorf("error whem getting bot token")
	}

	hcp := os.Getenv("HEALTH_CHECK_PORT")
	if hcp == "" {
		return nil, fmt.Errorf("error when getting nealf check port")
	}

	wp := os.Getenv("WHISPER_PORT")
	if wp == "" {
		return nil, fmt.Errorf("error when getting whisper port")
	}

	repo_url := os.Getenv("REPO_URL")
	if repo_url == "" {
		return nil, fmt.Errorf("error whem getting repo url")
	}

	tg_channel_url := os.Getenv("TG_CHANNEL_URL")
	if tg_channel_url == "" {
		return nil, fmt.Errorf("error whem getting tg channel url")
	}

	db_url := os.Getenv("DB_URL")
	if db_url == "" {
		return nil, fmt.Errorf("error whem getting db users url")
	}

	db_smoc := os.Getenv("POSTGRES_SMOC")
	if db_smoc == "" {
		return nil, fmt.Errorf("error whem getting db users smoc")
	}
	db_smic := os.Getenv("POSTGRES_SMIC")
	if db_smic == "" {
		return nil, fmt.Errorf("error whem getting db users smic")
	}
	int_db_smoc, err := strconv.Atoi(db_smoc)
	if err != nil {
		return nil, err
	}
	int_db_smic, err := strconv.Atoi(db_smic)
	if err != nil {
		return nil, err
	}

	root_id := os.Getenv("ROOT_ID")
	if root_id == "" {
		return nil, fmt.Errorf("error whem getting root id")
	}
	int_root_id, err := strconv.ParseInt(root_id, 10, 64)
	if err != nil {
		return nil, err
	}

	tg_id := os.Getenv("TG_CHANNEL_ID")
	if tg_id == "" {
		return nil, fmt.Errorf("error when getting tg chanal id")
	}
	int_th_id, err := strconv.ParseInt(tg_id, 10, 64)
	if err != nil {
		return nil, err
	}

	mus := os.Getenv("MAX_UPLOAD_SIZE")
	if mus == "" {
		return nil, fmt.Errorf("error when getting mus")
	}
	int_mus, err := strconv.ParseInt(mus, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error when parse int mus")
	}
	ceID := os.Getenv("CUSTOM_EMOJI_ID")
	if ceID == "" {
		return nil, fmt.Errorf("error when getting CUSTOM_EMOJI_ID")
	}
	spID := os.Getenv("STICKER_PROCESSING_ID")
	if spID == "" {
		return nil, fmt.Errorf("error when getting stiker ID")
	}

	return &Config{
		BOT_TOKEN:             bot_token,
		WHISPER_PORT:          wp,
		HEALTH_CHECK_PORT:     hcp,
		REPO_URL:              repo_url,
		TG_CHANNEL_URL:        tg_channel_url,
		DB_URL:                db_url,
		POSTGRES_SMOC:         int_db_smoc,
		POSTGRES_SMIC:         int_db_smic,
		ROOT_ID:               int_root_id,
		TG_CHANNEL_ID:         int_th_id,
		MAX_UPLOAD_SIZE:       int_mus,
		CUSTOM_EMOJI_ID:       ceID,
		STICKER_PROCESSING_ID: spID,
	}, nil
}
