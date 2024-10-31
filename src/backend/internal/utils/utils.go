package utils

import (
	"os"
	"strconv"

	// pb "chord-url-shortening/chordurlshortening"
)

func GetEnvInt(key string, fallback int) (envInt int) {
	if value, exists := os.LookupEnv(key); exists {
		intValue, err := strconv.Atoi(value)
		if err == nil {
			envInt = intValue
			return
		}
	}
	envInt = fallback
	return
}

func GetEnvFloat(key string, fallback float32) float32 {
	if value, exists := os.LookupEnv(key); exists {
		floatValue, err := strconv.ParseFloat(value, 32)
		if err == nil {
			return float32(floatValue)
		}
	}
	return fallback
}

func GetEnvString(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
