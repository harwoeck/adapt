package adapt

import "log/slog"

type codeSource struct {
	m    map[string]Hook
	list []string
}

// NewCodeSource provides a new HookSource for the single id-hook pair passed
// to it.
func NewCodeSource(id string, hook Hook) HookSource {
	return NewCodePackageSource(map[string]Hook{id: hook})
}

// NewCodePackageSource provides a new HookSource for the map of id-hook pairs
// passed to it.
func NewCodePackageSource(pkg map[string]Hook) HookSource {
	src := &codeSource{m: pkg}
	for key := range pkg {
		src.list = append(src.list, key)
	}
	return src
}

func (src *codeSource) Init(_ *slog.Logger) error {
	return nil
}

func (src *codeSource) ListMigrations() ([]string, error) {
	return src.list, nil
}

func (src *codeSource) GetHook(id string) Hook {
	return src.m[id]
}
