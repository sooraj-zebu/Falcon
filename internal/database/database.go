package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	DB *sql.DB
}

func New(path string) (*Database, error) {

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	fmt.Println("SQLite Connected")

	return &Database{
		DB: db,
	}, nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}
