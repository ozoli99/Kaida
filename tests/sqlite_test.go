package db_test

import (
	"testing"
	"time"

	"github.com/ozoli99/Kaida/db"
	"github.com/ozoli99/Kaida/models"

	"github.com/stretchr/testify/assert"
)

func TestSQLiteDatabase_CreateAppointment(t *testing.T) {
	database := &db.SQLiteDatabase{}
	err := database.InitializeDatabase()
	assert.NoError(t, err, "Database initialization should succeed")

	appointment := models.Appointment{
		CustomerName: "John Doe",
		Time:         time.Now().Add(1 * time.Hour),
		Duration:     60,
		Notes:        "First Appointment",
		Status:       "Scheduled",
		Resource:     "Room A",
	}
	id, err := database.CreateAppointment(appointment)
	assert.NoError(t, err, "Creating an appointment should succeed")
	assert.NotZero(t, id, "The returned ID should be non-zero")
}

func TestSQLiteDatabase_ResourceConflict(t *testing.T) {
	database := &db.SQLiteDatabase{}
	err := database.InitializeDatabase()
	assert.NoError(t, err, "Database initialization should succeed")

	conflictingAppointment := models.Appointment{
		CustomerName: "Jane Doe",
		Time:         time.Now().Add(2 * time.Hour),
		Duration:     60,
		Status:       "Scheduled",
		Resource:     "Room B",
	}
	_, _ = database.CreateAppointment(conflictingAppointment)

	newAppointment := models.Appointment{
		CustomerName: "John Smith",
		Time:         conflictingAppointment.Time.Add(30 * time.Minute),
		Duration:     60,
		Status:       "Scheduled",
		Resource:     "Room B",
	}
	_, err = database.CreateAppointment(newAppointment)
	assert.Error(t, err, "Creating a conflicting appointment should fail")
}

func TestSQLiteDatabase_SuggestAlternativeTimes(t *testing.T) {
	database := &db.SQLiteDatabase{}
	err := database.InitializeDatabase()
	assert.NoError(t, err, "Database initialization should succeed")

	conflictingAppointment := models.Appointment{
		CustomerName: "Alice",
		Time:         time.Now().Add(3 * time.Hour),
		Duration:     90,
		Status:       "Scheduled",
		Resource:     "Room C",
	}
	_, _ = database.CreateAppointment(conflictingAppointment)

	startTime := conflictingAppointment.Time
	duration := 60
	suggestions, err := database.SuggestAlternativeTimes("Room C", startTime, duration)
	assert.NoError(t, err, "Suggesting alternative times should succeed")
	assert.NotEmpty(t, suggestions, "Alternative time suggestions should not be empty")
}

func TestSQLiteDatabase_GetAllAppointments(t *testing.T) {
	database := &db.SQLiteDatabase{}
	err := database.InitializeDatabase()
	assert.NoError(t, err, "Database initialization should succeed")

	appointments := []models.Appointment{
		{CustomerName: "Test A", Time: time.Now(), Duration: 30, Resource: "Room D"},
		{CustomerName: "Test B", Time: time.Now().Add(1 * time.Hour), Duration: 30, Resource: "Room D"},
	}
	for _, app := range appointments {
		_, _ = database.CreateAppointment(app)
	}

	results, err := database.GetAllAppointments(10, 0, nil, "time ASC")
	assert.NoError(t, err, "Getting all appointments should succeed")
	assert.GreaterOrEqual(t, len(results), len(appointments), "Retrieved appointments should match or exceed the number inserted")
}

func TestSQLiteDatabase_DeleteAppointment(t *testing.T) {
	database := &db.SQLiteDatabase{}
	err := database.InitializeDatabase()
	assert.NoError(t, err, "Database initialization should succeed")

	appointment := models.Appointment{
		CustomerName: "Delete Me",
		Time:         time.Now().Add(4 * time.Hour),
		Duration:     30,
		Status:       "Scheduled",
		Resource:     "Room E",
	}
	id, _ := database.CreateAppointment(appointment)

	err = database.DeleteAppointment(id)
	assert.NoError(t, err, "Deleting the appointment should succeed")

	_, err = database.GetAppointmentByID(id)
	assert.Error(t, err, "Getting a deleted appointment should fail")
}