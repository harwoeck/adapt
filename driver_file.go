package adapt

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"time"
)

// FileDriverOption provides configuration values for a Driver implemented using
// on-disk file storage for meta-data.
type FileDriverOption func(*fileDriver) error

// FileDriverFilePermission sets the file permission used when writing files to
// disk. By default 0600 ist used.
func FileDriverFilePermission(perm os.FileMode) FileDriverOption {
	return func(driver *fileDriver) error {
		driver.optFilePermission = perm
		return nil
	}
}

// NewFileDriver returns a Driver from a filename and variadic FileDriverOption that
// can interact with local JSON-file as storage for meta information.
func NewFileDriver(filename string, opts ...FileDriverOption) Driver {
	return &fileDriver{
		filename:          filename,
		opts:              opts,
		optFilePermission: 0600,
	}
}

type fileDriver struct {
	filename          string
	opts              []FileDriverOption
	optFilePermission os.FileMode
	log               *slog.Logger
}

func (d *fileDriver) Name() string {
	return "driver_file"
}

func (d *fileDriver) Init(log *slog.Logger) error {
	d.log = log

	for _, opt := range d.opts {
		err := opt(d)
		if err != nil {
			d.log.Error("init failed due to option error", "error", err)
			return err
		}
	}

	return nil
}

type fileDriverStorage struct {
	Migrations []*Migration
}

func (d *fileDriver) readStorage() (*fileDriverStorage, error) {
	f, err := os.Open(d.filename)
	if err != nil && !os.IsNotExist(err) {
		d.log.Error("failed to open file descriptor", "filename", d.filename, "error", err)
		return nil, err
	}
	if os.IsNotExist(err) {
		return &fileDriverStorage{Migrations: []*Migration{}}, nil
	}
	defer func() {
		_ = f.Close()
	}()

	s := &fileDriverStorage{}
	err = json.NewDecoder(f).Decode(s)
	if err != nil {
		d.log.Error("failed to decode fileDriverStorage into memory structure", "filename", d.filename, "error", err)
		return nil, err
	}

	// sort the ordering of our migrations
	sort.Slice(s.Migrations, func(i, j int) bool {
		return s.Migrations[i].ID < s.Migrations[j].ID
	})

	return s, nil
}

func (d *fileDriver) writeStorage(s *fileDriverStorage) error {
	f, err := os.OpenFile(d.filename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, d.optFilePermission)
	if err != nil {
		d.log.Error("failed to open file descriptor", "filename", d.filename, "error", err)
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	buf, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		d.log.Error("failed to encode memory structure into json buffer", "error", err)
		return err
	}

	_, err = f.Write(buf)
	if err != nil {
		d.log.Error("failed to write encoded json buffer", "error", err)
		return err
	}

	return nil
}

func (d *fileDriver) Healthy() error {
	// check if filename exists
	_, err := os.Stat(d.filename)
	if err != nil && !os.IsNotExist(err) {
		d.log.Error("cannot stat filename. unknown error", "filename", d.filename, "error", err)
		return err
	}

	// file doesn't exist at the moment - will be created later
	if os.IsNotExist(err) {
		d.log.Debug("file doesn't exist. will be created on first write", "filename", d.filename)
		return nil
	}

	// read storage to check if we can decode (unmarshal) it.
	_, err = d.readStorage()
	if err != nil {
		return err
	}

	return nil
}

func (d *fileDriver) SupportsLocks() bool {
	// TODO: copy lockedfile package from go's "go" command and enable locking for basic driver
	// https://pkg.go.dev/cmd/go/internal/lockedfile
	// https://pkg.go.dev/cmd/go/internal/lockedfile/internal/filelock
	return false
}

func (d *fileDriver) AcquireLock() error {
	d.log.Error("not supported")
	panic("not supported")
}

func (d *fileDriver) ReleaseLock() error {
	d.log.Error("not supported")
	panic("not supported")
}

func (d *fileDriver) ListMigrations() ([]*Migration, error) {
	s, err := d.readStorage()
	if err != nil {
		return nil, err
	}

	return s.Migrations, nil
}

func (d *fileDriver) AddMigration(migration *Migration) error {
	s, err := d.readStorage()
	if err != nil {
		return err
	}

	for _, item := range s.Migrations {
		if item.ID == migration.ID {
			d.log.Error("migration already exists", "migration_id", migration.ID)
			return fmt.Errorf("adapt.fileDriver: migration duplication")
		}
	}

	s.Migrations = append(s.Migrations, migration)
	return d.writeStorage(s)
}

func (d *fileDriver) SetMigrationToFinished(migrationID string) error {
	s, err := d.readStorage()
	if err != nil {
		return err
	}

	var set bool
	for _, item := range s.Migrations {
		if item.ID == migrationID {
			now := time.Now().UTC()
			item.Finished = &now
			set = true
			break
		}
	}
	if !set {
		d.log.Error("migration not found", "migration_id", migrationID)
		return fmt.Errorf("adapt.fileDriver: migration missing")
	}

	return d.writeStorage(s)
}

func (d *fileDriver) Close() error {
	return nil
}
