package adapt

func (e *exec) acquireDriverLock() error {
	if e.optDisableDriverLocks {
		e.log.Debug("locking disabled by option")
		return nil
	}

	if !e.driver.SupportsLocks() {
		e.log.Debug("locking not supported by driver")
		return nil
	}

	e.log.Debug("locking enabled and supported by driver. Going to acquire an exclusive lock")
	err := e.driver.AcquireLock()
	if err != nil {
		e.log.Error("failed to acquire driver lock", "error", err)
		return err
	}

	e.driverLockAcquired = true
	e.log.Info("acquired an exclusive driver lock")

	return nil
}

func (e *exec) releaseDriverLock() error {
	if !e.driverLockAcquired {
		return nil
	}

	e.log.Debug("releasing driver lock")
	err := e.driver.ReleaseLock()
	if err != nil {
		e.log.Error("failed to release driver lock", "error", err)
		return err
	}

	e.driverLockAcquired = false
	e.log.Info("released driver lock")

	return nil
}
