package core

const (
	// Version is the package's version string used to store in meta tables
	Version = "adapt@v0.1.0"
)

/*
Migrate migrates all available migrations in your SourceCollection against the
Driver, when they weren't already run before.

Example:
	var db *sql.DB = getDB()

	err := adapt.Migrate(
		"myService@v1.3.12",
		adapt.NewMySQLDriver(db,
			adapt.MySQLDriverOptionTableName("_auth_migrations"),
			adapt.MySQLDriverOptionDisableTx,
		),
		adapt.SourceCollection{
			adapt116.NewEmbedFSSource(nil, ""),
			adapt.NewMemoryFSSource(map[string]string{}),
			adapt.NewCodeSource("x", adapt.Hook{
				MigrateUpTx: func(tx *sql.Tx) error {
					return nil
				},
			}),
		},
	)
*/
func Migrate(executor string, driver Driver, sources SourceCollection, options ...Option) error {
	e, err := newExec(executor, driver, sources, options...)
	if err != nil {
		return err
	}
	return e.run()
}
