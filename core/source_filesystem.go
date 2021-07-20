package core

import (
	"io"
	"io/ioutil"
	"os"
)

type filesystemSource struct {
}

func (a *filesystemSource) ReadDir(name string) ([]DirEntry, error) {
	entries, err := ioutil.ReadDir(name)
	wrapped := make([]DirEntry, len(entries))
	for i, e := range entries {
		wrapped[i] = DirEntry(e)
	}
	return wrapped, err
}

func (a *filesystemSource) Open(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

// NewFilesystemSource provides a new SqlStatementsSource that uses the SQL-files
// within the passed directory as migrations.
func NewFilesystemSource(directory string) SqlStatementsSource {
	return FromFilesystemAdapter(&filesystemSource{}, directory)
}
