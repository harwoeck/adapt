package core

import "database/sql"

// Hook provides callback functions for a HookSource migration. Either MigrateUp,
// MigrateUpDB or MigrateUpTx must be used. When the Driver executing the
// migrations isn't a DatabaseDriver only MigrateUp can be used, as adapt has no
// chance of providing a sql.DB or information to start a sql.Tx. MigrateDown can
// provide a ParsedMigration object that will get stored as the Down-Element
// of a stored Migration.
type Hook struct {
	// MigrateUp must be used when the used Driver isn't a DatabaseDriver. The
	// returned error specifies whether the migration succeeded or not.
	MigrateUp func() error
	// MigrateUpDB can be used when the used Driver is a DatabaseDriver. The
	// returned error specifies whether the migration succeeded or not.
	MigrateUpDB func(db *sql.DB) error
	// MigrateUpTx can be used when the executing Driver is a DatabaseDriver. The
	// returned error specifies whether the migration succeeded or not. The
	// provided sql.Tx is fully managed. Therefore the Hook callback is NOT
	// allowed to call tx.Commit or tx.Rollback.
	MigrateUpTx func(tx *sql.Tx) error
	// MigrateDown provides a ParsedMigration that will get stored as the
	// Down-Element of the Hook's associated Migration inside the Driver's
	// meta-storage
	MigrateDown func() *ParsedMigration
}
