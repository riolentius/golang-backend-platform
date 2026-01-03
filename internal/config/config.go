package config

import "os"

type Config struct {
	AppName string
	Port    string

	DatabaseURL string

	JWTSecret         string
	JWTExpiresMinutes string
}

func Load() Config {
	return Config{
		AppName: getEnv("APP_NAME", "cahaya-gading-backend"),
		Port:    getEnv("PORT", "8080"),

		DatabaseURL: mustEnv("DATABASE_URL"),

		JWTSecret:         mustEnv("JWT_SECRET"),
		JWTExpiresMinutes: getEnv("JWT_EXPIRES_MINUTES", "60"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("missing env: " + key)
	}
	return v
}
