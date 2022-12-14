package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

// DBSQLite implements the DB interface using SQLite
type DBSQLite struct {
	conn *sql.Conn
	ctx  context.Context
}

// NewDBSQLite creates new SQLite (modernc/sqlite) DB connection
func NewDBSQLite(filename string) (*DBSQLite, error) {

	//sqlite3.Xsqlite3_config(nil, sqlite3.SQLITE_CONFIG_SERIALIZED, 1)

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

	_, err := conn.ExecContext(ctx, `create table if not exists object (object_id varchar(50), property_id varchar(100), data BLOB, PRIMARY KEY (object_id, property_id))`)
	if err != nil {
		log.Printf("unable to create changes table: %v", err)
		return err
	}
	return nil
}

// Add is really an upsert now. Might refactor update to be removed
// Currently this is NOT thread safe...  so as soon as we're dealing with 2 different objects this will
// blow up due to transaction within transaction.
// Need to investigate modernc/sqlite threadsafety. If I be naive and throw a lock around this
// then I can see definite lag on test clients and everything gets out of sync.
//
// Will probably move all writing out to a separate goroutine and have the various processors just dump their
// changes onto a bufferless channel. Goroutine does write, signals back to caller (via another channel?) that
// write is done. Investigate... FIXME(kpfaulkner)
func (db *DBSQLite) Add(objectID string, propertyID string, data []byte) error {

	//t := time.Now()
	ctx := context.Background()
	txn, err := db.conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		return fmt.Errorf("unable to create transaction: %w", err)
	}

	defer txn.Rollback()
	insertObjectStatement := `INSERT INTO object ( object_id, property_id,data) VALUES(?, ?, ?) ON CONFLICT(object_id, property_id) DO UPDATE SET data=?`
	statement, err := txn.PrepareContext(ctx, insertObjectStatement)
	if err != nil {
		log.Errorf("unable to prepare statement: %v", err)
		return fmt.Errorf("unable to prepare statement: %w", err)
	}
	defer statement.Close()

	_, err = statement.Exec(objectID, propertyID, data, data)
	if err != nil {
		log.Errorf("unable to insert object %v", err)
		return fmt.Errorf("unable to insert object: %w", err)
	}
	txn.Commit()

	//log.Debugf("Add took %d ms", time.Since(t).Milliseconds())
	return nil
}

// Delete objectID/propertyID from table.
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

// Update an existing objectID/propertyID with new data.
// Given Add has become an upsert, this function can probably go.
func (db *DBSQLite) Update(objectID string, propertyID string, data []byte) error {

	t := time.Now()

	ctx := context.Background()
	txn, err := db.conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		return fmt.Errorf("unable to create transaction: %w", err)
	}
	defer txn.Rollback()

	updateObjectStatement := `UPDATE object SET data = ? where object_id = ? and property_id = ?`
	statement, err := txn.PrepareContext(ctx, updateObjectStatement)
	if err != nil {
		log.Errorf("unable to prepare statement: %v", err)
		return fmt.Errorf("unable to prepare statement: %w", err)
	}
	defer statement.Close()

	_, err = statement.Exec(data, objectID, propertyID)
	if err != nil {
		log.Errorf("unable to update object %v", err)
		return fmt.Errorf("unable to update object: %w", err)
	}

	txn.Commit()

	log.Debugf("Update took %d ms", time.Since(t).Milliseconds())

	return nil
}

// Import will take a map of property/data and store it as an object.
func (db *DBSQLite) Import(objectID string, properties map[string][]byte) (string, error) {
	panic("Not implemented")
	return "", nil
}

// Get returns an object (id + property/data map)
func (db *DBSQLite) Get(objectID string) (*Object, error) {

	ctx := context.Background()
	queryObjectStatement := `SELECT  property_id, data FROM object WHERE object_id = ?`
	statement, err := db.conn.PrepareContext(ctx, queryObjectStatement)
	if err != nil {
		log.Errorf("unable to prepare statement: %v", err)
		return nil, fmt.Errorf("unable to prepare statement: %w", err)
	}
	defer statement.Close()

	rows, err := statement.QueryContext(ctx, objectID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("get object by object_id: %w", err)
	}

	objectProperties := make(map[string][]byte)
	for rows.Next() {
		var propertyID string
		var data []byte

		err := rows.Scan(&propertyID, &data)
		if err != nil {
			return nil, err
		}
		objectProperties[propertyID] = data
	}

	object := Object{ObjectID: objectID, Properties: objectProperties}
	return &object, nil
}
