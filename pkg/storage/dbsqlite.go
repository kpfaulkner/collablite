package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

// DBSQLite implements the DB interface using SQLite
type DBSQLite struct {
	conn *sql.Conn
	ctx  context.Context
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
	dbs.ctx = ctx

	if err := createTables(ctx, conn); err != nil {
		return nil, fmt.Errorf("unable to create table: %w", err)
	}
	return &dbs, nil
}

// createTables creates the tables required for storing the documents.
func createTables(ctx context.Context, conn *sql.Conn) error {

	_, err := conn.ExecContext(ctx, `create table if not exists document (id varchar(50) primary key, object_id varchar(50), property_id varchar(50), value varchar(10000))`)
	if err != nil {
		log.Printf("unable to create changes table: %v", err)
		return err
	}
	return nil
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
