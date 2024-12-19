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
		notes TEXT,
		recurrence_rule TEXT,
		status TEXT DEFAULT 'Scheduled' CHECK(status IN ('Scheduled', 'Completed', 'Cancelled')),
		resource TEXT
	);`

	if _, err = connection.Exec(tableCreationQuery); err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	db.Connection = connection
	return nil
}

func (db *SQLiteDatabase) CreateAppointment(appointment models.Appointment) (int, error) {
	query := `SELECT COUNT(*) FROM appointments WHERE resource = ? AND time < ? AND datetime(time, '+' || duration || ' minutes') > ?`
	var count int
	err := db.Connection.QueryRow(query, appointment.Resource, appointment.Time.Format(time.RFC3339), appointment.Time.Add(time.Minute*time.Duration(appointment.Duration)).Format(time.RFC3339)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to check for resource conflicts: %v", err)
	}
	if count > 0 {
		suggestions, err := db.SuggestAlternativeTimes(appointment.Resource, appointment.Time, appointment.Duration)
		if err != nil {
			return 0, fmt.Errorf("resource conflict: failed to suggest alternatives: %v", err)
		}
		return 0, fmt.Errorf("resource conflict: the resource is already booked. Suggested times: %v", suggestions)
	}
	
	query = "INSERT INTO appointments (customer_name, time, duration, notes, recurrence_rule, status, resource) VALUES (?, ?, ?, ?, ?, ?, ?)"
	result, err := db.Connection.Exec(query, appointment.CustomerName, appointment.Time.Format(time.RFC3339), appointment.Duration, appointment.Notes, appointment.RecurrenceRule, appointment.Status, appointment.Resource)
	if err != nil {
		return 0, fmt.Errorf("failed to insert appointment: %v", err)
	}

	insertedID, _ := result.LastInsertId()
	return int(insertedID), nil
}

func (db *SQLiteDatabase) GetAllAppointments(limit, offset int, filters map[string]interface{}, sort string) ([]models.Appointment, error) {
	query := "SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource FROM appointments"
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
		if err := rows.Scan(&appointment.ID, &appointment.CustomerName, &appointmentTime, &appointment.Duration, &appointment.Notes, &appointment.RecurrenceRule, &appointment.Status, &appointment.Resource); err != nil {
			return nil, fmt.Errorf("failed to scan appointment row: %v", err)
		}
		appointment.Time, _ = time.Parse(time.RFC3339, appointmentTime)
		appointments = append(appointments, appointment)
	}

	return appointments, nil
}

func (db *SQLiteDatabase) GetAppointmentByID(appointmentID int) (models.Appointment, error) {
	query := "SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource FROM appointments WHERE id = ?"
	row := db.Connection.QueryRow(query, appointmentID)

	var appointment models.Appointment
	if err := row.Scan(&appointment.ID, &appointment.CustomerName, &appointment.Time, &appointment.Duration, &appointment.Notes, &appointment.RecurrenceRule, &appointment.Status, &appointment.Resource); err != nil {
		return appointment, err
	}
	return appointment, nil
}

func (db *SQLiteDatabase) GetAppointmentsByCustomerAndTimeRange(customerName string, startTime, endTime time.Time) ([]models.Appointment, error) {
	query := `SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource FROM appointments WHERE customer_name = ? AND time < ? AND datetime(time, '+' || duration || ' minutes') > ?`

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

func (db *SQLiteDatabase) GetRecurringAppointments(limit int) ([]models.Appointment, error) {
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

func (db *SQLiteDatabase) UpdateAppointment(appointment models.Appointment) error {
	_, err := db.Connection.Exec(
		"UPDATE appointments SET customer_name = ?, time = ?, duration = ?, notes = ?, recurrence_rule = ?, status = ?, resource = ? WHERE id = ?",
		appointment.CustomerName, appointment.Time.Format(time.RFC3339), appointment.Duration, appointment.Notes, appointment.RecurrenceRule, appointment.Status, appointment.Resource, appointment.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update appointment: %v", err)
	}
	return nil
}

func (db *SQLiteDatabase) UpdateAppointmentStatus(appointmentID int, status string) error {
	query := "UPDATE appointments SET status = ? WHERE id = ?"
	_, err := db.Connection.Exec(query, status, appointmentID)
	return err
}

func (db *SQLiteDatabase) DeleteAppointment(appointmentID int) error {
	_, err := db.Connection.Exec("DELETE FROM appointments WHERE id = ?", appointmentID)
	if err != nil {
		return fmt.Errorf("failed to delete appointment: %v", err)
	}
	return nil
}

func (db *SQLiteDatabase) SuggestAlternativeTimes(resource string, startTime time.Time, duration int) ([]time.Time, error) {
	query := `SELECT time, duration FROM appointments WHERE resource = ? AND time >= ? ORDER BY time ASC`
	rows, err := db.Connection.Query(query, resource, startTime.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("failed to get conflicting appointments: %v", err)
	}
	defer rows.Close()

	var suggestions []time.Time
	endTime := startTime.Add(time.Minute * time.Duration(duration))
	for rows.Next() {
		var bookedStartTime time.Time
		var bookedDuration int
		if err := rows.Scan(&bookedStartTime, &bookedDuration); err != nil {
			return nil, err
		}
		bookedEndTime := bookedStartTime.Add(time.Minute * time.Duration(bookedDuration))

		if endTime.Before(bookedStartTime) {
			suggestions = append(suggestions, endTime)
			break
		}
		startTime = bookedEndTime
		endTime = startTime.Add(time.Minute * time.Duration(duration))
	}

	suggestions = append(suggestions, endTime)
	return suggestions, nil
}