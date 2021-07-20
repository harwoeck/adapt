package core

import (
	"io"
	"path"
	"strings"

	logger "github.com/harwoeck/liblog/contract"
)

// DirEntry is similar to the fs.DirEntry interface provided by Go 1.16+. It's
// used by adapt to remove the Go 1.16 requirement and allow implementations
// like NewMemoryFSSource to provide a minimal directory entry object.
type DirEntry interface {
	IsDir() bool
	Name() string
}

// FilesystemAdapter is a minimal interface for any filesystem. It can be used
// in combination with FromFilesystemAdapter to convert the FilesystemAdapter
// to a full SqlStatementsSource
type FilesystemAdapter interface {
	ReadDir(name string) ([]DirEntry, error)
	Open(name string) (io.ReadCloser, error)
}

type fsAdapter struct {
	log       logger.Logger
	adapter   FilesystemAdapter
	directory string
	fsMap     map[string]string
	fsList    []string
}

func (src *fsAdapter) Init(log logger.Logger) error {
	src.log = log

	entries, err := src.adapter.ReadDir(src.directory)
	if err != nil {
		return log.ErrorReturn("unable to read directory content", field("error", err))
	}

	filterMap := make(map[string]struct{})

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		id := e.Name()
		if strings.HasSuffix(id, ".sql") {
			id = strings.TrimSuffix(id, ".sql")
		}

		if strings.HasSuffix(id, ".up") {
			filterMap[strings.TrimSuffix(id, ".up")] = struct{}{}
		} else if strings.HasSuffix(id, ".down") {
			filterMap[strings.TrimSuffix(id, ".down")] = struct{}{}
		} else {
			return log.ErrorReturn("migration with invalid id. Doesn't have '.up'/'.up.sql' or '.down'/'.down.sql' suffix",
				field("migration_id", id), field("filename", e.Name()))
		}

		src.fsMap[id] = path.Join(src.directory, e.Name())
	}

	// generate list of map keys
	for key := range filterMap {
		src.fsList = append(src.fsList, key)
	}

	return nil
}

func (src *fsAdapter) ListMigrations() ([]string, error) {
	return src.fsList, nil
}

func (src *fsAdapter) get(id, filename string) (*ParsedMigration, error) {
	f, err := src.adapter.Open(filename)
	if err != nil {
		return nil, src.log.ErrorReturn("unable to open file",
			field("error", err), field("id", id), field("filename", filename))
	}
	defer func() {
		_ = f.Close()
	}()

	return Parse(f)
}

func (src *fsAdapter) GetParsedUpMigration(id string) (*ParsedMigration, error) {
	if filename, ok := src.fsMap[id+".up"]; ok {
		return src.get(id, filename)
	}

	return nil, src.log.ErrorReturn("unable to find up migration for id", field("id", id))
}

func (src *fsAdapter) GetParsedDownMigration(id string) (*ParsedMigration, error) {
	if filename, ok := src.fsMap[id+".down"]; ok {
		return src.get(id, filename)
	}
	return nil, nil
}

// FromFilesystemAdapter converts an FilesystemAdapter implementation to a
// full-fledged SqlStatementsSource. It unifies the code across most filesystem
// and the in-memory statements sources.
func FromFilesystemAdapter(adapter FilesystemAdapter, directory string) SqlStatementsSource {
	return &fsAdapter{
		adapter:   adapter,
		directory: directory,
		fsMap:     make(map[string]string),
	}
}
