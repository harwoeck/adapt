package core

import (
	"encoding/json"

	logger "github.com/harwoeck/liblog/contract"
)

func (e *exec) stageRollback() error {
	e.log.Debug("rollback")

	if !allUnknownProvideParsedDown(e.unknownApplied, e.log) {
		return e.log.ErrorReturn("there are unknown migrations, which don't provide a parsed Down field. Aborting to protect integrity",
			field("unknown_amount", len(e.unknownApplied)))
	}

	e.log.Info("found n migrations in database that can rollback using provided down migrations", field("n", len(e.unknownApplied)))

	var reversed []*Migration
	for idx := len(e.unknownApplied) - 1; idx >= 0; idx-- {
		reversed = append(reversed, e.unknownApplied[idx])
	}

	for _, u := range reversed {
		down := &ParsedMigration{}
		err := json.Unmarshal(*u.Down, down)
		if err != nil {
			e.log.Error("failed to unmarshal down migration", field("error", err), field("migration_id", u.ID))
			return err
		}

		e.log.Info("using parsed down migration to rollback", field("migration_id", u.ID))

		err = e.migrateWithSqlStatements(down, func(execDestination DBTarget) error {
			err := e.driverAsDatabaseDriver.DeleteMigration(u.ID, execDestination)
			if err != nil {
				e.log.Error("failed to delete migration meta entry, although down migration succeeded before",
					field("error", err),
					field("migration_id", u.ID))
				return err
			}
			e.log.Debug("deleted meta entry successful", field("migration_id", u.ID))
			return nil
		})
		if err != nil {
			e.log.Error("failed to migrate down", field("error", err))
			return err
		}

		// delete the migration we performed a rollback from the applied list
		for i := range e.applied {
			if e.applied[i].ID == u.ID {
				e.applied = append(e.applied[:i], e.applied[i+1:]...)
			}
		}

		e.log.Info("down migration successful", field("migration_id", u.ID))
	}

	e.log.Info("rollback successful")
	return nil
}

func allUnknownProvideParsedDown(unknown []*Migration, log logger.Logger) bool {
	ok := true
	for _, u := range unknown {
		if u.Down == nil {
			ok = false
			log.Warn("found migration without Down field", field("migration_id", u.ID))
		}
	}
	return ok
}
