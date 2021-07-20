package core

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	logger "github.com/harwoeck/liblog/contract"
)

// PostgresOption provides configuration values for a DatabaseDriver implementing
// the PostgreSQL dialect.
type PostgresOption func(*postgresDriver) error

// NewPostgresDriver returns a DatabaseDriver from a sql.DB and variadic
// PostgresOption that can interact with a PostgreSQL database.
func NewPostgresDriver(db *sql.DB, opts ...PostgresOption) DatabaseDriver {
	return FromSqlStatementsDriver(&postgresDriver{
		log:       nil,
		db:        db,
		opts:      opts,
		dbName:    "_adapt",
		tableName: "public._migrations",
		txBeginOptsFactory: func() (context.Context, *sql.TxOptions) {
			return context.Background(), nil
		},
	})
}

type postgresDriver struct {
	log                logger.Logger
	db                 *sql.DB
	opts               []PostgresOption
	dbName             string
	dbCreateStmt       string
	tableName          string
	txBeginOptsFactory func() (context.Context, *sql.TxOptions)
	txDisabled         bool
}

func (d *postgresDriver) Name() string {
	return "driver_postgres"
}

func (d *postgresDriver) Init(log logger.Logger) error {
	d.log = log.Named(d.Name())

	for _, opt := range d.opts {
		err := opt(d)
		if err != nil {
			return d.log.ErrorReturn("init failed due to option error", field("error", err))
		}
	}

	d.tableName = fmt.Sprintf("%s.%s", d.dbName, d.tableName)

	return nil
}

func (d *postgresDriver) Healthy() error {
	if d.db == nil {
		return d.log.ErrorReturn("not healthy: provided db is nil")
	}
	if err := d.db.Ping(); err != nil {
		return d.log.ErrorReturn("not healthy: pinging db errors", field("error", err))
	}

	createTable := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s
(
    id               TEXT         NOT NULL,
    executor         TEXT         NOT NULL,
    started          TIMESTAMP(6) NOT NULL,
    finished         TIMESTAMP(6),
    hash             TEXT,
    adapt            TEXT         NOT NULL,
    deployment       TEXT         NOT NULL,
    deployment_order INTEGER      NOT NULL,
    down             BYTEA,
    PRIMARY KEY (id),
    UNIQUE (deployment, deployment_order)
);`, d.tableName)
	_, err := d.DB().Exec(createTable)
	if err != nil {
		return d.log.ErrorReturn("failed to create or check if table exists",
			field("error", err))
	}

	return nil
}

func (d *postgresDriver) SupportsLocks() bool {
	return true
}

func (d *postgresDriver) AcquireLock() (query string) {
	// https://www.postgresql.org/docs/13/sql-lock.html
	return fmt.Sprintf("LOCK TABLE %s IN ACCESS EXCLUSIVE MODE", d.tableName)
}

func (d *postgresDriver) ReleaseLock() (query string) {
	// According to PostgreSQL's documentation locks are automatically released
	// when the transaction is committed.
	return ""
}

func (d *postgresDriver) ListMigrations() (query string) {
	return fmt.Sprintf("SELECT id, executor, started, finished, hash, adapt, deployment, deployment_order, down FROM %s ORDER BY id", d.tableName)
}

func (d *postgresDriver) AddMigration(m *Migration) (query string, args []interface{}) {
	return fmt.Sprintf("INSERT INTO %s (id, executor, started, hash, adapt, deployment, deployment_order, down) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", d.tableName),
		[]interface{}{m.ID, m.Executor, m.Started, m.Hash, m.Adapt, m.Deployment, m.DeploymentOrder, m.Down}
}

func (d *postgresDriver) SetMigrationToFinished(migrationID string) (query string, args []interface{}) {
	return fmt.Sprintf("UPDATE %s SET finished=$1 WHERE id=$2", d.tableName),
		[]interface{}{time.Now().UTC(), migrationID}
}

func (d *postgresDriver) Close() error {
	return d.db.Close()
}

func (d *postgresDriver) DB() *sql.DB {
	return d.db
}

func (d *postgresDriver) SupportsTx() bool {
	return !d.txDisabled
}

func (d *postgresDriver) TxBeginOpts() (ctx context.Context, opts *sql.TxOptions) {
	return d.txBeginOptsFactory()
}

func (d *postgresDriver) UseGlobalTx() bool {
	return true
}

func (d *postgresDriver) DeleteMigration(migrationID string) (query string, args []interface{}) {
	return fmt.Sprintf("DELETE FROM %s WHERE id=$1", d.tableName), []interface{}{migrationID}
}
