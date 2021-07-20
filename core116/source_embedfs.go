package core116

import (
	"embed"
	"io"
	"io/fs"

	"github.com/harwoeck/adapt/core"
)

type embedFSSource struct {
	fs embed.FS
}

func (a *embedFSSource) ReadDir(name string) ([]core.DirEntry, error) {
	entries, err := fs.ReadDir(a.fs, name)
	wrapped := make([]core.DirEntry, len(entries))
	for i, e := range entries {
		wrapped[i] = core.DirEntry(e)
	}
	return wrapped, err
}

func (a *embedFSSource) Open(name string) (io.ReadCloser, error) {
	return a.fs.Open(name)
}

// NewEmbedFSSource provides a new core.SqlStatementsSource that uses the SQL-files
// within the passed embedded FS (embed.FS) as migrations.
func NewEmbedFSSource(fs embed.FS, directory string) core.SqlStatementsSource {
	return core.FromFilesystemAdapter(&embedFSSource{fs}, directory)
}
