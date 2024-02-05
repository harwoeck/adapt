<img src="adapt-logo.png"></img>
<table>
  <tbody>
    <td align="center">
      <br>
      <i>adapt</i> is a simple, non-magical general purpose migration library that gets embedded into your application.<br>
      <img width="2000" height="0">
    </td>
  </tbody>
</table>

### Features

- 🟢 **Simplicity:** Migration lifecycle and schema versioning is completely abstracted and managed by _adapt_
- 🟢 **Concurrency controlled (protected from race-conditions):** Mutexes and other techniques are used to prevent multiple _adapt_ instances conflicting each other during concurrent boot-ups
- 🟢 **Extensible:** Simple `Driver` and migration `Source` interfaces
- 🟢 **Migrations:** can be provided in a variety of ways and get merged and sorted into a single migration-collection, which is applied against the version-controlled schema in your storage
    - **Go Code** for all non-database migrations or situations where SQL doesn't do the job
    - **Embed migration folders** containing your SQL scripts
    - Hardcoded SQL statements
- 🟢 **Branching and merging compatible:** _adapt_ automatically catches up with missing migrations ("holes")
- 🟢 **Zero** external dependencies
- 🟢 **Customizable Logging:** _adapt_ uses [`slog`](https://pkg.go.dev/log/slog) [included with Go 1.21+](https://go.dev/blog/slog) so that you can provide your own logging backend.

#### Supported Storage `Driver`

- [File](https://pkg.go.dev/github.com/harwoeck/adapt#NewFileDriver) - Basic driver that stores migration meta-data in a local JSON file (demonstrates how a `Driver` without any reliance or dependency on `database/sql` can be written.)
- [MySQL / MariaDB](https://pkg.go.dev/github.com/harwoeck/adapt#NewMySQLDriver)
- [SQLite](https://pkg.go.dev/github.com/harwoeck/adapt#NewSQLiteDriver)
- [PostgreSQL](https://pkg.go.dev/github.com/harwoeck/adapt#NewPostgresDriver)
- [Add driver ?](https://github.com/harwoeck/adapt/issues/new)

**Any other storage backend** by providing your own [`Driver`](https://pkg.go.dev/github.com/harwoeck/adapt#Driver), [`DatabaseDriver`](https://pkg.go.dev/github.com/harwoeck/adapt#DatabaseDriver) or [`SqlStatementsDriver`](https://pkg.go.dev/github.com/harwoeck/adapt#SqlStatementsDriver). Unlike most other migration tools, with _adapt_ there is no reliance on `database/sql` (such a case can be seen with the included `FileDriver`)

#### Supported Migrations

- [Go Code](https://pkg.go.dev/github.com/harwoeck/adapt#NewCodePackageSource)
- [Filesystem](https://pkg.go.dev/github.com/harwoeck/adapt#NewFilesystemSource)
- [In-memory](https://pkg.go.dev/github.com/harwoeck/adapt#NewMemoryFSSource)
- [Embedded Filesystem](https://pkg.go.dev/github.com/harwoeck/adapt#NewEmbedFSSource) - Using Go 1.16+ [go:embed](https://pkg.go.dev/embed)

> [!NOTE]
> Please support this project and provide additional sources that could be useful for other people

### Install

```bash
$ go get github.com/harwoeck/adapt
```

## Usage

```go
var db *sql.DB = initDB()

err := adapt.Migrate(
    "backend@v0.1.17",                      // <name> of executor
    adapt.NewMySQLDriver(db),               // Database driver
    adapt.SourceCollection{
        adapt.NewFilesystemSource("./sql"), // SQL-Migration-Scripts from filesystem
    })
```

> [!NOTE]
> **Next example:** Due to compliance rules you decide to encrypt your users email addresses inside your database. Since this requires actual Go code (and not just SQL statements), you could implement one of the [`adapt.Hook`](https://pkg.go.dev/github.com/harwoeck/adapt#Hook) functions and during your next deployment _adapt_ will notice this new unapplied migration and execute your hook. When no error is returned _adapt_ will commit the transaction and update the schema table with the relevant meta information.

```go
err := adapt.Migrate(
    "backend@v0.1.17",                      // <name> of executor
    adapt.NewMySQLDriver(db),               // Database driver
    adapt.SourceCollection{                 // adapt will automatically merge and sort all provided sources for you
        adapt.NewFilesystemSource("./sql"), // SQL-Migration-Scripts from filesystem
        adapt.NewCodeSource("2020-04-17_1104_encrypt-user-email", adapt.Hook{
            MigrateUpTx: func(tx *sql.Tx) error {
                // For this encrypt-migration you could use `MigrateUpTx` to load
                // the rows, encrypt the content in Go and update the columns again
                // within a single managed transaction.
                return nil
            },
        },
    })
```

## Inspired by

This project was heavily inspired by the features and ideas of these great projects:

- https://www.liquibase.org
- https://github.com/Boostport/migration
- https://github.com/lopezator/migrator
