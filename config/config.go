package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

type InitConfig struct {
	AlertsEndpoint       string        `env:"ALERTS_ENDPOINT" envDefault:"https://www.oref.org.il/warningMessages/alert/Alerts.json"`
	TelegramBotToken     string        `env:"TELEGRAM_BOT_TOKEN,required"`
	TelegramChatID       string        `env:"TELEGRAM_CHAT_ID,required"`
	TargetLocationFilter string        `env:"TARGET_LOCATION_FILTER,required"`
	PollInterval         time.Duration `env:"POLL_INTERVAL" envDefault:"5s"`
	GoogleMapsAPIKey     string        `env:"GOOGLE_MAPS_API_KEY,required"`
}

func LoadConfig() (*InitConfig, error) {
	_ = godotenv.Load()
	cfg := InitConfig{}

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, nil
}
