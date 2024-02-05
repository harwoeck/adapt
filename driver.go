package adapt

import "log/slog"

// Driver is the most basic backend against which migrations from a SourceCollection
// are executed. The special extension DatabaseDriver extends this Driver, but doesn't
// differ in usage. The main advantage in splitting Driver and DatabaseDriver is to
// facilitate usages that don't depend an SQL or databases. This property gives adapt
// a general purpose migration library character.
type Driver interface {
	// Name reports the name of this Driver. It is mainly used for logging
	// output and can be an empty string.
	Name() string
	// Init initializes the internal state of a driver. It should be used to
	// apply options or return an error when the internal state is invalid.
	// For tasks like establishing connections and performing health checks
	// Healthy should be used.
	Init(log *slog.Logger) error
	// Healthy should report if everything is ready and healthy to proceed
	// with running migrations. One example would be to ping the database
	// and check if the connection is intact. Also Healthy is responsible for
	// creating the structure of your meta-storage (in the context of a
	// DatabaseDriver this would be, that the database and meta-table are
	// created)
	Healthy() error
	// SupportsLocks reports whether the driver supports locking or not. This
	// influences if AcquireLock and ReleaseLock are called.
	SupportsLocks() bool
	// AcquireLock acquires a lock if SupportsLocks reports that this Driver
	// supports locking
	AcquireLock() error
	// ReleaseLock is called after running migrations and only if AcquireLock
	// successfully acquired a lock (e.g. didn't return an error). ReleaseLock
	// is always called when an lock was acquired, even when an error is encountered
	// somewhere or the library panics
	ReleaseLock() error
	// ListMigrations lists all already applied migrations
	ListMigrations() ([]*Migration, error)
	// AddMigration adds the meta-data of a new migration. After successfully
	// adding this migration to the driver the migration Source will be executed.
	AddMigration(migration *Migration) error
	// SetMigrationToFinished must set the finished field of a migration to
	// the current time, which indicates that this migration has finished
	// successfully.
	SetMigrationToFinished(migrationID string) error
	// Close should close all underlying connections opened during Healthy and
	// perform any necessary clean-up operations. Close is always called, even
	// when an error is encountered somewhere or the library panics
	Close() error
}
