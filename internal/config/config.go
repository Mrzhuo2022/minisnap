package config

import (
	"errors"
	"os"
)

// Config 描述服务器运行时所需的关键配置。
type Config struct {
	BindAddr      string
	AdminPassword string
	ContentDir    string
}

// Load 从环境变量读取配置，并提供合理的默认值。
func Load() (Config, error) {
	cfg := Config{
		BindAddr:   getEnvDefault("BIND_ADDR", ":8080"),
		ContentDir: getEnvDefault("CONTENT_DIR", "content"),
	}

	cfg.AdminPassword = os.Getenv("ADMIN_PASSWORD")
	if cfg.AdminPassword == "" {
		return Config{}, errors.New("ADMIN_PASSWORD is required")
	}

	return cfg, nil
}

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
