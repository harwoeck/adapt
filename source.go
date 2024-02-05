package adapt

import "log/slog"

// Source is the basis interface for every single migration-source. It provides
// information about the available migrations via ListMigrations. Every Source
// must additionally implement SqlStatementsSource or HookSource.
type Source interface {
	// Init should initialize everything inside the Source. If an error is returned
	// adapt will abort execution and stop with an Source failure.
	Init(log *slog.Logger) error
	// ListMigrations should list unique migration-IDs for all available migrations.
	// If a particular migration supports Up and Down variants only a single-ID (with
	// the Up/Down differentiator removed) should be returned. The IDs don't need to
	// be in any particular order, as adapt will merge it will other Source providers
	// and sort the complete SourceCollection afterwards
	ListMigrations() ([]string, error)
}

// SourceCollection is a collection of Source elements
type SourceCollection []Source

// SqlStatementsSource is a Source interface that provides parsed SQL-Statements
// via the ParsedMigration struct. It can provide Up (GetParsedUpMigration) or
// Down (GetParsedDownMigration) migrations for the same migration id.
type SqlStatementsSource interface {
	Source
	// GetParsedUpMigration must return a ParsedMigration for the passed ID. The ID
	// is always an element from the list returned from Source.ListMigrations. When
	// no custom parser is needed one can use Parse which implements a general
	// purpose parser for most SQL styles.
	GetParsedUpMigration(id string) (*ParsedMigration, error)
	// GetParsedDownMigration can return a ParsedMigration or nil (if no Down migration
	// is available) for the passed ID. The ID is always an element from the list
	// returned from Source.ListMigrations. When no custom parser is needed one can use
	// Parse which implements a general purpose parser for most SQL styles.
	GetParsedDownMigration(id string) (*ParsedMigration, error)
}

// HookSource provides migrations via a callback Hook object. Adapt will manage the
// migration meta-information and callback to the Hook when the migration needs to be
// executed. If the current Driver is an DatabaseDriver uses can even outsource the
// sql.Tx lifecycle management (Begin/Commit/Rollback) to adapt.
type HookSource interface {
	Source
	// GetHook must return the Hook object for the passed ID. The ID is always an
	// element from the list returned from Source.ListMigrations.
	GetHook(id string) Hook
}
