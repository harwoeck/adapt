package core

import (
	logger "github.com/harwoeck/liblog/contract"
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
		e.log.Debug("found unknown migrations. Starting with rollback protocol", field("unknown_migrations", len(e.unknownApplied)))
		err = e.stageRollback()
		if err != nil {
			return e.log.ErrorReturn("rollback procedure failed", field("error", err))
		}
	} else {
		e.log.Debug("all stored migrations are known. Continuing with migration")
	}

	return e.stageMigrate()
}

func unknownAppliedMigrations(applied []*Migration, available []*AvailableMigration, performHashIntegrityChecks bool, log logger.Logger) ([]*Migration, error) {
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
			return nil, log.ErrorReturn("hash of local migration changed. Aborting to protect integrity, as changes to already applied scripts aren't allowed",
				field("migration_id", a.ID), field("local_hash", *local.Hash), field("storage_hash", *a.Hash))
		}

		if len(unknown) > 0 {
			return nil, log.ErrorReturn("found known migration AFTER an unknown one. Aborting to " +
				"protect integrity, because unknown migrations (and their eventual rollbacks) must be at the " +
				"end of applied migrations. This ensures rollbacks are clean and don't interfere with migrations " +
				"that depend on these changes.")
		}
	}

	return unknown, nil
}
