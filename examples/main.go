package main

import (
	"log"

	"github.com/ozoli99/Kaida/api"
	"github.com/ozoli99/Kaida/db"
	"github.com/ozoli99/Kaida/service"
)

func main() {
	var database db.Database
	usePostgres := false

	if usePostgres {
		database = &db.PostgresDB{}
	} else {
		database = &db.SQLiteDB{}
	}

	if err := database.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	svc := service.AppointmentService{DB: database}
	server := api.Server{Service: &svc}

	log.Println("Starting server on :8080...")
	if err := server.StartServer("8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}