package core

func (e *exec) stageHealthCheck() error {
	e.log.Debug("health check")

	// check if driver is healthy
	if err := e.driver.Healthy(); err != nil {
		e.log.Error("health check of driver failed", field("error", err))
		return err
	}

	e.log.Info("health check successful")
	return nil
}
