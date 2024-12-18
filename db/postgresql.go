package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ozoli99/Kaida/models"

	_ "github.com/lib/pq"
)

type PostgresDB struct {
	DB *sql.DB
}

func (p *PostgresDB) Init() error {
	connStr := "host=localhost user=postgres password=yourpassword dbname=appointments sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil  {
		return fmt.Errorf("Failed to connect to PostgreSQL: %v", err)
	}

	createTableStmt := `
	CREATE TABLE IF NOT EXISTS appointments (
		id SERIAL PRIMARY KEY,
		customer_name TEXT NOT NULL,
		time TIMESTAMP NOT NULL,
		duration INTEGER NOT NULL,
		notes TEXT
	);`

	if _, err := db.Exec(createTableStmt); err != nil {
		return fmt.Errorf("Failed to create table: %v", err)
	}
	p.DB = db
	return nil
}

func (p *PostgresDB) CreateAppointment(a models.Appointment) (int, error) {
	var id int
	err := p.DB.QueryRow("INSERT INTO appointments (customer_name, time, duration, notes) VALUES ($1, $2, $3, $4) RETURNING id", a.CustomerName, a.Time, a.Duration, a.Notes).Scan(&id)
	return id, err
}

func (p *PostgresDB) GetAllAppointments() ([]models.Appointment, error) {
	rows, err := p.DB.Query("SELECT id, customer_name, time, duration, notes FROM appointments")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appointments []models.Appointment
	for rows.Next() {
		var a models.Appointment
		var t time.Time
		if err := rows.Scan(&a.ID, &a.CustomerName, &t, &a.Duration, &a.Notes); err != nil {
			return nil, err
		}
		a.Time = t
		appointments = append(appointments, a)
	}
	return appointments, nil
}

func (p *PostgresDB) UpdateAppointment(a models.Appointment) error {
	_, err := p.DB.Exec("UPDATE appointments SET customer_name = $1, time = $2, duration = $3, notes = $4 WHERE id = $5", a.CustomerName, a.Time, a.Duration, a.Notes, a.ID)
	return err
}

func (p *PostgresDB) DeleteAppointment(id int) error {
	_, err := p.DB.Exec("DELETE FROM appointments WHERE id = $1", id)
	return err
}