package adapt

import "fmt"

func (e *exec) migrateWithHook(migrationID string, source HookSource) error {
	hook := source.GetHook(migrationID)

	if hook.MigrateUp != nil {
		return e.migrateWithHookUp(hook)
	}
	if hook.MigrateUpDB != nil {
		return e.migrateWithHookUpDB(hook)
	}
	if hook.MigrateUpTx != nil {
		return e.migrateWithHookUpTx(hook)
	}

	e.log.Error("all hook callbacks are nil. nothing to do ?")
	return ErrInvalidSource
}

func (e *exec) migrateWithHookUp(hook Hook) error {
	e.log.Debug("executing migration using hook.MigrateUp")
	err := hook.MigrateUp()
	if err != nil {
		e.log.Error("failed to migrate using hook.MigrateUp", "error", err)
		return err
	}

	return nil
}

func (e *exec) migrateWithHookUpDB(hook Hook) error {
	if !e.driverIsDatabaseDriver {
		e.log.Error("underlying driver isn't a DatabaseDriver, but Hook uses MigrateUpDB")
		return fmt.Errorf("Hook usage violation")
	}

	e.log.Debug("executing migration using hook.MigrateUpDB")
	err := hook.MigrateUpDB(e.driverAsDatabaseDriver.DB())
	if err != nil {
		e.log.Error("failed to migrate using hook.MigrateUpDB", "error", err)
		return err
	}

	return nil
}

func (e *exec) migrateWithHookUpTx(hook Hook) error {
	if !e.driverIsDatabaseDriver {
		e.log.Error("underlying driver isn't a DatabaseDriver, but Hook uses MigrateUpTx")
		return fmt.Errorf("Hook usage violation")
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

	e.log.Debug("executing migration using hook.MigrateUpTx")
	err = hook.MigrateUpTx(tx)
	if err != nil {
		e.log.Error("failed to migrate using hook.MigrateUpTx", "error", err)
		return err
	}

	return nil
}
