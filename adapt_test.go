package adapt

import (
	"os"
	"testing"
)

func ensureFileIsDeleted(filename string) {
	_ = os.Remove(filename)
}

func TestMigrate(t *testing.T) {
	filename := "test.json"
	ensureFileIsDeleted(filename)
	defer ensureFileIsDeleted(filename)

	var (
		called       int
		init         bool
		initTime     int
		migrated     bool
		migratedTime int
		undoInit     bool
		undoInitTime int
	)
	err := Migrate("adapt-tester@v1.1.7",
		NewFileDriver(filename),
		SourceCollection{
			NewCodePackageSource(map[string]Hook{
				"20201115_1717_undo-init": {
					MigrateUp: func() error {
						undoInit = true
						undoInitTime = called
						called++
						return nil
					},
				},
				"20201115_1214_init": {
					MigrateUp: func() error {
						init = true
						initTime = called
						called++
						return nil
					},
				},
			}),
			NewCodeSource("20201115_1215_init-code", Hook{
				MigrateUp: func() error {
					migrated = true
					migratedTime = called
					called++
					return nil
				},
			}),
		},
	)
	if err != nil {
		t.Errorf("not expected error: %v", err)
	}
	if !init {
		t.Errorf("never migrated init source")
	}
	if !migrated {
		t.Errorf("never migrated code source")
	}
	if !undoInit {
		t.Errorf("never migrated undoInit source")
	}
	if initTime != 0 || migratedTime != 1 || undoInitTime != 2 {
		t.Errorf("ordering of migration not correct")
	}

	basic := NewFileDriver(filename)
	listed, err := basic.ListMigrations()
	if err != nil {
		t.Errorf("not expected error")
	}
	if listed[0].ID != "20201115_1214_init" || listed[1].ID != "20201115_1215_init-code" || listed[2].ID != "20201115_1717_undo-init" {
		t.Errorf("ordering false")
	}
	if listed[0].Executor != "adapt-tester@v1.1.7" {
		t.Errorf("executor false")
	}
	if listed[0].Finished == nil {
		t.Errorf("finished not set")
	}
}

func TestMigrate_Error(t *testing.T) {
	type args struct {
		src SourceCollection
	}
	tests := []struct {
		name string
		args args
	}{
		{"Invalid CodeSource (empty Hook)", args{SourceCollection{NewCodeSource("ID", Hook{})}}},
		{"Invalid MemorySource (invalid id)", args{SourceCollection{NewMemoryFSSource(map[string]string{
			"invalid_id": "CREATE DATABASE test_db",
		})}}},
		{"Invalid MemorySource (only down)", args{SourceCollection{NewMemoryFSSource(map[string]string{
			"init.down.sql": "DROP DATABASE test_db",
		})}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := "test.json"
			ensureFileIsDeleted(filename)
			defer ensureFileIsDeleted(filename)

			err := Migrate("adapt-tester@v1.1.7",
				NewFileDriver(filename, FileDriverFilePermission(0600)),
				tt.args.src,
			)
			if err == nil {
				t.Errorf("Migrate() got nil error, but expected one")
			}
		})
	}
}
