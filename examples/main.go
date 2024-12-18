package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ozoli99/Kaida/db"
	"github.com/ozoli99/Kaida/models"
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
	newAppointment := models.Appointment{
		CustomerName: "John Doe",
		Time: time.Now(),
		Duration: 60,
		Notes: "Massage Therapy",
	}

	id, err := svc.Create(newAppointment)
	if err != nil {
		log.Fatalf("Failed to create appointment: %v", err)
	}
	fmt.Printf("Created appointment with ID: %d\n", id)
}