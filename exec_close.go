package adapt

func (e *exec) stageClose() error {
	e.log.Debug("close")

	if err := e.driver.Close(); err != nil {
		e.log.Error("failed to close driver", "error", err)
		return err
	}

	e.log.Info("close successful")
	return nil
}
