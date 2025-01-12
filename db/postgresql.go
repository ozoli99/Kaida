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

	_, err = connection.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            username VARCHAR(50) NOT NULL,
            email VARCHAR(100) NOT NULL UNIQUE,
            password TEXT NOT NULL,
            role VARCHAR(50) NOT NULL
        );
    `)
    if err != nil {
        return fmt.Errorf("failed to create users table: %v", err)
    }

	_, err = connection.Exec(`
        CREATE TABLE IF NOT EXISTS appointments (
            id SERIAL PRIMARY KEY,
            customer_name VARCHAR(100) NOT NULL,
            time TIMESTAMP NOT NULL,
            duration INT NOT NULL,
            notes TEXT,
            recurrence_rule TEXT,
            status VARCHAR(20) DEFAULT 'Scheduled' 
                CHECK (status IN ('Scheduled', 'Completed', 'Cancelled')),
            resource VARCHAR(100),
            customer_id INT REFERENCES users(id),
			provider_id INT REFERENCES users(id)
        );
    `)
    if err != nil {
        return fmt.Errorf("failed to create appointments table: %v", err)
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
		suggestions, err := db.SuggestAlternativeTimes(appointment.Resource, appointment.Time, appointment.Duration)
		if err != nil {
			return 0, fmt.Errorf("resource conflict: failed to suggest alternatives: %v", err)
		}
		return 0, fmt.Errorf("resource conflict: the resource is already booked. Suggested times: %v", suggestions)
	}

	query = "INSERT INTO appointments (customer_name, time, duration, notes, recurrence_rule, status, resource, customer_id, provider_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id"
	var insertedID int
	err = db.Connection.QueryRow(query, appointment.CustomerName, appointment.Time, appointment.Duration, appointment.Notes, appointment.RecurrenceRule, appointment.Status, appointment.Resource, appointment.CustomerID, appointment.ProviderID).Scan(&insertedID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert appointment: %v", err)
	}

	return insertedID, nil
}

func (db *PostgresDatabase) GetAllAppointments(limit, offset int, filters map[string]interface{}, sort string) ([]models.Appointment, error) {
	query := "SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource, customer_id, provider_id FROM appointments"
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
		if err := rows.Scan(&appointment.ID, &appointment.CustomerName, &appointmentTime, &appointment.Duration, &appointment.Notes, &appointment.RecurrenceRule, &appointment.Status, &appointment.Resource, &appointment.CustomerID, &appointment.ProviderID); err != nil {
			return nil, fmt.Errorf("failed to scan appointment row: %v", err)
		}
		appointment.Time = appointmentTime
		appointments = append(appointments, appointment)
	}

	return appointments, nil
}

func (db *PostgresDatabase) GetAppointmentByID(appointmentID int) (models.Appointment, error) {
	query := "SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource, customer_id, provider_id FROM appointments WHERE id = $1"
	row := db.Connection.QueryRow(query, appointmentID)

	var appointment models.Appointment
	if err := row.Scan(&appointment.ID, &appointment.CustomerName, &appointment.Time, &appointment.Duration, &appointment.Notes, &appointment.RecurrenceRule, &appointment.Status, &appointment.Resource, &appointment.CustomerID, &appointment.ProviderID); err != nil {
		return appointment, err
	}
	return appointment, nil
}

func (db *PostgresDatabase) GetAppointmentsByCustomerID(userID int) ([]models.Appointment, error) {
    query := `
        SELECT 
            id, 
            customer_name, 
            time, 
            duration, 
            notes, 
            recurrence_rule, 
            status, 
            resource,
            customer_id,
			provider_id
        FROM appointments
        WHERE customer_id = $1
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
        var apTime time.Time
        if err := rows.Scan(
            &ap.ID,
            &ap.CustomerName,
            &apTime,
            &ap.Duration,
            &ap.Notes,
            &ap.RecurrenceRule,
            &ap.Status,
            &ap.Resource,
            &ap.CustomerID,
			&ap.ProviderID,
        ); err != nil {
            return nil, err
        }
        ap.Time = apTime
        appointments = append(appointments, ap)
    }
    return appointments, nil
}

func (db *PostgresDatabase) GetAppointmentsByCustomerAndTimeRange(customerName string, startTime, endTime time.Time) ([]models.Appointment, error) {
	query := `SELECT id, customer_name, time, duration, notes, recurrence_rule, status, resource, customer_id, provider_id FROM appointments 
		WHERE customer_name = $1 AND time < $2 AND (time + (duration || ' minutes')::interval) > $3`

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

func (db *PostgresDatabase) GetRecurringAppointments(limit int) ([]models.Appointment, error) {
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

func (db *PostgresDatabase) UpdateAppointment(appointment models.Appointment) error {
	query := "UPDATE appointments SET customer_name = $1, time = $2, duration = $3, notes = $4, recurrence_rule = $5, status = $6, resource = $7, customer_id = $8, provider_id = $9 WHERE id = $10"
	_, err := db.Connection.Exec(query, appointment.CustomerName, appointment.Time, appointment.Duration, appointment.Notes, appointment.RecurrenceRule, appointment.Status, appointment.Resource, appointment.CustomerID, appointment.ProviderID, appointment.ID)
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

func (db *PostgresDatabase) SuggestAlternativeTimes(resource string, startTime time.Time, duration int) ([]time.Time, error) {
	query := `SELECT time, duration FROM appointments WHERE resource = $1 AND time >= $2 ORDER BY time ASC`
	rows, err := db.Connection.Query(query, resource, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch conflicting appointments: %v", err)
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

func (db *PostgresDatabase) CreateUser(user *models.User) error {
    query := `
    INSERT INTO users (username, email, password, role)
    VALUES ($1, $2, $3, $4)
    RETURNING id;
    `

    var newID int
    err := db.Connection.QueryRow(query, user.Username, user.Email, user.Password, user.Role).Scan(&newID)
    if err != nil {
        return fmt.Errorf("failed to insert user: %w", err)
    }

    user.ID = newID
    return nil
}

func (db *PostgresDatabase) GetUserByEmail(email string) (*models.User, error) {
    query := `SELECT id, username, email, password, role FROM users WHERE email = $1 LIMIT 1;`

    user := models.User{}
    err := db.Connection.QueryRow(query, email).Scan(
        &user.ID,
        &user.Username,
        &user.Email,
        &user.Password,
        &user.Role,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to get user by email: %w", err)
    }

    return &user, nil
}

func (db *PostgresDatabase) GetUserByID(userID int) (*models.User, error) {
    query := `SELECT id, username, email, password, role FROM users WHERE id = $1;`

    user := models.User{}
    err := db.Connection.QueryRow(query, userID).Scan(
        &user.ID,
        &user.Username,
        &user.Email,
        &user.Password,
        &user.Role,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to get user by ID: %w", err)
    }

    return &user, nil
}

func (db *PostgresDatabase) UpdateUser(user *models.User) error {
	query := `UPDATE users
		SET
			username = $1,
			email = $2,
			password = $3,
			role = $4
		WHERE id = $5
	`
	_, err := db.Connection.Exec(query, user.Username, user.Email, user.Password, user.Role, user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user with ID %d: %v", user.ID, err)
	}
	return nil
}

func (db *PostgresDatabase) DeleteUser(userID int) error {
	_, err := db.Connection.Exec(`DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user with ID %d: %v", userID, err)
	}
	return nil
}

func (db *PostgresDatabase) GetAllUsers(limit, offset int) ([]models.User, error) {
	query := `
		SELECT id, username, email, password, role
		FROM users
		ORDER BY id
		LIMIT $1
		OFFSET $2
	`

	rows, err := db.Connection.Query(query, limit, offset)
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

func (db *PostgresDatabase) UpdatePassword(userID int, hashedPassword string) error {
	query := `
		UPDATE users
		SET password = $1
		WHERE id = $2
	`
	_, err := db.Connection.Exec(query, hashedPassword, userID)
	if err != nil {
		return fmt.Errorf("failed to update password for user ID %d: %v", userID, err)
	}
	return nil
}
