package core

import (
	"sort"

	logger "github.com/harwoeck/liblog/contract"
)

func (e *exec) stagePrepareLocal() error {
	e.log.Debug("prepare local")

	// merge all sources into available migrations
	available, err := mergeSources(e.sources, e.log)
	if err != nil {
		return err
	}

	// save to exec
	e.available = available

	e.log.Info("prepare local successful")
	return nil
}

func mergeSources(sources SourceCollection, log logger.Logger) ([]*AvailableMigration, error) {
	migrationMap := make(map[string]*AvailableMigration)

	for _, src := range sources {
		migrations, err := src.ListMigrations()
		if err != nil {
			log.Error("listing migrations failed", field("error", err))
			return nil, err
		}

		for _, id := range migrations {
			// we must stop, because we cannot take the "same" migration from multiple
			// sources!
			if _, ok := migrationMap[id]; ok {
				return nil, log.ErrorReturn("migration was provided by multiple sources", field("migration_id", id))
			}

			// migration with this id isn't available -> add it
			am := &AvailableMigration{
				ID:     id,
				Source: src,
			}
			err = am.Enrich(log)
			if err != nil {
				return nil, err
			}

			migrationMap[id] = am
		}
	}

	// copy all migrations from map to slice
	migrationList := make([]*AvailableMigration, 0)
	for _, m := range migrationMap {
		migrationList = append(migrationList, m)
	}

	// sort the ordering of our migrations
	sort.Slice(migrationList, func(i, j int) bool {
		return migrationList[i].ID < migrationList[j].ID
	})

	log.Info("merged all sources into a single migration collection", field("sources_amount", len(sources)), field("migrations_amount", len(migrationList)))
	return migrationList, nil
}
