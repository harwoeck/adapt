package core

import (
	"time"

	logger "github.com/harwoeck/liblog/contract"
)

// Migration is a object containing meta-information of an applied migration
type Migration struct {
	// ID is the unique identifier of this Migration
	ID string
	// Executor is the name of the program that run this migration. Usually
	// this should be combination of name and version like "myService@v1.17.0"
	Executor string
	// Started is the time this Migration was started
	Started time.Time
	// Finished is the time this Migrations was finished. When nil the Migration
	// hasn't finished or errored
	Finished *time.Time
	// Hash contains the calculated hash identifier of this migration's content.
	// It is calculated if this Migration associated Source provides a Hash
	// function, like ParsedMigration does
	Hash *string
	// Adapt is the version string of adapt itself. The information is embedded
	// into this module with the public Version field.
	Adapt string
	// Deployment is a unique identifier that groups together multiple migrations
	// that have been executed within the same deployment cycle.
	Deployment string
	// DeploymentOrder is the order in which migrations within a Deployment group
	// were executed.
	DeploymentOrder int
	// Down can contain a json-marshaled ParsedMigration that can be used to
	// rollback this migration.
	Down *[]byte
}

// AvailableMigration is a container for a locally found migration that could be
// applied to the database. In it's base-form it consists of a ID and a Source
// element. When calling Enrich the type of Source is checked and additional
// information added
type AvailableMigration struct {
	// ID is the unique identifier of this AvailableMigration
	ID string
	// Source is the origin of this AvailableMigration
	Source Source
	// ParsedUp is a ParsedMigration set by Enrich if the Source is a
	// SqlStatementsSource
	ParsedUp *ParsedMigration
	// Hash is the unique migration hash from ParsedMigration.Hash set by Enrich
	// if the Source is a SqlStatementsSource
	Hash *string
}

// Enrich checks the type of Source and adds further information to the
// AvailableMigration, like ParsedUp and Hash for SqlStatementsSource
func (m *AvailableMigration) Enrich(log logger.Logger) error {
	switch src := m.Source.(type) {
	case SqlStatementsSource:
		// parse migration from Source
		parsed, err := src.GetParsedUpMigration(m.ID)
		if err != nil {
			log.Warn("failed to get parsed migration from SqlStatementsSource",
				field("migration_id", m.ID))
			return err
		}

		m.ParsedUp = parsed
		m.Hash = parsed.Hash()
	}
	return nil
}
