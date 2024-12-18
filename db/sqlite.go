package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ozoli99/Kaida/models"

	_ "modernc.org/sqlite"
)

type SQLiteDB struct {
	DB *sql.DB
}

func (s *SQLiteDB) Init() error {
	db, err := sql.Open("sqlite", "file:appointments.db?cache=shared&mode=rwc")
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %v", err)
	}

	createTableStmt := `CREATE TABLE IF NOT EXISTS appointments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		customer_name TEXT NOT NULL,
		time DATETIME NOT NULL,
		duration INTEGER NOT NULL,
		notes TEXT
	);`

	if _, err = db.Exec(createTableStmt); err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	s.DB = db
	return nil
}

func (s *SQLiteDB) CreateAppointment(a models.Appointment) (int, error) {
	res, err := s.DB.Exec("INSERT INTO appointments (customer_name, time, duration, notes) VALUES (?, ?, ?, ?)", a.CustomerName, a.Time.Format(time.RFC3339), a.Duration, a.Notes)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (s *SQLiteDB) GetAllAppointments(limit, offset int, filters map[string]interface{}, sort string) ([]models.Appointment, error) {
	baseQuery := "SELECT id, customer_name, time, duration, notes FROM appointments"
	var conditions []string
	var args []interface{}

	if customerName, ok := filters["customer_name"]; ok {
		conditions = append(conditions, "customer_name LIKE ?")
		args = append(args, "%"+customerName.(string)+"%")
	}

	if start, ok := filters["start"]; ok {
		conditions = append(conditions, "time >= ?")
		args = append(args, start.(string))
	}

	if end, ok := filters["end"]; ok {
		conditions = append(conditions, "time <= ?")
		args = append(args, end.(string))
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	if sort != "" {
		baseQuery += " ORDER BY " + sort
	}

	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.DB.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appointments []models.Appointment
	for rows.Next() {
		var a models.Appointment
		var t string
		if err := rows.Scan(&a.ID, &a.CustomerName, &t, &a.Duration, &a.Notes); err != nil {
			return nil, err
		}
		a.Time, _ = time.Parse(time.RFC3339, t)
		appointments = append(appointments, a)
	}
	return appointments, nil
}

func (s *SQLiteDB) UpdateAppointment(a models.Appointment) error {
	_, err := s.DB.Exec("UPDATE appointments SET customer_name = ?, time = ?, duration = ?, notes = ? WHERE id = ?",
		a.CustomerName, a.Time.Format(time.RFC3339), a.Duration, a.Notes, a.ID)
	return err
}

func (s *SQLiteDB) DeleteAppointment(id int) error {
	_, err := s.DB.Exec("DELETE FROM appointments WHERE id = ?", id)
	return err
}