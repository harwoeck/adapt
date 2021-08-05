package core

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	logger "github.com/harwoeck/liblog/contract"
)

// MySQLOption provides configuration values for a DatabaseDriver implementing the
// MySQL dialect.
type MySQLOption func(*mysqlDriver) error

// MySQLDBName sets the database name in which adapts meta-table is stored. By
// default, this database is named "_adapt". However, you can also specify your own
// database. During starting adapt the database will be created/checked if exists
// using the MySQLDBCreateStatement
func MySQLDBName(dbName string) MySQLOption {
	return func(driver *mysqlDriver) error {
		dbName = strings.TrimSpace(dbName)
		if len(dbName) == 0 {
			return fmt.Errorf("adapt/core.mysqlDriver: dbName cannot be empty")
		}

		driver.dbName = dbName
		return nil
	}
}

// MySQLDBCreateStatement sets the statement used to create-if-not-exists the
// database used for adapts meta-table. The name must contain a single %s
// placeholder, which will be formatted with the set MySQLDBName or "_adapt"
// by default.
//
// The default script used is:
//     CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
func MySQLDBCreateStatement(stmt string) MySQLOption {
	return func(driver *mysqlDriver) error {
		stmt = strings.TrimSpace(stmt)
		if len(stmt) == 0 {
			return fmt.Errorf("adapt/core.mysqlDriver: stmt cannot be empty")
		}

		driver.dbCreateStmt = stmt
		return nil
	}
}

// MySQLTableName sets the table name for adapts meta-table. By default, this is
// "_migrations"
func MySQLTableName(tableName string) MySQLOption {
	return func(driver *mysqlDriver) error {
		tn := strings.TrimSpace(tableName)
		if len(tn) == 0 {
			return fmt.Errorf("adapt/core.mysqlDriver: tableName cannot be empty")
		}

		driver.tableName = tn
		return nil
	}
}

// MySQLTxBeginOpts provides a factory function for creating a context.Context and
// *sql.TxOptions. If this factory is provided it will be called when adapt needs
// to start a sql.Tx for running migrations. By default, the values from the Go
// standard library are use (context.Background() and nil for *sql.TxOptions)
func MySQLTxBeginOpts(factory func() (context.Context, *sql.TxOptions)) MySQLOption {
	return func(driver *mysqlDriver) error {
		driver.txBeginOptsFactory = factory
		return nil
	}
}

// MySQLDisableTx disables transaction for this driver. When set adapt will never
// run a migration inside a transaction, even when the ParsedMigration reports using
// a transaction.
func MySQLDisableTx() MySQLOption {
	return func(driver *mysqlDriver) error {
		driver.txDisabled = true
		return nil
	}
}

// NewMySQLDriver returns a DatabaseDriver from a sql.DB and variadic MySQLOption
// that can interact with a MySQL database.
func NewMySQLDriver(db *sql.DB, opts ...MySQLOption) DatabaseDriver {
	return FromSqlStatementsDriver(&mysqlDriver{
		db:           db,
		opts:         opts,
		dbName:       "_adapt",
		dbCreateStmt: "CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		tableName:    "_migrations",
		txBeginOptsFactory: func() (context.Context, *sql.TxOptions) {
			return context.Background(), nil
		},
	})
}

type mysqlDriver struct {
	db                 *sql.DB
	opts               []MySQLOption
	dbName             string
	dbCreateStmt       string
	tableName          string
	log                logger.Logger
	txBeginOptsFactory func() (context.Context, *sql.TxOptions)
	txDisabled         bool
}

func (d *mysqlDriver) Name() string {
	return "driver_mysql"
}

func (d *mysqlDriver) Init(log logger.Logger) error {
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

func (d *mysqlDriver) Healthy() error {
	if d.db == nil {
		return d.log.ErrorReturn("not healthy: provided db is nil")
	}
	if err := d.db.Ping(); err != nil {
		return d.log.ErrorReturn("not healthy: pinging db errors", field("error", err))
	}

	createDB := fmt.Sprintf(d.dbCreateStmt, d.dbName)
	_, err := d.DB().Exec(createDB)
	if err != nil {
		return d.log.ErrorReturn("failed to create or check if database exists",
			field("error", err))
	}

	createTable := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s
(
    id               VARCHAR(255) NOT NULL,
    executor         VARCHAR(255) NOT NULL,
    started          TIMESTAMP(6) NOT NULL,
    finished         TIMESTAMP(6),
    hash             VARCHAR(255),
    adapt            VARCHAR(32)  NOT NULL,
    deployment       VARCHAR(255) NOT NULL,
    deployment_order INT          NOT NULL,
    down             MEDIUMBLOB,
    PRIMARY KEY (id),
    UNIQUE (deployment, deployment_order)
);`, d.tableName)
	_, err = d.DB().Exec(createTable)
	if err != nil {
		return d.log.ErrorReturn("failed to create or check if table exists",
			field("error", err))
	}

	return nil
}

func (d *mysqlDriver) SupportsLocks() bool {
	return true
}

func (d *mysqlDriver) AcquireLock() (query string) {
	// https://dev.mysql.com/doc/refman/8.0/en/lock-tables.html
	return fmt.Sprintf("LOCK TABLE %s WRITE", d.tableName)
}

func (d *mysqlDriver) ReleaseLock() (query string) {
	// https://dev.mysql.com/doc/refman/8.0/en/lock-tables.html
	return "UNLOCK TABLES"
}

func (d *mysqlDriver) ListMigrations() (query string) {
	return fmt.Sprintf("SELECT id, executor, started, finished, hash, adapt, deployment, deployment_order, down FROM %s ORDER BY id", d.tableName)
}

func (d *mysqlDriver) AddMigration(m *Migration) (query string, args []interface{}) {
	return fmt.Sprintf("INSERT INTO %s (id, executor, started, hash, adapt, deployment, deployment_order, down) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", d.tableName),
		[]interface{}{m.ID, m.Executor, m.Started, m.Hash, m.Adapt, m.Deployment, m.DeploymentOrder, m.Down}
}

func (d *mysqlDriver) SetMigrationToFinished(migrationID string) (query string, args []interface{}) {
	return fmt.Sprintf("UPDATE %s SET finished=? WHERE id=?", d.tableName),
		[]interface{}{time.Now().UTC(), migrationID}
}

func (d *mysqlDriver) Close() error {
	return d.db.Close()
}

func (d *mysqlDriver) DB() *sql.DB {
	return d.db
}

func (d *mysqlDriver) SupportsTx() bool {
	return !d.txDisabled
}

func (d *mysqlDriver) TxBeginOpts() (ctx context.Context, opts *sql.TxOptions) {
	return d.txBeginOptsFactory()
}

func (d *mysqlDriver) UseGlobalTx() bool {
	return false
}

func (d *mysqlDriver) DeleteMigration(migrationID string) (query string, args []interface{}) {
	return fmt.Sprintf("DELETE FROM %s WHERE id=?", d.tableName), []interface{}{migrationID}
}
