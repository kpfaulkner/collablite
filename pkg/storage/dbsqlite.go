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

// createTables creates the tables required for storing the objects.
func createTables(ctx context.Context, conn *sql.Conn) error {

	_, err := conn.ExecContext(ctx, `create table if not exists object (object_id varchar(50), property_id varchar(50), data BLOB, PRIMARY KEY (object_id, property_id))`)
	if err != nil {
		log.Printf("unable to create changes table: %v", err)
		return err
	}
	return nil
}

// Add is really an upsert now. Might refactor update to be removed
func (db *DBSQLite) Add(objectID string, propertyID string, data []byte) error {
	ctx := context.Background()
	txn, err := db.conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		return fmt.Errorf("unable to create transaction: %w", err)
	}

	defer txn.Rollback()

	if _, err := txn.ExecContext(ctx, "INSERT INTO object ( object_id, property_id,data) VALUES(?, ?, ?) ON CONFLICT(object_id, property_id) DO UPDATE SET data=?",
		objectID, propertyID, data, data); err != nil {
		return fmt.Errorf("insert object: %w", err)
	}
	txn.Commit()

	return nil
}

func (db *DBSQLite) Delete(objectID string, propertyID string) error {
	ctx := context.Background()
	txn, err := db.conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		return fmt.Errorf("unable to create transaction: %w", err)
	}

	if _, err := txn.ExecContext(ctx, "DELETE FROM object WHERE object_id = ? AND property_id = ?", objectID, propertyID); err != nil {
		return fmt.Errorf("delete change: %w", err)
	}

	return nil
}

func (db *DBSQLite) Update(objectID string, propertyID string, data []byte) error {
	ctx := context.Background()
	txn, err := db.conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		return fmt.Errorf("unable to create transaction: %w", err)
	}
	defer txn.Rollback()

	if _, err := txn.ExecContext(ctx, "UPDATE object SET data = ? where object_id = ? and property_id = ?",
		data, objectID, propertyID); err != nil {
		return fmt.Errorf("update object of %s: %w", objectID, err)
	}
	txn.Commit()
	return nil
}

func (db *DBSQLite) Import(objectID string, properties map[string][]byte) (string, error) {
	return "", nil
}
