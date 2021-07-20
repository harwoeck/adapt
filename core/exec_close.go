package core

func (e *exec) stageClose() error {
	e.log.Debug("close")

	if err := e.driver.Close(); err != nil {
		e.log.Error("failed to close driver", field("error", err))
		return err
	}

	e.log.Info("close successful")
	return nil
}
