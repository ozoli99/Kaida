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
		return fmt.Errorf("failed to connect to PostgreSQL database: %v", err)
	}

	tableCreationQuery := `
	CREATE TABLE IF NOT EXISTS appointments (
		id SERIAL PRIMARY KEY,
		customer_name TEXT NOT NULL,
		time TIMESTAMP NOT NULL,
		duration INTEGER NOT NULL,
		notes TEXT,
		recurrence_rule TEXT,
		status TEXT DEFAULT 'Scheduled' CHECK(status IN ('Scheduled', 'Completed', 'Cancelled')),
		resource TEXT
	);`

	if _, err := connection.Exec(tableCreationQuery); err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}
	db.Connection = connection
	return nil
}

func (db *PostgresDatabase) CreateAppointment(appointment models.Appointment) (int, error) {
	query := `SELECT COUNT(*) FROM appointments WHERE resource = $1 AND time < $2 AND (time + (duration || ' minutes')::interval) > $3`
	var count int
	err := db.Connection.QueryRow(query, appointment.Resource, appointment.Time, appointment.Time.Add(time.Minute*time.Duration(appointment.Duration))).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to check for resource conflicts: %v", err)
	}
	if count > 0 {
		return 0, fmt.Errorf("resource conflict: the resource is already booked for this time")
	}

	query = "INSERT INTO appointments (customer_name, time, duration, notes, recurrence_rule, status, resource) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id"
	var insertedID int
	err = db.Connection.QueryRow(query, appointment.CustomerName, appointment.Time, appointment.Duration, appointment.Notes, appointment.RecurrenceRule, appointment.Status, appointment.Resource).Scan(&insertedID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert appointment: %v", err)
	}

	return insertedID, nil
}

func (db *PostgresDatabase) GetAllAppointments(limit, offset int, filters map[string]interface{}, sort string) ([]models.Appointment, error) {
	query := "SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource FROM appointments"
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
		if err := rows.Scan(&appointment.ID, &appointment.CustomerName, &appointmentTime, &appointment.Duration, &appointment.Notes, &appointment.RecurrenceRule, &appointment.Status, &appointment.Resource); err != nil {
			return nil, fmt.Errorf("failed to scan appointment row: %v", err)
		}
		appointment.Time = appointmentTime
		appointments = append(appointments, appointment)
	}

	return appointments, nil
}

func (db *PostgresDatabase) GetAppointmentByID(appointmentID int) (models.Appointment, error) {
	query := "SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource FROM appointments WHERE id = $1"
	row := db.Connection.QueryRow(query, appointmentID)

	var appointment models.Appointment
	if err := row.Scan(&appointment.ID, &appointment.CustomerName, &appointment.Time, &appointment.Duration, &appointment.Notes, &appointment.RecurrenceRule, &appointment.Status, &appointment.Resource); err != nil {
		return appointment, err
	}
	return appointment, nil
}

func (db *PostgresDatabase) GetAppointmentsByCustomerAndTimeRange(customerName string, startTime, endTime time.Time) ([]models.Appointment, error) {
	query := `SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource FROM appointments 
		WHERE customer_name = $1 AND time < $2 AND (time + (duration || ' minutes')::interval) > $3`

	rows, err := db.Connection.Query(query, customerName, endTime, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointments: %v", err)
	}
	defer rows.Close()

	var appointments []models.Appointment
	for rows.Next() {
		var appointment models.Appointment
		if err := rows.Scan(&appointment.ID, &appointment.CustomerName, &appointment.Time, &appointment.Duration, &appointment.Notes, &appointment.RecurrenceRule, &appointment.Status, &appointment.Resource); err != nil {
			return nil, fmt.Errorf("failed to scan appointment row: %v", err)
		}
		appointments = append(appointments, appointment)
	}

	return appointments, nil
}

func (db *PostgresDatabase) GetRecurringAppointments(limit int) ([]models.Appointment, error) {
	query := "SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource FROM appointments WHERE recurrence_rule IS NOT NULL"
	rows, err := db.Connection.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recurringAppointments []models.Appointment
	for rows.Next() {
		var appointment models.Appointment
		if err := rows.Scan(&appointment.ID, &appointment.CustomerName, &appointment.Time, &appointment.Duration, &appointment.Notes, &appointment.RecurrenceRule, &appointment.Status, &appointment.Resource); err != nil {
			return nil, err
		}
		recurringAppointments = append(recurringAppointments, appointment)
	}
	return recurringAppointments, nil
}

func (db *PostgresDatabase) UpdateAppointment(appointment models.Appointment) error {
	query := "UPDATE appointments SET customer_name = $1, time = $2, duration = $3, notes = $4, recurrence_rule = $5, status = $6, resource = $7 WHERE id = $8"
	_, err := db.Connection.Exec(query, appointment.CustomerName, appointment.Time, appointment.Duration, appointment.Notes, appointment.RecurrenceRule, appointment.Status, appointment.Resource, appointment.ID)
	if err != nil {
		return fmt.Errorf("failed to update appointment: %v", err)
	}
	return nil
}

func (db *PostgresDatabase) UpdateAppointmentStatus(appointmentID int, status string) error {
	query := "UPDATE appointments SET status = $1 WHERE id = $2"
	_, err := db.Connection.Exec(query, status, appointmentID)
	return err
}

func (db *PostgresDatabase) DeleteAppointment(appointmentID int) error {
	_, err := db.Connection.Exec("DELETE FROM appointments WHERE id = $1", appointmentID)
	if err != nil {
		return fmt.Errorf("failed to delete appointment: %v", err)
	}
	return nil
}