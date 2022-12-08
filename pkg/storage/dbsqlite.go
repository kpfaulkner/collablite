package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// DBSQLite implements the DB interface using SQLite
type DBSQLite struct {
	conn *sql.Conn
}

func NewDBSQLite(filename string) (*DBSQLite, error) {
	dbs := DBSQLite{}
	db, err := sql.Open("sqlite", filename)
	if err != nil {
		return nil, fmt.Errorf("new sqlitedb: %w", err)
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("new sqlitedb: %w", err)
	}

	dbs.conn = conn
	return &dbs, nil
}

func (db *DBSQLite) Add(objectID string, path string, data []byte) error {
	return nil
}

func (db *DBSQLite) Delete(objectID string, path string) error {
	return nil
}

func (db *DBSQLite) Update(objectID string, path string, data []byte) error {
	return nil
}

func (db *DBSQLite) Import(data []byte) (string, error) {
	return "", nil
}
