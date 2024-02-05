package adapt

import "fmt"

func (e *exec) stageInit() error {
	e.log.Debug("init")

	// init driver
	if err := e.driver.Init(e.log.With("adapt_driver_name", e.driver.Name())); err != nil {
		e.log.Error("failed to init driver", "error", err)
		return err
	}

	// init sources
	for idx, src := range e.sources {
		if err := src.Init(e.log); err != nil {
			e.log.Error("failed to init source", "source_index", idx, "error", err)
			return err
		}

		switch src.(type) {
		case SqlStatementsSource:
		case HookSource:
		default:
			e.log.Error("source must be either implement SqlStatementsSource or HookSource", "source_index", idx)
			return fmt.Errorf("adapt: source must be either implement SqlStatementsSource or HookSource")
		}
	}

	e.log.Info("init successful")
	return nil
}
