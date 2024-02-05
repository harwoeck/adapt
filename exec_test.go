package adapt

import (
	"log/slog"
	"os"
	"reflect"
	"testing"
	"time"
)

func Test_mergeSources(t *testing.T) {
	type args struct {
		sources []Source
	}
	tests := []struct {
		name    string
		args    args
		want    []*AvailableMigration
		wantErr bool
	}{
		{"single id and source", args{[]Source{
			NewMemoryFSSource(map[string]string{
				"20201115_1214_init.up.sql": "CREATE DATABASE",
			}),
		}}, []*AvailableMigration{
			{ID: "20201115_1214_init"},
		}, false},
		{"multiple id and single source", args{[]Source{
			NewMemoryFSSource(map[string]string{
				"20201115_1214_init.up.sql":      "CREATE DATABASE",
				"20201115_1717_undo-init.up.sql": "DELETE DATABASE",
			}),
		}}, []*AvailableMigration{
			{ID: "20201115_1214_init"},
			{ID: "20201115_1717_undo-init"},
		}, false},
		{"multiple id and single source with sort", args{[]Source{
			NewMemoryFSSource(map[string]string{
				"20201115_1717_undo-init.up.sql": "DELETE DATABASE",
				"20201115_1214_init.up.sql":      "CREATE DATABASE",
			}),
		}}, []*AvailableMigration{
			{ID: "20201115_1214_init"},
			{ID: "20201115_1717_undo-init"},
		}, false},
		{"same id in multiple sources", args{[]Source{
			NewMemoryFSSource(map[string]string{
				"20201115_1717_undo-init.up.sql": "DELETE DATABASE",
				"20201115_1214_init.up.sql":      "CREATE DATABASE",
			}),
			NewMemoryFSSource(map[string]string{
				"20201115_1717_undo-init.up.sql": "DELETE DATABASE",
				"20201115_1214_init.up.sql":      "CREATE DATABASE",
			}),
		}}, []*AvailableMigration{}, true},
		{"multiple sources with sort", args{[]Source{
			NewMemoryFSSource(map[string]string{
				"20201115_1717_undo-init.up.sql": "DELETE DATABASE",
			}),
			NewMemoryFSSource(map[string]string{
				"20201115_1214_init.up.sql": "CREATE DATABASE",
			}),
		}}, []*AvailableMigration{
			{ID: "20201115_1214_init"},
			{ID: "20201115_1717_undo-init"},
		}, false},
		{"complex multiple sources with holes and sorting", args{[]Source{
			NewMemoryFSSource(map[string]string{
				"20201115_1717_undo-init.up.sql": "DELETE DATABASE",
			}),
			NewMemoryFSSource(map[string]string{
				"20201115_1214_init.up.sql":       "CREATE DATABASE",
				"20201115_1328_improve-db.up.sql": "CREATE DATABASE",
			}),
			NewCodeSource("20210508_2046_password-box-hashing", Hook{}),
			NewCodePackageSource(map[string]Hook{
				"20201115_1215_init-code":      {},
				"20201115_1215_init-code-2":    {},
				"20201115_1718_undo-init-code": {},
			}),
		}}, []*AvailableMigration{
			{ID: "20201115_1214_init"},
			{ID: "20201115_1215_init-code"},
			{ID: "20201115_1215_init-code-2"},
			{ID: "20201115_1328_improve-db"},
			{ID: "20201115_1717_undo-init"},
			{ID: "20201115_1718_undo-init-code"},
			{ID: "20210508_2046_password-box-hashing"},
		}, false},
		{"complex multiple sources with holes and sorting but error", args{[]Source{
			NewMemoryFSSource(map[string]string{
				"20201115_1717_undo-init.up.sql": "DELETE DATABASE",
			}),
			NewMemoryFSSource(map[string]string{
				"20201115_1214_init.up.sql":       "CREATE DATABASE",
				"20201115_1328_improve-db.up.sql": "CREATE DATABASE",
			}),
			NewCodeSource("20210508_2046_password-box-hashing", Hook{}),
			NewCodePackageSource(map[string]Hook{
				"20201115_1215_init-code":      {},
				"20201115_1214_init":           {},
				"20201115_1718_undo-init-code": {},
			}),
		}}, []*AvailableMigration{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := slog.New(slog.NewTextHandler(os.Stdout, nil))

			for _, s := range tt.args.sources {
				err := s.Init(l)
				if err != nil {
					t.Errorf("mergeSources() didn't expect src.Init() error = %v", err)
					return
				}
			}

			got, err := mergeSources(tt.args.sources, l)
			if (err != nil) != tt.wantErr {
				t.Errorf("mergeSources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for i, g := range got {
				if g.ID != tt.want[i].ID {
					t.Errorf("mergeSources() got[%d] = %v, want %v", i, got, tt.want[i].ID)
				}
			}
		})
	}
}

func Test_healthCheckAppliedMigration(t *testing.T) {
	timeAddr := time.Now()

	type args struct {
		applied []*Migration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"ok", args{
			applied: []*Migration{
				{
					ID:       "1",
					Finished: &timeAddr,
				},
			},
		}, false},
		{"multiple ok", args{
			applied: []*Migration{
				{
					ID:       "1",
					Finished: &timeAddr,
				},
				{
					ID:       "2",
					Finished: &timeAddr,
				},
			},
		}, false},
		{"null pointer", args{
			applied: []*Migration{
				{
					ID:       "1",
					Finished: nil,
				},
			},
		}, true},
		{"mixed", args{
			applied: []*Migration{
				{
					ID:       "1",
					Finished: &timeAddr,
				},
				{
					ID:       "2",
					Finished: nil,
				},
				{
					ID:       "3",
					Finished: &timeAddr,
				},
			},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := slog.New(slog.NewTextHandler(os.Stdout, nil))

			if err := healthCheckAppliedMigration(tt.args.applied, l); (err != nil) != tt.wantErr {
				t.Errorf("healthCheckAppliedMigration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_unknownAppliedMigrations(t *testing.T) {
	strPtr := func(s string) *string {
		return &s
	}

	type args struct {
		applied   []*Migration
		available []*AvailableMigration
		unknown   []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"equal", args{
			applied: []*Migration{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
			},
			available: []*AvailableMigration{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
			},
		}, false},
		{"db empty", args{
			applied: []*Migration{},
			available: []*AvailableMigration{
				{ID: "1"},
			},
		}, false},
		{"db behind available", args{
			applied: []*Migration{
				{ID: "1"},
			},
			available: []*AvailableMigration{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
			},
		}, false},
		{"db ahead of available", args{
			applied: []*Migration{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
			},
			available: []*AvailableMigration{
				{ID: "1"},
			},
			unknown: []string{"2", "3"},
		}, false},
		{"no migrations", args{
			applied:   []*Migration{},
			available: []*AvailableMigration{},
		}, false},
		{"hash mismatch", args{
			applied: []*Migration{
				{ID: "1", Hash: strPtr("DB_HASH")},
			},
			available: []*AvailableMigration{
				{ID: "1", Hash: strPtr("LOCAL_HASH")},
			},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := slog.New(slog.NewTextHandler(os.Stdout, nil))

			unknown, err := unknownAppliedMigrations(tt.args.applied, tt.args.available, true, l)
			if (err != nil) != tt.wantErr {
				t.Errorf("unknownAppliedMigrations() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(unknown) != len(tt.args.unknown) {
				t.Errorf("unknownAppliedMigrations() len(unknown) %v, wanted len %v", len(unknown), len(tt.args.unknown))
			}

			for i, u := range tt.args.unknown {
				if unknown[i].ID != u {
					t.Errorf("unknownAppliedMigrations() unknown[%d] = %v, want = %v", i, unknown[i].ID, u)
				}
			}
		})
	}
}

func Test_findNeededMigrations(t *testing.T) {
	type args struct {
		applied   []*Migration
		available []*AvailableMigration
	}
	tests := []struct {
		name string
		args args
		want []*AvailableMigration
	}{
		{"all applied", args{
			applied: []*Migration{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
			},
			available: []*AvailableMigration{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
			},
		}, []*AvailableMigration{}},
		{"db ahead of available", args{
			applied: []*Migration{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
				{ID: "4"},
				{ID: "5"},
			},
			available: []*AvailableMigration{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
			},
		}, []*AvailableMigration{}},
		{"db behind available", args{
			applied: []*Migration{
				{ID: "1"},
				{ID: "2"},
			},
			available: []*AvailableMigration{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
				{ID: "4"},
				{ID: "5"},
			},
		}, []*AvailableMigration{
			{ID: "3"},
			{ID: "4"},
			{ID: "5"},
		}},
		{"db empty", args{
			applied: []*Migration{},
			available: []*AvailableMigration{
				{ID: "1"},
				{ID: "2"},
			},
		}, []*AvailableMigration{
			{ID: "1"},
			{ID: "2"},
		}},
		{"db with merch hole", args{
			applied: []*Migration{
				{ID: "1"},
				{ID: "2"},
				{ID: "4"},
			},
			available: []*AvailableMigration{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
				{ID: "4"},
			},
		}, []*AvailableMigration{
			{ID: "3"},
		}},
		{"db behind available and with merch hole", args{
			applied: []*Migration{
				{ID: "1"},
				{ID: "2"},
				{ID: "4"},
			},
			available: []*AvailableMigration{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
				{ID: "4"},
				{ID: "5"},
				{ID: "7"},
			},
		}, []*AvailableMigration{
			{ID: "3"},
			{ID: "5"},
			{ID: "7"},
		}},
		{"with dates", args{
			applied: []*Migration{
				{ID: "20210110_1919_init.sql"},        // 1
				{ID: "20210113_2341_alter-table.sql"}, // 2
				{ID: "20210116_0712_add-users.sql"},   // 4
			},
			available: []*AvailableMigration{
				{ID: "20210110_1919_init.sql"},               // 1
				{ID: "20210113_2341_alter-table.sql"},        // 2
				{ID: "20210114_1012_improve-db.sql"},         // 3
				{ID: "20210116_0712_add-users.sql"},          // 4
				{ID: "20210418_0112_improve-users.sql"},      // 5
				{ID: "20210501_1114_add-analytics-user.sql"}, // 6
			},
		}, []*AvailableMigration{
			{ID: "20210114_1012_improve-db.sql"},         // 3
			{ID: "20210418_0112_improve-users.sql"},      // 5
			{ID: "20210501_1114_add-analytics-user.sql"}, // 6
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := slog.New(slog.NewTextHandler(os.Stdout, nil))

			if got := findNeededMigrations(tt.args.applied, tt.args.available, l); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findNeededMigrations() = %v, want %v", got, tt.want)
			}
		})
	}
}
