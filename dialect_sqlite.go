package adapt

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// SQLiteOption provides configuration values for a DatabaseDriver implementing the
// SQLite dialect.
type SQLiteOption func(*sqliteDriver) error

// SQLiteDisableTx disables transaction for this driver. When set adapt will never
// run a migration inside a transaction, even when the ParsedMigration reports to
// use a transaction.
func SQLiteDisableTx() SQLiteOption {
	return func(driver *sqliteDriver) error {
		driver.txDisabled = true
		return nil
	}
}

// SQLiteTxBeginOpts provides a factory function for creating a context.Context and
// *sql.TxOptions. If this factory is provided it will be called when adapt needs
// to start an sql.Tx for running migrations. By default the values from the Go
// standard library are use (context.Background() and nil for *sql.TxOptions)
func SQLiteTxBeginOpts(factory func() (context.Context, *sql.TxOptions)) SQLiteOption {
	return func(driver *sqliteDriver) error {
		driver.txBeginOptsFactory = factory
		return nil
	}
}

// NewSQLiteDriver returns a DatabaseDriver from a sql.DB and variadic SQLiteOption
// that can interact with a SQLite database.
func NewSQLiteDriver(db *sql.DB, opts ...SQLiteOption) DatabaseDriver {
	return FromSqlStatementsDriver(&sqliteDriver{
		db:        db,
		opts:      opts,
		tableName: "_adapt_migrations",
		txBeginOptsFactory: func() (context.Context, *sql.TxOptions) {
			return context.Background(), nil
		},
	})
}

type sqliteDriver struct {
	log                *slog.Logger
	db                 *sql.DB
	opts               []SQLiteOption
	tableName          string
	txBeginOptsFactory func() (context.Context, *sql.TxOptions)
	txDisabled         bool
}

func (d *sqliteDriver) Name() string {
	return "driver_sqlite"
}

func (d *sqliteDriver) Init(log *slog.Logger) error {
	d.log = log

	for _, opt := range d.opts {
		err := opt(d)
		if err != nil {
			d.log.Error("init failed due to option error", "error", err)
			return err
		}
	}

	return nil
}

func (d *sqliteDriver) Healthy() error {
	if d.db == nil {
		return fmt.Errorf("adapt.sqliteDriver: not healthy: provided db is nil")
	}
	if err := d.db.Ping(); err != nil {
		d.log.Error("not healthy: pinging db errors", "error", err)
		return err
	}

	create := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s
(
    id               TEXT     NOT NULL,
    executor         TEXT     NOT NULL,
    started          DATETIME NOT NULL,
    finished         DATETIME,
    hash             TEXT,
    adapt            TEXT     NOT NULL,
    deployment       TEXT     NOT NULL,
    deployment_order INT      NOT NULL,
    down             BLOB,
    PRIMARY KEY (id),
    UNIQUE (deployment, deployment_order)
)`, d.tableName)
	_, err := d.DB().Exec(create)
	if err != nil {
		d.log.Error("failed to create or check if table exists", "error", err)
		return err
	}

	return nil
}

func (d *sqliteDriver) SupportsLocks() bool {
	return false
}

func (d *sqliteDriver) AcquireLock() (query string) {
	d.log.Error("not supported")
	panic("not supported")
}

func (d *sqliteDriver) ReleaseLock() (query string) {
	d.log.Error("not supported")
	panic("not supported")
}

func (d *sqliteDriver) ListMigrations() (query string) {
	return fmt.Sprintf("SELECT id, executor, started, finished, hash, adapt, deployment, deployment_order, down FROM %s ORDER BY id", d.tableName)
}

func (d *sqliteDriver) AddMigration(m *Migration) (query string, args []interface{}) {
	return fmt.Sprintf("INSERT INTO %s (id, executor, started, hash, adapt, deployment, deployment_order, down) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", d.tableName),
		[]interface{}{m.ID, m.Executor, m.Started, m.Hash, m.Adapt, m.Deployment, m.DeploymentOrder, m.Down}
}

func (d *sqliteDriver) SetMigrationToFinished(migrationID string) (query string, args []interface{}) {
	return fmt.Sprintf("UPDATE %s SET finished=? WHERE id=?", d.tableName),
		[]interface{}{time.Now().UTC(), migrationID}
}

func (d *sqliteDriver) Close() error {
	return d.db.Close()
}

func (d *sqliteDriver) DB() *sql.DB {
	return d.db
}

func (d *sqliteDriver) SupportsTx() bool {
	return !d.txDisabled
}

func (d *sqliteDriver) TxBeginOpts() (ctx context.Context, opts *sql.TxOptions) {
	return d.txBeginOptsFactory()
}

func (d *sqliteDriver) UseGlobalTx() bool {
	return false
}

func (d *sqliteDriver) DeleteMigration(migrationID string) (query string, args []interface{}) {
	return fmt.Sprintf("DELETE FROM %s WHERE id=?", d.tableName), []interface{}{migrationID}
}
