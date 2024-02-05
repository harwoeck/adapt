package adapt

import (
	"fmt"
	"log/slog"
)

func (e *exec) stagePrepareRemote() error {
	e.log.Debug("prepare remote")

	// list all already applied migrations
	applied, err := e.driver.ListMigrations()
	if err != nil {
		e.log.Error("failed to list already applied migrations from driver", "error", err)
		return err
	}
	e.log.Info("loaded migrations from driver", "applied_migration_amount", len(applied))

	// run health check of applied migration data
	err = healthCheckAppliedMigration(applied, e.log)
	if err != nil {
		return err
	}

	// save to exec
	e.applied = applied

	e.log.Info("prepare remote successful")
	return nil
}

func healthCheckAppliedMigration(applied []*Migration, log *slog.Logger) error {
	for _, m := range applied {
		if m.Finished == nil {
			log.Error("migration started but never finished according to saved meta data. Check your integrity manually",
				"migration_id", m.ID, "started_on", m.Started)
			return fmt.Errorf("migration started but never finished according to saved meta data. Check your integrity manually")
		}
	}

	return nil
}
