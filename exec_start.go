package adapt

import (
	"log/slog"
)

func (e *exec) stageStart() error {
	e.log.Debug("start")

	// compare local against store
	unknown, err := unknownAppliedMigrations(e.applied, e.available, !e.optDisableHashIntegrityChecks, e.log)
	if err != nil {
		return err
	}
	e.unknownApplied = unknown

	// branch between rollback and migrate
	if len(e.unknownApplied) > 0 {
		e.log.Debug("found unknown migrations. Starting with rollback protocol", "unknown_migrations", len(e.unknownApplied))
		err = e.stageRollback()
		if err != nil {
			e.log.Error("rollback procedure failed", "error", err)
			return err
		}
	} else {
		e.log.Debug("all stored migrations are known. Continuing with migration")
	}

	return e.stageMigrate()
}

func unknownAppliedMigrations(applied []*Migration, available []*AvailableMigration, performHashIntegrityChecks bool, log *slog.Logger) ([]*Migration, error) {
	searchLocal := func(id string) *AvailableMigration {
		for _, local := range available {
			if local.ID == id {
				return local
			}
		}
		return nil
	}

	hashEqual := func(h1 *string, h2 *string) bool {
		return h1 == nil || h2 == nil || *h1 == *h2
	}

	var unknown []*Migration
	for _, a := range applied {
		local := searchLocal(a.ID)
		if local == nil {
			unknown = append(unknown, a)
			continue
		}

		if performHashIntegrityChecks && !hashEqual(a.Hash, local.Hash) {
			log.Error("hash of local migration changed. Aborting to protect integrity, as changes to already applied scripts aren't allowed",
				"migration_id", a.ID, "local_hash", *local.Hash, "storage_hash", *a.Hash)
			return nil, ErrIntegrityProtection
		}

		if len(unknown) > 0 {
			log.Error("found known migration AFTER an unknown one. Aborting to " +
				"protect integrity, because unknown migrations (and their eventual rollbacks) must be at the " +
				"end of applied migrations. This ensures rollbacks are clean and don't interfere with migrations " +
				"that depend on these changes.")
			return nil, ErrIntegrityProtection
		}
	}

	return unknown, nil
}
