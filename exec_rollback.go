package adapt

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

func (e *exec) stageRollback() error {
	e.log.Debug("rollback")

	if !allUnknownProvideParsedDown(e.unknownApplied, e.log) {
		e.log.Error("there are unknown migrations, which don't provide a parsed Down field. Aborting to protect integrity", "unknown_amount", len(e.unknownApplied))
		return fmt.Errorf("adapt: unknown migrations")
	}

	e.log.Info("found n migrations in database that can rollback using provided down migrations", "n", len(e.unknownApplied))

	var reversed []*Migration
	for idx := len(e.unknownApplied) - 1; idx >= 0; idx-- {
		reversed = append(reversed, e.unknownApplied[idx])
	}

	for _, u := range reversed {
		down := &ParsedMigration{}
		err := json.Unmarshal(*u.Down, down)
		if err != nil {
			e.log.Error("failed to unmarshal down migration", "migration_id", u.ID, "error", err)
			return err
		}

		e.log.Info("using parsed down migration to rollback", "migration_id", u.ID)

		err = e.migrateWithSqlStatements(down, func(execDestination DBTarget) error {
			err := e.driverAsDatabaseDriver.DeleteMigration(u.ID, execDestination)
			if err != nil {
				e.log.Error("failed to delete migration meta entry, although down migration succeeded before",
					"migration_id", u.ID, "error", err)
				return err
			}
			e.log.Debug("deleted meta entry successful", "migration_id", u.ID)
			return nil
		})
		if err != nil {
			e.log.Error("failed to migrate down", "error", err)
			return err
		}

		// delete the migration we performed a rollback from the applied list
		for i := range e.applied {
			if e.applied[i].ID == u.ID {
				e.applied = append(e.applied[:i], e.applied[i+1:]...)
			}
		}

		e.log.Info("down migration successful", "migration_id", u.ID)
	}

	e.log.Info("rollback successful")
	return nil
}

func allUnknownProvideParsedDown(unknown []*Migration, log *slog.Logger) bool {
	ok := true
	for _, u := range unknown {
		if u.Down == nil {
			ok = false
			log.Warn("found migration without Down field", "migration_id", u.ID)
		}
	}
	return ok
}
