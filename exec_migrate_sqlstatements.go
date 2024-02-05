package adapt

import (
	"fmt"
	"time"
)

func (e *exec) migrateWithSqlStatements(parsed *ParsedMigration, beforeFinishCallback func(target DBTarget) error) error {
	if !e.driverIsDatabaseDriver {
		e.log.Error("underlying driver isn't a DatabaseDriver! No way to apply a SqlStatementsSource")
		return fmt.Errorf("SqlStatementsSource usage violation")
	}

	e.log.Debug("parsed migration has n statements", "n", len(parsed.Stmts))

	if e.driverIsDatabaseDriverCustomMigration {
		e.log.Debug("driver is a DatabaseDriverCustomMigration. Using the provided Migrate callback")

		err := e.driverAsDatabaseDriverCustomMigration.Migrate(parsed, beforeFinishCallback)
		if err != nil {
			e.log.Error("failed to migrate using the custom migrate callback provided", "error", err)
			return err
		}

		return nil
	}

	exec := func(target DBTarget) error {
		for _, s := range parsed.Stmts {
			e.log.Debug("executing statement", "statement", s)

			started := time.Now()
			if _, err := target.Exec(s); err != nil {
				e.log.Error("failed executing statement", "statement", s, "error", err)
				return err
			}
			end := time.Now()

			e.log.Debug("executing statement took", "duration", end.Sub(started))
		}

		if beforeFinishCallback != nil {
			e.log.Debug("beforeFinishCallback is provided. calling so cleanup can be performed within the (eventually running) same transaction")

			err := beforeFinishCallback(target)
			if err != nil {
				e.log.Error("beforeFinishCallback failed", "error", err)
				return err
			} else {
				e.log.Debug("beforeFinishCallback successful")
			}
		}

		return nil
	}

	if !e.driverAsDatabaseDriver.SupportsTx() {
		e.log.Debug("executing statements without transaction, because driver doesn't support transactions")
		return exec(e.driverAsDatabaseDriver.DB())
	}
	if !parsed.UseTx {
		e.log.Debug("executing statements without transaction, because transactions are disabled for this migration specifically")
		return exec(e.driverAsDatabaseDriver.DB())
	}

	ctx, opts := e.driverAsDatabaseDriver.TxBeginOpts()
	e.log.Debug("starting tx")
	tx, err := e.driverAsDatabaseDriver.DB().BeginTx(ctx, opts)
	if err != nil {
		e.log.Error("failed to begin tx", "error", err)
		return err
	}
	defer func() {
		if err != nil {
			e.log.Warn("exec failed. trying to rollback tx", "error", err)
			if errRb := tx.Rollback(); errRb != nil {
				e.log.Error("rollback failed too", "error", errRb)
			} else {
				e.log.Info("rollback successful")
			}

			err = fmt.Errorf("exec failed (%q) but rollback succeeded. Integrity should be protected, but manual cleanup is probably necessary", err)
			return
		}

		e.log.Debug("committing tx")
		err = tx.Commit()
		if err != nil {
			e.log.Error("commit failed", "error", err)
		}
	}()

	e.log.Debug("executing statements in transaction")
	err = exec(tx)
	return err
}
