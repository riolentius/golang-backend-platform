package main

import (
	"log"

	"github.com/joho/godotenv"

	"github.com/riolentius/cahaya-gading-backend/internal/app"
)

func main() {
	_ = godotenv.Load()

	a := app.New()
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
