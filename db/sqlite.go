package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ozoli99/Kaida/models"

	_ "modernc.org/sqlite"
)

type SQLiteDatabase struct {
	Connection *sql.DB
}

func (db *SQLiteDatabase) InitializeDatabase() error {
	connection, err := sql.Open("sqlite", "file:appointments.db?cache=shared&mode=rwc")
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %v", err)
	}

	tableCreationQuery := `CREATE TABLE IF NOT EXISTS appointments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		customer_name TEXT NOT NULL,
		time DATETIME NOT NULL,
		duration INTEGER NOT NULL,
		notes TEXT
	);`

	if _, err = connection.Exec(tableCreationQuery); err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	db.Connection = connection
	return nil
}

func (db *SQLiteDatabase) CreateAppointment(appointment models.Appointment) (int, error) {
	result, err := db.Connection.Exec(
		"INSERT INTO appointments (customer_name, time, duration, notes) VALUES (?, ?, ?, ?)",
		appointment.CustomerName, appointment.Time.Format(time.RFC3339), appointment.Duration, appointment.Notes,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert appointment: %v", err)
	}

	insertedID, _ := result.LastInsertId()
	return int(insertedID), nil
}

func (db *SQLiteDatabase) GetAllAppointments(limit, offset int, filters map[string]interface{}, sort string) ([]models.Appointment, error) {
	query := "SELECT id, customer_name, time, duration, notes FROM appointments"
	var conditions []string
	var parameters []interface{}

	if customerName, exists := filters["customer_name"]; exists {
		conditions = append(conditions, "customer_name LIKE ?")
		parameters = append(parameters, "%"+customerName.(string)+"%")
	}

	if startTime, exists := filters["start_time"]; exists {
		conditions = append(conditions, "time >= ?")
		parameters = append(parameters, startTime.(string))
	}

	if endTime, exists := filters["end_time"]; exists {
		conditions = append(conditions, "time <= ?")
		parameters = append(parameters, endTime.(string))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	if sort != "" {
		query += " ORDER BY " + sort
	}

	query += " LIMIT ? OFFSET ?"
	parameters = append(parameters, limit, offset)

	rows, err := db.Connection.Query(query, parameters...)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointments: %v", err)
	}
	defer rows.Close()

	var appointments []models.Appointment
	for rows.Next() {
		var appointment models.Appointment
		var appointmentTime string
		if err := rows.Scan(&appointment.ID, &appointment.CustomerName, &appointmentTime, &appointment.Duration, &appointment.Notes); err != nil {
			return nil, fmt.Errorf("failed to scan appointment row: %v", err)
		}
		appointment.Time, _ = time.Parse(time.RFC3339, appointmentTime)
		appointments = append(appointments, appointment)
	}

	return appointments, nil
}

func (db *SQLiteDatabase) GetAppointmentsByCustomerAndTimeRange(customerName string, startTime, endTime time.Time) ([]models.Appointment, error) {
	query := `SELECT id, customer_name, time, duration, notes FROM appointments WHERE customer_name = ? AND time < ? AND datetime(time, '+' || duration || ' minutes') > ?`

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

func (db *SQLiteDatabase) UpdateAppointment(appointment models.Appointment) error {
	_, err := db.Connection.Exec(
		"UPDATE appointments SET customer_name = ?, time = ?, duration = ?, notes = ? WHERE id = ?",
		appointment.CustomerName, appointment.Time.Format(time.RFC3339), appointment.Duration, appointment.Notes, appointment.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update appointment: %v", err)
	}
	return nil
}

func (db *SQLiteDatabase) DeleteAppointment(appointmentID int) error {
	_, err := db.Connection.Exec("DELETE FROM appointments WHERE id = ?", appointmentID)
	if err != nil {
		return fmt.Errorf("failed to delete appointment: %v", err)
	}
	return nil
}