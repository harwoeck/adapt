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
- 🟢 **Zero** external dependencies*

<sub>* <i>adapt</i> includes <a href="https://pkg.go.dev/github.com/harwoeck/liblog/contract">liblog/contract</a> (a public contract - Go <code>interface</code> - for logging backends, so you can provide your own structured, leveled logging)

#### Supported Storage `Driver`

- [File]() - Basic driver that stores migration meta-data in a local JSON file (demonstrates how a `Driver` without any reliance or dependency on `database/sql` can be written.)
- [MySQL / MariaDB]()
- [SQLite]()
- [PostgreSQL]()
- [Add driver ?](https://github.com/harwoeck/adapt/issues/new)

**Any other storage backend** by providing your own [`Driver`](), [`DatabaseDriver`]() or [`SqlStatementsDriver`](). Unlike most other migration tools, with _adapt_ there is no reliance on `database/sql` (such a case can be seen with the included `FileDriver`)

#### Supported Migrations

- [Go Code]()
- [Filesystem]()
- [In-memory]()
- [Embedded Filesystem]() - Using Go 1.16+ [go:embed](https://golang.org/pkg/embed/)

> Please support this project and provide additional sources that could be useful for other people

### Install

```bash
$ go get github.com/harwoeck/adapt/core
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

**Next example:** Due to compliance rules you decide encrypt the email address of your users inside your database. Using _adapt_ you simply provide one of `adapt.Hook`'s callback functions and during your next deployment _adapt_ will notice that this migration hasn't been applied and therefore call your migration hook. When no error is returned _adapt_ will commit the transaction and update the schema table with the relevant meta information.

```go
err := adapt.Migrate(
    "backend@v0.1.17",                      // <name> of executor
    adapt.NewMySQLDriver(db),               // Database driver
    adapt.SourceCollection{
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
