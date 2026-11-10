package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	GeminiAPIKey      string
	GeminiModel       string
	TelegramBotToken  string
	TelegramChannelID string
	StateFilePath     string
}

func loadConfig() (Config, error) {
	cfg := loadConfigSoft()
	if cfg.GeminiAPIKey == "" {
		return cfg, fmt.Errorf("GEMINI_API_KEY missing")
	}
	if cfg.TelegramBotToken == "" || cfg.TelegramChannelID == "" {
		return cfg, fmt.Errorf("TELEGRAM_BOT_TOKEN / TELEGRAM_CHANNEL_ID missing")
	}
	return cfg, nil
}

func loadConfigSoft() Config {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".config", "techtips", ".env"),
		".env",
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append([]string{filepath.Join(filepath.Dir(exe), ".env")}, candidates...)
	}
	for _, p := range candidates {
		_ = godotenv.Load(p)
	}
	cfg := Config{
		GeminiAPIKey:      os.Getenv("GEMINI_API_KEY"),
		GeminiModel:       envOr("GEMINI_MODEL", "gemini-flash-lite-latest"),
		TelegramBotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChannelID: os.Getenv("TELEGRAM_CHANNEL_ID"),
		StateFilePath:     os.Getenv("STATE_FILE_PATH"),
	}
	if cfg.StateFilePath == "" {
		cfg.StateFilePath = filepath.Join(home, ".local", "share", "techtips", "state.json")
	}
	return cfg
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
