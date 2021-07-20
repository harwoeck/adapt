package core

func (e *exec) stageInit() error {
	e.log.Debug("init")

	// init driver
	if err := e.driver.Init(e.log); err != nil {
		e.log.Error("failed to init driver", field("error", err))
		return err
	}

	// init sources
	for idx, src := range e.sources {
		if err := src.Init(e.log); err != nil {
			e.log.Error("failed to init source", field("error", err), field("source_index", idx))
			return err
		}

		switch src.(type) {
		case SqlStatementsSource:
		case HookSource:
		default:
			return e.log.ErrorReturn("source must be either implement SqlStatementsSource or HookSource",
				field("source_index", idx))
		}
	}

	e.log.Info("init successful")
	return nil
}
