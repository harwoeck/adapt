package core

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

	return e.log.ErrorReturn("all hook callbacks are nil. nothing to do ?")
}

func (e *exec) migrateWithHookUp(hook Hook) error {
	e.log.Debug("executing migration using hook.MigrateUp")
	err := hook.MigrateUp()
	if err != nil {
		e.log.Error("failed to migrate using hook.MigrateUp", field("error", err))
		return err
	}

	return nil
}

func (e *exec) migrateWithHookUpDB(hook Hook) error {
	if !e.driverIsDatabaseDriver {
		return e.log.ErrorReturn("underlying driver isn't a DatabaseDriver, but Hook uses MigrateUpDB")
	}

	e.log.Debug("executing migration using hook.MigrateUpDB")
	err := hook.MigrateUpDB(e.driverAsDatabaseDriver.DB())
	if err != nil {
		e.log.Error("failed to migrate using hook.MigrateUpDB", field("error", err))
		return err
	}

	return nil
}

func (e *exec) migrateWithHookUpTx(hook Hook) error {
	if !e.driverIsDatabaseDriver {
		return e.log.ErrorReturn("underlying driver isn't a DatabaseDriver, but Hook uses MigrateUpTx")
	}

	ctx, opts := e.driverAsDatabaseDriver.TxBeginOpts()
	e.log.Debug("starting tx")
	tx, err := e.driverAsDatabaseDriver.DB().BeginTx(ctx, opts)
	if err != nil {
		e.log.Error("failed to begin tx", field("error", err))
		return err
	}
	defer func() {
		if err != nil {
			e.log.Warn("exec failed. trying to rollback tx", field("error", err))
			if errRb := tx.Rollback(); errRb != nil {
				e.log.Error("rollback failed too", field("error", errRb))
			} else {
				e.log.Info("rollback successful")
			}

			err = e.log.ErrorReturn("exec failed but rollback succeeded. Integrity should be protected, but manual cleanup is probably necessary", field("error", err))
			return
		}

		e.log.Debug("committing tx")
		err = tx.Commit()
		if err != nil {
			e.log.Error("commit failed", field("error", err))
		}
	}()

	e.log.Debug("executing migration using hook.MigrateUpTx")
	err = hook.MigrateUpTx(tx)
	if err != nil {
		e.log.Error("failed to migrate using hook.MigrateUpTx", field("error", err))
		return err
	}

	return nil
}
