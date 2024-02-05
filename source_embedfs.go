package adapt

import (
	"embed"
	"io"
	"io/fs"
)

type embedFSSource struct {
	fs embed.FS
}

func (a *embedFSSource) ReadDir(name string) ([]DirEntry, error) {
	entries, err := fs.ReadDir(a.fs, name)
	wrapped := make([]DirEntry, len(entries))
	for i, e := range entries {
		wrapped[i] = DirEntry(e)
	}
	return wrapped, err
}

func (a *embedFSSource) Open(name string) (io.ReadCloser, error) {
	return a.fs.Open(name)
}

// NewEmbedFSSource provides a new SqlStatementsSource that uses the SQL-files
// within the passed embedded FS (embed.FS) as migrations.
func NewEmbedFSSource(fs embed.FS, directory string) SqlStatementsSource {
	return FromFilesystemAdapter(&embedFSSource{fs}, directory)
}
