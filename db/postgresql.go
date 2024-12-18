package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ozoli99/Kaida/models"

	_ "github.com/lib/pq"
)

type PostgresDatabase struct {
	Connection *sql.DB
}

func (db *PostgresDatabase) InitializeDatabase() error {
	connectionString := "host=localhost user=postgres password=yourpassword dbname=appointments sslmode=disable"
	connection, err := sql.Open("postgres", connectionString)
	if err != nil  {
		return fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}

	tableCreationQuery := `
	CREATE TABLE IF NOT EXISTS appointments (
		id SERIAL PRIMARY KEY,
		customer_name TEXT NOT NULL,
		time TIMESTAMP NOT NULL,
		duration INTEGER NOT NULL,
		notes TEXT
	);`

	if _, err := connection.Exec(tableCreationQuery); err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}
	db.Connection = connection
	return nil
}

func (db *PostgresDatabase) CreateAppointment(appointment models.Appointment) (int, error) {
	var insertedID int
	err := db.Connection.QueryRow(
		"INSERT INTO appointments (customer_name, time, duration, notes) VALUES ($1, $2, $3, $4) RETURNING id",
		appointment.CustomerName, appointment.Time, appointment.Duration, appointment.Notes,
	).Scan(&insertedID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert appointment: %v", err)
	}

	return insertedID, nil
}

func (db *PostgresDatabase) GetAllAppointments(limit, offset int, filters map[string]interface{}, sort string) ([]models.Appointment, error) {
	query := "SELECT id, customer_name, time, duration, notes FROM appointments"
	var conditions []string
	var parameters []interface{}

	if customerName, exists := filters["customer_name"]; exists {
		conditions = append(conditions, "customer_name ILIKE $1")
		parameters = append(parameters, "%"+customerName.(string)+"%")
	}

	if startTime, exists := filters["start_time"]; exists {
		conditions = append(conditions, "time >= $2")
		parameters = append(parameters, startTime.(string))
	}

	if endTime, exists := filters["end_time"]; exists {
		conditions = append(conditions, "time <= $3")
		parameters = append(parameters, endTime.(string))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	if sort != "" {
		query += " ORDER BY " + sort
	}

	query += " LIMIT $4 OFFSET $5"
	parameters = append(parameters, limit, offset)

	rows, err := db.Connection.Query(query, parameters...)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointments: %v", err)
	}
	defer rows.Close()

	var appointments []models.Appointment
	for rows.Next() {
		var appointment models.Appointment
		var appointmentTime time.Time
		if err := rows.Scan(&appointment.ID, &appointment.CustomerName, &appointmentTime, &appointment.Duration, &appointment.Notes); err != nil {
			return nil, fmt.Errorf("failed to scan appointment row: %v", err)
		}
		appointment.Time = appointmentTime
		appointments = append(appointments, appointment)
	}

	return appointments, nil
}

func (db *PostgresDatabase) GetAppointmentsByCustomerAndTimeRange(customerName string, startTime, endTime time.Time) ([]models.Appointment, error) {
	query := `SELECT id, customer_name, time, duration, notes FROM appointments 
		WHERE customer_name = $1 AND time < $2 AND (time + (duration || ' minutes')::interval) > $3`

	rows, err := db.Connection.Query(query, customerName, endTime, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointments: %v", err)
	}
	defer rows.Close()

	var appointments []models.Appointment
	for rows.Next() {
		var appointment models.Appointment
		if err := rows.Scan(&appointment.ID, &appointment.CustomerName, &appointment.Time, &appointment.Duration, &appointment.Notes); err != nil {
			return nil, fmt.Errorf("failed to scan appointment row: %v", err)
		}
		appointments = append(appointments, appointment)
	}

	return appointments, nil
}

func (db *PostgresDatabase) UpdateAppointment(appointment models.Appointment) error {
	_, err := db.Connection.Exec(
		"UPDATE appointments SET customer_name = $1, time = $2, duration = $3, notes = $4 WHERE id = $5",
		appointment.CustomerName, appointment.Time, appointment.Duration, appointment.Notes, appointment.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update appointment: %v", err)
	}
	return nil
}

func (db *PostgresDatabase) DeleteAppointment(appointmentID int) error {
	_, err := db.Connection.Exec("DELETE FROM appointments WHERE id = $1", appointmentID)
	if err != nil {
		return fmt.Errorf("failed to delete appointment: %v", err)
	}
	return nil
}