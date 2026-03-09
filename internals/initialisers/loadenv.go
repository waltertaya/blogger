package initialisers

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		if os.IsNotExist(err) {
			log.Println("No .env found, using environment variables from the host")
			return
		}

		log.Printf("Failed to load .env: %v", err)
	}
}
