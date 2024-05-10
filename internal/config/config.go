package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/HardDie/harddie_go_template/internal/logger"
)

type Config struct {
	HTTP HTTP
}

func Get() Config {
	if err := godotenv.Load(); err != nil {
		if check := os.IsNotExist(err); !check {
			logger.Error(fmt.Sprintf("failed to load env vars: %s", err))
			panic(err)
		}
	}

	cfg := Config{
		HTTP: httpConfig(),
	}
	return cfg
}
