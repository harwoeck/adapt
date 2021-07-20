package core

import (
	"context"
	"database/sql"
)

// DBTarget is a container for a sql execution target (either sql.DB or sql.Tx)
type DBTarget interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

// DatabaseDriver is a special extension of Driver. It is always needed when
// adapt should execute a migration from a SqlStatementsSource.
type DatabaseDriver interface {
	Driver
	// DB should return the database connection that gets used to execute
	// sql statements
	DB() *sql.DB
	// SupportsTx reports whether the driver supports database transactions.
	// If SupportsTx is true and ParsedMigration wants transactions to be used
	// all migration statements will be executed within a single transaction.
	SupportsTx() bool
	// TxBeginOpts provides the transaction begin options that should be used
	// when adapt starts a new transaction.
	TxBeginOpts() (ctx context.Context, opts *sql.TxOptions)
	// DeleteMigration should delete the migration from the database. It is
	// important that the provided DBTarget is used, which is a container for
	// the underlying execution target (either sql.DB directly or an eventually
	// running sql.Tx).
	DeleteMigration(migrationID string, target DBTarget) error
}

// DatabaseDriverCustomMigration extends DatabaseDriver by providing a custom
// migration callback. This can be used when the default execution strategy of
// a DatabaseDriver isn't sufficient and the Driver needs fine-grained control
// over every single executed statement. When using DatabaseDriverCustomMigration
// the Driver itself is fully responsible for starting/committing transactions
// and checking if ParsedMigrations can be executed within a transaction.
type DatabaseDriverCustomMigration interface {
	DatabaseDriver
	// Migrate provides a callback for fine-grained manual migrations. It is
	// responsible for the full transaction-lifecycle and checking if all
	// components support transactions. As long as Migrate doesn't return an
	// error adapt assumes that the ParsedMigration was applied successfully
	// and continues with setting the finished time in it's meta store. If
	// Migrate internally starts a transaction is should call beforeFinish
	// before committing the transaction. In certain situations (for example
	// during Down-migrations) adapt will want to execute statements within
	// the same transaction as the migration itself. If Migrate doesn't start
	// it's own migration it should call beforeFinish before returning the
	// function. beforeFinish is allowed to be nil.
	Migrate(migration *ParsedMigration, beforeFinish func(target DBTarget) error) error
}
