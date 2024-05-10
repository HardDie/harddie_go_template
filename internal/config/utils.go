package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/HardDie/harddie_go_template/internal/logger"
)

func getEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		err := fmt.Errorf("env %q value not found", key)
		logger.Error(err.Error())
		panic(err)
	}
	return value
}

func getEnvAsInt(key string) int {
	value := getEnv(key)
	v, e := strconv.Atoi(value)
	if e != nil {
		err := fmt.Errorf("env %q value invalid int", key)
		logger.Error(err.Error())
		panic(err)
	}
	return v
}
