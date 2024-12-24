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

	usersTableQuery := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT NOT NULL,
        email TEXT NOT NULL UNIQUE,
        password TEXT NOT NULL,
        role TEXT NOT NULL
    );`

    if _, err = connection.Exec(usersTableQuery); err != nil {
        return fmt.Errorf("failed to create users table: %v", err)
    }

	appointmentsTableQuery := `CREATE TABLE IF NOT EXISTS appointments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		customer_name TEXT NOT NULL,
		time DATETIME NOT NULL,
		duration INTEGER NOT NULL,
		notes TEXT,
		recurrence_rule TEXT,
		status TEXT DEFAULT 'Scheduled' CHECK(status IN ('Scheduled', 'Completed', 'Cancelled')),
		resource TEXT,
		customer_id INTEGER REFERENCES users(id),
		provider_id INTEGER REFERENCES users(id)
	  );`

	if _, err = connection.Exec(appointmentsTableQuery); err != nil {
		return fmt.Errorf("failed to create appointments table: %v", err)
	}

	db.Connection = connection
	return nil
}

func (db *SQLiteDatabase) CreateAppointment(appointment models.Appointment) (int, error) {
	query := `SELECT COUNT(*) FROM appointments
		      	WHERE resource = ?
				AND time < datetime(?, '+' || duration || ' minutes')
				AND datetime(time, '+' || duration || ' minutes') > ?`
	var count int
	err := db.Connection.QueryRow(query, appointment.Resource, appointment.Time.Add(time.Minute*time.Duration(appointment.Duration)).Format(time.RFC3339), appointment.Time.Format(time.RFC3339)).Scan(&count)
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
	
	query = "INSERT INTO appointments (customer_name, time, duration, notes, recurrence_rule, status, resource, customer_id, provider_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	result, err := db.Connection.Exec(query, appointment.CustomerName, appointment.Time.Format(time.RFC3339), appointment.Duration, appointment.Notes, appointment.RecurrenceRule, appointment.Status, appointment.Resource, appointment.CustomerID, appointment.ProviderID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert appointment: %v", err)
	}

	insertedID, _ := result.LastInsertId()
	return int(insertedID), nil
}

func (db *SQLiteDatabase) GetAllAppointments(limit, offset int, filters map[string]interface{}, sort string) ([]models.Appointment, error) {
	query := "SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource, customer_id, provider_id FROM appointments"
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
		if err := rows.Scan(&appointment.ID, &appointment.CustomerName, &appointmentTime, &appointment.Duration, &appointment.Notes, &appointment.RecurrenceRule, &appointment.Status, &appointment.Resource, &appointment.CustomerID, &appointment.ProviderID); err != nil {
			return nil, fmt.Errorf("failed to scan appointment row: %v", err)
		}
		appointment.Time, _ = time.Parse(time.RFC3339, appointmentTime)
		appointments = append(appointments, appointment)
	}

	return appointments, nil
}

func (db *SQLiteDatabase) GetAppointmentByID(appointmentID int) (models.Appointment, error) {
	query := "SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource, customer_id, provider_id FROM appointments WHERE id = ?"
	row := db.Connection.QueryRow(query, appointmentID)

	var appointment models.Appointment
	if err := row.Scan(&appointment.ID, &appointment.CustomerName, &appointment.Time, &appointment.Duration, &appointment.Notes, &appointment.RecurrenceRule, &appointment.Status, &appointment.Resource, &appointment.CustomerID, &appointment.ProviderID); err != nil {
		return appointment, err
	}
	return appointment, nil
}

func (db *SQLiteDatabase) GetAppointmentsByCustomerID(userID int) ([]models.Appointment, error) {
    query := `
        SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource, customer_id, provider_id
        FROM appointments
        WHERE customer_id = ?
        ORDER BY time ASC
    `
    rows, err := db.Connection.Query(query, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get appointments for user %d: %v", userID, err)
    }
    defer rows.Close()

    var appointments []models.Appointment
    for rows.Next() {
        var ap models.Appointment
        var apTime string
        if err := rows.Scan(&ap.ID, &ap.CustomerName, &apTime, &ap.Duration, &ap.Notes, &ap.RecurrenceRule, &ap.Status, &ap.Resource, &ap.CustomerID, &ap.ProviderID); err != nil {
            return nil, err
        }
        ap.Time, _ = time.Parse(time.RFC3339, apTime)
        appointments = append(appointments, ap)
    }
    return appointments, nil
}

func (db *SQLiteDatabase) GetAppointmentsByCustomerAndTimeRange(customerName string, startTime, endTime time.Time) ([]models.Appointment, error) {
	query := `SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource, customer_id, provider_id FROM appointments WHERE customer_name = ? AND time < ? AND datetime(time, '+' || duration || ' minutes') > ?`

	rows, err := db.Connection.Query(query, customerName, endTime, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointments: %v", err)
	}
	defer rows.Close()

	var appointments []models.Appointment
	for rows.Next() {
		var appointment models.Appointment
		if err := rows.Scan(&appointment.ID, &appointment.CustomerName, &appointment.Time, &appointment.Duration, &appointment.Notes, &appointment.RecurrenceRule, &appointment.Status, &appointment.Resource, &appointment.CustomerID, &appointment.ProviderID); err != nil {
			return nil, fmt.Errorf("failed to scan appointment row: %v", err)
		}
		appointments = append(appointments, appointment)
	}

	return appointments, nil
}

func (db *SQLiteDatabase) GetRecurringAppointments(limit int) ([]models.Appointment, error) {
	query := "SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource, customer_id, provider_id FROM appointments WHERE recurrence_rule IS NOT NULL"
	rows, err := db.Connection.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recurringAppointments []models.Appointment
	for rows.Next() {
		var appointment models.Appointment
		if err := rows.Scan(&appointment.ID, &appointment.CustomerName, &appointment.Time, &appointment.Duration, &appointment.Notes, &appointment.RecurrenceRule, &appointment.Status, &appointment.Resource, &appointment.CustomerID, &appointment.ProviderID); err != nil {
			return nil, err
		}
		recurringAppointments = append(recurringAppointments, appointment)
	}
	return recurringAppointments, nil
}

func (db *SQLiteDatabase) UpdateAppointment(appointment models.Appointment) error {
	_, err := db.Connection.Exec(
		"UPDATE appointments SET customer_name = ?, time = ?, duration = ?, notes = ?, recurrence_rule = ?, status = ?, resource = ?, customer_id = ?, provider_id = ? WHERE id = ?",
		appointment.CustomerName, appointment.Time.Format(time.RFC3339), appointment.Duration, appointment.Notes, appointment.RecurrenceRule, appointment.Status, appointment.Resource, appointment.CustomerID, appointment.ProviderID, appointment.ID,
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

func (db *SQLiteDatabase) CreateUser(u *models.User) error {
    stmt, err := db.Connection.Prepare(`
        INSERT INTO users (username, email, password, role) 
        VALUES (?, ?, ?, ?)
    `)
    if err != nil {
        return err
    }
    res, err := stmt.Exec(u.Username, u.Email, u.Password, u.Role)
    if err != nil {
        return err
    }
    id, err := res.LastInsertId()
    if err != nil {
        return err
    }
    u.ID = int(id)
    return nil
}

func (db *SQLiteDatabase) GetUserByEmail(email string) (*models.User, error) {
    row := db.Connection.QueryRow(`
        SELECT id, username, email, password, role 
        FROM users 
        WHERE email = ? 
        LIMIT 1
    `, email)

    user := models.User{}
    if err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role); err != nil {
        return nil, err
    }

    return &user, nil
}

func (db *SQLiteDatabase) GetUserByID(id int) (*models.User, error) {
    row := db.Connection.QueryRow(`
        SELECT id, username, email, password, role 
        FROM users 
        WHERE id = ?
    `, id)

    user := models.User{}
    if err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role); err != nil {
        return nil, err
    }

    return &user, nil
}

func (db *SQLiteDatabase) UpdateUser(user *models.User) error {
	query := `
		UPDATE users
		SET username = ?, email = ?, password = ?, role = ?
		WHERE id = ?
	`
	_, err := db.Connection.Exec(query, user.Username, user.Email, user.Password, user.Role, user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user with ID %d: %v", user.ID, err)
	}
	return nil
}

func (db *SQLiteDatabase) DeleteUser(userID int) error {
	_, err := db.Connection.Exec(`
		DELETE FROM users
		WHERE id = ?
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user with ID %d: %v", userID, err)
	}
	return nil
}

func (db *SQLiteDatabase) GetAllUsers(limit, offset int) ([]models.User, error) {
	rows, err := db.Connection.Query(`
		SELECT id, username, email, password, role
		FROM users
		ORDER BY id
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %v", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (db *SQLiteDatabase) UpdatePassword(userID int, hashedPassword string) error {
	_, err := db.Connection.Exec(`
		UPDATE users
		SET password = ?
		WHERE id = ?
	`, hashedPassword, userID)
	if err != nil {
		return fmt.Errorf("failed to change password for user %d: %v", userID, err)
	}
	return nil
}
