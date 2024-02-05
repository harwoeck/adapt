package adapt

import (
	"io"
	"strings"
)

type memoryFSSource struct {
	fs map[string]string
}

type memoryFSEntry struct {
	name string
}

func (e *memoryFSEntry) IsDir() bool  { return false }
func (e *memoryFSEntry) Name() string { return e.name }

func (a *memoryFSSource) ReadDir(_ string) ([]DirEntry, error) {
	wrapped := make([]DirEntry, 0)
	for name := range a.fs {
		wrapped = append(wrapped, &memoryFSEntry{name})
	}
	return wrapped, nil
}

func (a *memoryFSSource) Open(name string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(a.fs[name])), nil
}

// NewMemoryFSSource provides a SqlStatementsSource for an in-memory filesystem
// represented by a Name->FileContent map
func NewMemoryFSSource(fs map[string]string) SqlStatementsSource {
	return FromFilesystemAdapter(&memoryFSSource{fs}, "")
}
