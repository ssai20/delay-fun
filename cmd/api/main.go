package main

import (
	"fun-delay/internal/api"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	// Создаем директорию для результатов
	resultsDir := os.Getenv("RESULTS_DIR")
	if resultsDir == "" {
		resultsDir = "./results"
	}
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		log.Fatal("Failed to create results directory:", err)
	}

	// Настраиваем маршруты
	router := api.SetupRoutes(resultsDir)

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
