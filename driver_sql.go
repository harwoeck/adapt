package adapt

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// SqlStatementsDriver is a special interface for defining a DatabaseDriver that
// only wants to use there database specific dialect. A SqlStatementsDriver can
// be converted into a full DatabaseDriver using a provided adapter, that can be
// accessed using FromSqlStatementsDriver. Basically a SqlStatementsDriver only
// reports it's features and (query, args) pairs for the different needed
// operations that depend on the underlying SQL dialect.
//
// The big advantage of SqlStatementsDriver is that it reduces boilerplate and
// error checking drastically for all primitive DatabaseDriver that support
// sql.DB.
type SqlStatementsDriver interface {
	// Name reports the name of this Driver. It is mainly used for logging
	// output and can be an empty string.
	Name() string
	// Init initializes the internal state of a driver. It should be used to
	// apply options or return an error which the internal state is invalid.
	// For tasks like establishing connections and performing health checks
	// Healthy should be used.
	Init(log *slog.Logger) error
	// Healthy should report if everything is ready and healthy to proceed
	// with running migrations. One example would be to ping the database
	// and check if the connection is intact. Also Healthy is responsible for
	// creating the structure of your meta-storage, e.g. the database and
	// meta-table.
	Healthy() error
	// SupportsLocks reports whether the driver supports locking or not. This
	// influences if AcquireLock and ReleaseLock are called.
	SupportsLocks() bool
	// AcquireLock must return a database query that acquires an exclusive
	// lock.
	AcquireLock() (query string)
	// ReleaseLock must return a database query that released the previously
	// acquired lock.
	ReleaseLock() (query string)
	// ListMigrations must return a database query that selects all Migration
	// data in the following order: ID, Executor, Started, Finished, Hash, Adapt
	// Deployment, DeploymentOrder, Down. The field's types are the same as in the
	// Migration struct.
	ListMigrations() (query string)
	// AddMigration must return a database query and it's corresponding args
	// that insert the passed Migration into the meta-table.
	AddMigration(m *Migration) (query string, args []interface{})
	// SetMigrationToFinished must return a database query and it's corresponding
	// args in order to set the migration with migrationID to finished.
	SetMigrationToFinished(migrationID string) (query string, args []interface{})
	// Close should close all underlying connections opened during Healthy and
	// perform any necessary clean-up operations. Close is always called, even
	// when an error is encountered somewhere or the library panics
	Close() error
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
	// UseGlobalTx instructs the adapter to start a single global transaction
	// for all database queries/executes. When used the transaction is started
	// during Init and committed/rollbacked during Close.
	UseGlobalTx() bool
	// DeleteMigration must return a database query and it's corresponding args
	// in order to delete the specified migration.
	DeleteMigration(migrationID string) (query string, args []interface{})
}

// FromSqlStatementsDriver converts a SqlStatementsDriver to a full DatabaseDriver
// by wrapping it in an internal adapter that handles all sql.DB operations
// according to the features specified by SqlStatementsDriver
func FromSqlStatementsDriver(driver SqlStatementsDriver) DatabaseDriver {
	return &stmtDriver{
		driver: driver,
	}
}

type stmtDriver struct {
	driver   SqlStatementsDriver
	log      *slog.Logger
	target   DBTarget
	tx       *sql.Tx
	rollback bool
}

func (d *stmtDriver) Name() string {
	return d.driver.Name()
}

func (d *stmtDriver) Init(log *slog.Logger) error {
	d.log = log

	err := d.driver.Init(log)
	if err != nil {
		return err
	}

	if d.driver.SupportsTx() && d.driver.UseGlobalTx() {
		log.Debug("driver supports tx and instructs us to use a global tx. Beginning global tx")

		ctx, opts := d.driver.TxBeginOpts()
		tx, err := d.driver.DB().BeginTx(ctx, opts)
		if err != nil {
			log.Error("unable to start tx", "error", err)
			return err
		}

		log.Info("using global tx as database target")
		d.target = tx
		d.tx = tx
	} else {
		d.target = d.driver.DB()
	}

	return nil
}

func (d *stmtDriver) Healthy() error {
	return d.driver.Healthy()
}

func (d *stmtDriver) SupportsLocks() bool {
	return d.driver.SupportsLocks()
}

func (d *stmtDriver) AcquireLock() error {
	var err error
	if query := d.driver.AcquireLock(); len(query) > 0 {
		_, err = d.target.Exec(query)
		if err != nil {
			d.rollback = true
		}
	}
	return err
}

func (d *stmtDriver) ReleaseLock() error {
	var err error
	if query := d.driver.ReleaseLock(); len(query) > 0 {
		_, err = d.target.Exec(query)
		if err != nil {
			d.rollback = true
		}
	}
	return err
}

func (d *stmtDriver) ListMigrations() ([]*Migration, error) {
	var migrations []*Migration

	rows, err := d.target.Query(d.driver.ListMigrations())
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var id, executor, adapt, deployment string
		var deploymentOrder int
		var started time.Time
		var finished sql.NullTime
		var hash sql.NullString
		var down *[]byte

		err = rows.Scan(&id, &executor, &started, &finished, &hash, &adapt, &deployment, &deploymentOrder, &down)
		if err != nil {
			return nil, err
		}

		m := &Migration{
			ID:              id,
			Executor:        executor,
			Started:         started,
			Adapt:           adapt,
			Deployment:      deployment,
			DeploymentOrder: deploymentOrder,
			Down:            down,
		}
		if finished.Valid && finished.Time.Year() > 1 {
			m.Finished = &(finished.Time)
		}
		if hash.Valid {
			m.Hash = &(hash.String)
		}

		migrations = append(migrations, m)
	}
	err = rows.Err()
	if err != nil {
		d.rollback = true
		return nil, err
	}

	return migrations, nil
}

func (d *stmtDriver) AddMigration(m *Migration) error {
	query, args := d.driver.AddMigration(m)
	_, err := d.target.Exec(query, args...)
	if err != nil {
		d.rollback = true
	}
	return err
}

func (d *stmtDriver) Migrate(migration *ParsedMigration, beforeFinish func(target DBTarget) error) error {
	for _, s := range migration.Stmts {
		d.log.Debug("executing statement", "statement", s)

		started := time.Now()
		if _, err := d.target.Exec(s); err != nil {
			d.log.Error("failed executing statement", "statement", s, "error", err)
			d.rollback = true
			return err
		}
		end := time.Now()

		d.log.Debug("executing statement took", "duration", end.Sub(started))
	}

	if beforeFinish != nil {
		d.log.Debug("beforeFinishCallback is provided. calling so cleanup can be performed within the (eventually running) same transaction")

		err := beforeFinish(d.target)
		if err != nil {
			d.log.Error("beforeFinishCallback failed", "error", err)
			d.rollback = true
			return err
		} else {
			d.log.Debug("beforeFinishCallback successful")
		}
	}

	return nil
}

func (d *stmtDriver) SetMigrationToFinished(migrationID string) error {
	query, args := d.driver.SetMigrationToFinished(migrationID)
	_, err := d.target.Exec(query, args...)
	if err != nil {
		d.rollback = true
	}
	return err
}

func (d *stmtDriver) Close() error {
	// if tx is not nil, we started a tx and need to commit/rollback it
	if d.tx != nil {
		d.log.Debug("ending global tx")

		if d.rollback {
			d.log.Debug("rollback of global tx")

			err := d.tx.Rollback()
			if err != nil {
				d.log.Error("rollback of global tx failed", "error", err)
			} else {
				d.log.Info("rollback of global tx succeeded")
			}
		} else {
			d.log.Debug("commit of global tx")

			err := d.tx.Commit()
			if err != nil {
				d.log.Error("commit of global tx failed", "error", err)
			} else {
				d.log.Info("commit of global tx succeeded")
			}
		}
	}

	return d.driver.Close()
}

func (d *stmtDriver) DB() *sql.DB {
	return d.driver.DB()
}

func (d *stmtDriver) SupportsTx() bool {
	return d.driver.SupportsTx()
}

func (d *stmtDriver) TxBeginOpts() (ctx context.Context, opts *sql.TxOptions) {
	return d.driver.TxBeginOpts()
}

func (d *stmtDriver) DeleteMigration(migrationID string, target DBTarget) error {
	query, args := d.driver.DeleteMigration(migrationID)
	_, err := target.Exec(query, args...)
	if err != nil {
		d.rollback = true
	}
	return err
}
