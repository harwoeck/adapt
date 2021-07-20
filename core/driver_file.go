package core

import (
	"encoding/json"
	"os"
	"sort"
	"time"

	logger "github.com/harwoeck/liblog/contract"
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
	log               logger.Logger
}

func (d *fileDriver) Name() string {
	return "driver_file"
}

func (d *fileDriver) Init(log logger.Logger) error {
	d.log = log.Named(d.Name())

	for _, opt := range d.opts {
		err := opt(d)
		if err != nil {
			return d.log.ErrorReturn("init failed due to option error", field("error", err))
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
		return nil, d.log.ErrorReturn("failed to open file descriptor",
			field("error", err), field("filename", d.filename))
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
		return nil, d.log.ErrorReturn("failed to decode fileDriverStorage into memory structure",
			field("error", err), field("filename", d.filename))
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
		return d.log.ErrorReturn("failed to open file descriptor",
			field("error", err), field("filename", d.filename))
	}
	defer func() {
		_ = f.Close()
	}()

	buf, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return d.log.ErrorReturn("failed to encode memory structure into json buffer", field("error", err))
	}

	_, err = f.Write(buf)
	if err != nil {
		return d.log.ErrorReturn("failed to write encoded json buffer", field("error", err))
	}

	return nil
}

func (d *fileDriver) Healthy() error {
	// check if filename exists
	_, err := os.Stat(d.filename)
	if err != nil && !os.IsNotExist(err) {
		return d.log.ErrorReturn("cannot stat filename. unknown error",
			field("error", err), field("filename", d.filename))
	}

	// file doesn't exist at the moment - will be created later
	if os.IsNotExist(err) {
		d.log.Debug("file doesn't exist. will be created on first write", field("filename", d.filename))
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
	d.log.DPanic("not supported")
	return nil
}

func (d *fileDriver) ReleaseLock() error {
	d.log.DPanic("not supported")
	return nil
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
			return d.log.ErrorReturn("migration already exists", field("migration_id", migration.ID))
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
		return d.log.ErrorReturn("migration not found", field("migration_id", migrationID))
	}

	return d.writeStorage(s)
}

func (d *fileDriver) Close() error {
	return nil
}
