package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/joncrlsn/dque"
	"github.com/kpfaulkner/collablite/client"
	log "github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

// DBSQLite implements the DB interface using SQLite
type DBSQLite struct {
	conn *sql.Conn
	ctx  context.Context

	// queue to speed things up.
	queue *dque.DQue
}

func itemBuilder() interface{} {
	return &client.ChangeConfirmation{}
}

// NewDBSQLite creates new SQLite (modernc/sqlite) DB connection
func NewDBSQLite(filename string) (*DBSQLite, error) {

	//sqlite3.Xsqlite3_config(nil, sqlite3.SQLITE_CONFIG_SERIALIZED, 1)

	// queue for potential speed up.
	queue, err := dque.NewOrOpen("objectqueue", `.`, 100, itemBuilder)
	if err != nil {
		return nil, fmt.Errorf("new queue: %w", err)
	}

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
	dbs.queue = queue

	if err := createTables(ctx, conn); err != nil {
		return nil, fmt.Errorf("unable to create table: %w", err)
	}

	// start queue processor
	go dbs.queueProcessor()
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

// queueProcessor reads the persisted BQue and does the real sqlite add.
// This works brilliantly and allows the clients to not be blocked on writing...
// BUT... if a new client connects it will NOT have the latest state since
// changes will be in the queue and not yet written to the DB.
// FIXME(kpfaulkner) - need to think about this.
func (db *DBSQLite) queueProcessor() {
	var iface interface{}
	var err error
	for {
		// want to block until something arrives.
		if iface, err = db.queue.DequeueBlock(); err != nil {
			log.Fatal("Error dequeuing item ", err)
			return // FIXME(kpfaulkner)... return or start goroutine again.. or just continue?
		}

		msg, ok := iface.(*client.ChangeConfirmation)
		if !ok {
			log.Fatal("Dequeued object is not a ChangeConfirmation pointer")
			continue // FIXME(kpfaulkner) check...
		}

		if err = db.add(msg.ObjectID, msg.PropertyID, msg.Data); err != nil {
			log.Errorf("Unable to write to DB: %v", err)
			continue
		}
	}
}

// Add will queue the changes onto a persisted queue and THEN they will be written to the DB
// Purely an optimisation to try and speed things up.
func (db *DBSQLite) Add(objectID string, propertyID string, data []byte) error {

	msg := client.ChangeConfirmation{
		ObjectID:   objectID,
		PropertyID: propertyID,
		Data:       data,
	}
	err := db.queue.Enqueue(&msg)
	if err != nil {
		return fmt.Errorf("unable to enqueue: %w", err)
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
func (db *DBSQLite) add(objectID string, propertyID string, data []byte) error {

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
