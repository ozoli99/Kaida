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

	webSocketServer := api.NewWebSocketServer()
	api.StartWebSocketServer(webSocketServer, "8081")

	httpServer := api.Server{
		Service: &svc,
		WebSocket: webSocketServer,
	}

	httpServer.AddMiddleware(api.LoggingMiddleware)
	httpServer.AddMiddleware(api.CORSMiddleware)

	log.Println("Starting HTTP server on :8080...")
	if err := httpServer.StartServer("8080"); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}