package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	logger "github.com/harwoeck/liblog/contract"
)

func (e *exec) stageMigrate() error {
	e.log.Debug("migrate")

	// generate deployment ID
	dID, err := genDeploymentID()
	if err != nil {
		e.log.Error("failed to generate deployment id", field("error", err))
		return err
	}

	// find all needed migrations
	needed := findNeededMigrations(e.applied, e.available, e.log)
	if len(needed) == 0 {
		e.log.Info("all migrations already applied. everything up-to-date")
		return nil
	}

	// sequentially apply needed migrations
	for dOrder, migration := range needed {
		// convert all information to a Migration object
		meta, err := convertToMigration(migration, e.executor, dID, dOrder, e.log)
		if err != nil {
			return err
		}

		// apply migration
		err = e.migrate(migration, meta)
		if err != nil {
			return err
		}
	}

	e.log.Info("migrate successful")
	return nil
}

func genDeploymentID() (string, error) {
	buf := make([]byte, 12)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}

	str := hex.EncodeToString(buf)
	var (
		p1 = str[:6]
		p2 = str[6:12]
		p3 = str[12:18]
		p4 = str[18:]
	)
	return fmt.Sprintf("ADAPT-%s-%s-%s-%s", p1, p2, p3, p4), nil
}

func findNeededMigrations(applied []*Migration, available []*AvailableMigration, log logger.Logger) []*AvailableMigration {
	// if there aren't any applied just return all available
	if len(applied) == 0 {
		return available
	}

	// store all needed migrations
	needed := make([]*AvailableMigration, 0)

	dbIdx := 0
	for memIdx := 0; memIdx < len(available); memIdx++ {
		// migration at current moving index-positions are equal. Therefore this migration
		// was already applied
		if applied[dbIdx].ID == available[memIdx].ID {
			// move db index-position
			dbIdx++

			// database has new further migrations => add all "remaining" new migrations
			// into needed and stop
			if dbIdx == len(applied) {
				needed = append(needed, available[memIdx+1:]...)
				break
			}

			// continue next round (increase available index position and check if there
			// is one left)
			continue
		}

		// current migration in memory is not applied. This is a "hole" inside our db (most
		// often caused by merges). Increase memIdx until hole in db is closed.
		needed = append(needed, available[memIdx])
		log.Info("found migration hole. Adding local migrations until hole is closed", field("migration_id", available[memIdx].ID))
	}

	return needed
}

func convertToMigration(a *AvailableMigration, executor string, deployment string, deploymentOrder int, log logger.Logger) (*Migration, error) {
	meta := &Migration{
		ID:              a.ID,
		Executor:        executor,
		Started:         time.Now().UTC(),
		Adapt:           Version,
		Deployment:      deployment,
		DeploymentOrder: deploymentOrder,
	}

	if a.Hash != nil {
		meta.Hash = a.Hash
	}

	var parsed *ParsedMigration
	switch src := a.Source.(type) {
	case SqlStatementsSource:
		var err error
		parsed, err = src.GetParsedDownMigration(meta.ID)
		if err != nil {
			log.Error("failed to get parsed down migration", field("error", err))
			return nil, err
		}
	case HookSource:
		hook := src.GetHook(meta.ID)
		if hook.MigrateDown != nil {
			parsed = hook.MigrateDown()
		}
	}
	if parsed == nil {
		log.Debug("unable to find down migration for id", field("id", meta.ID))
	} else {
		buf, err := json.Marshal(parsed)
		if err != nil {
			log.Error("failed to json encode parsed down migration", field("error", err))
			return nil, err
		}

		meta.Down = &buf
	}

	return meta, nil
}

func (e *exec) migrate(migration *AvailableMigration, meta *Migration) (err error) {
	log := e.log.With(field("migration_id", migration.ID))

	defer func(started time.Time) {
		if err == nil {
			log.Debug("migration finished successfully after", field("took_duration", time.Now().Sub(started)))
		} else {
			log.Debug("migration errored after", field("took_duration", time.Now().Sub(started)))
		}
	}(time.Now())

	log.Info("applying migration", field("deployment", meta.Deployment), field("deployment_order", meta.DeploymentOrder))

	// add meta information that we started this migration
	err = e.driver.AddMigration(meta)
	if err != nil {
		return err
	}

	// switch between the source type of the migration
	switch src := migration.Source.(type) {
	case SqlStatementsSource:
		err = e.migrateWithSqlStatements(migration.ParsedUp, nil)
	case HookSource:
		err = e.migrateWithHook(meta.ID, src)
	}
	if err != nil {
		return err
	}

	// migration finished successful -> add label to store to signal that everything is ok
	return e.driver.SetMigrationToFinished(migration.ID)
}
