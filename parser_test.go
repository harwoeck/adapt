package adapt

import (
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    *ParsedMigration
		wantErr bool
	}{
		{"All features", args{strings.NewReader(`
-- +adapt NoTransaction
CREATE DATABASE IF NOT EXISTS testdb
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

CREATE TABLE testdb.accounts_old (id INT NOT NULL, PRIMARY KEY (id));
CREATE TABLE testdb.accounts_new (id INT NOT NULL, PRIMARY KEY (id));

-- +adapt BeginStatement
CREATE TRIGGER ` + "`accounts_trigger`" + ` BEFORE UPDATE ON ` + "`testdb.accounts_old`" + ` FOR EACH ROW BEGIN
    INSERT INTO testdb.accounts_new (id) VALUES(OLD.id)
END
-- +adapt EndStatement

INSERT INTO testdb.accounts_old (id) VALUES(1); INSERT INTO testdb.accounts_old (id) VALUES(2); INSERT INTO testdb.accounts_old (id) VALUES(3)
`)}, &ParsedMigration{
			UseTx: false,
			Stmts: []string{
				"CREATE DATABASE IF NOT EXISTS testdb\n    CHARACTER SET utf8mb4\n    COLLATE utf8mb4_unicode_ci;",
				"CREATE TABLE testdb.accounts_old (id INT NOT NULL, PRIMARY KEY (id));",
				"CREATE TABLE testdb.accounts_new (id INT NOT NULL, PRIMARY KEY (id));",
				"CREATE TRIGGER `accounts_trigger` BEFORE UPDATE ON `testdb.accounts_old` FOR EACH ROW BEGIN\n    INSERT INTO testdb.accounts_new (id) VALUES(OLD.id)\nEND",
				"INSERT INTO testdb.accounts_old (id) VALUES(1);",
				"INSERT INTO testdb.accounts_old (id) VALUES(2);",
				"INSERT INTO testdb.accounts_old (id) VALUES(3)",
			},
		}, false},
		{"Option unknown", args{strings.NewReader(`
-- +adapt UnknownInvalidOption
CREATE DATABASE IF NOT EXISTS testdb;`)}, nil, true},
		{"Option NoTransaction not in first line", args{strings.NewReader(`
CREATE DATABASE IF NOT EXISTS testdb;
-- +adapt NoTransaction`)}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() got = %v, want %v", got, tt.want)
			}
		})
	}
}
