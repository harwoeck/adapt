package core

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ParsedMigration is a parsed migration
type ParsedMigration struct {
	UseTx bool     `json:"UseTransaction"`
	Stmts []string `json:"Statements"`
}

// Hash calculates a unique hash for the ParsedMigration. It includes the UseTx
// field and every single statement from the Stmts field
func (m *ParsedMigration) Hash() *string {
	hash := sha256.New()
	hash.Write([]byte(strconv.FormatBool(m.UseTx)))
	for _, stmt := range m.Stmts {
		// hash.Write never returns an error as to it's documentation
		_, _ = hash.Write([]byte(stmt))
	}
	hashStr := hex.EncodeToString(hash.Sum([]byte{}))
	return &hashStr
}

// Parse scans everything from an io.Reader into a ParsedMigration structure, while
// preserving SQL-specific structures like multi-line statements (procedures). It
// also checks for special "-- +adapt" options at the beginning of the file, like
// "NoTransaction".
//
// The following example should give you an overview how Parse works. Given the
// following file-content:
//
//     -- +adapt NoTransaction
//     CREATE DATABASE IF NOT EXISTS testdb
//         CHARACTER SET utf8mb4
//         COLLATE utf8mb4_unicode_ci;
//
//     CREATE TABLE testdb.accounts_old (id INT NOT NULL, PRIMARY KEY (id));
//     CREATE TABLE testdb.accounts_new (id INT NOT NULL, PRIMARY KEY (id));
//
//     -- +adapt BeginStatement
//     CREATE TRIGGER `accounts_trigger` BEFORE UPDATE ON `testdb.accounts_old` FOR EACH ROW BEGIN
//         INSERT INTO testdb.accounts_new (id) VALUES(OLD.id)
//     END
//     -- +adapt EndStatement
//
//     INSERT INTO testdb.accounts_old (id) VALUES(1); INSERT INTO testdb.accounts_old (id) VALUES(2);
//
// Parse would create the following ParsedMigration:
//
//     &ParsedMigration{
//         UseTx: false,
//         Stmts: []string{
//             "CREATE DATABASE IF NOT EXISTS testdb\n    CHARACTER SET utf8mb4\n    COLLATE utf8mb4_unicode_ci;",
//             "CREATE TABLE testdb.accounts_old (id INT NOT NULL, PRIMARY KEY (id));",
//             "CREATE TABLE testdb.accounts_new (id INT NOT NULL, PRIMARY KEY (id));",
//             "CREATE TRIGGER `accounts_trigger` BEFORE UPDATE ON `testdb.accounts_old` FOR EACH ROW BEGIN\n    INSERT INTO testdb.accounts_new (id) VALUES(OLD.id)\nEND",
//             "INSERT INTO testdb.accounts_old (id) VALUES(1);",
//             "INSERT INTO testdb.accounts_old (id) VALUES(2);",
//         },
//     }
func Parse(r io.Reader) (*ParsedMigration, error) {
	p := &ParsedMigration{
		UseTx: true,
		Stmts: []string{},
	}

	scanner := bufio.NewScanner(r)
	scanner.Split(scanLines)

	var buf strings.Builder
	var inStatement bool

	for scanner.Scan() {
		line := scanner.Text()
		line = dropCR(line)
		trimmedLine := strings.TrimSpace(line)

		// skip all empty lines when we aren't in a statement block
		if !inStatement && len(trimmedLine) == 0 {
			continue
		}

		cmdPrefix := "-- +adapt "
		if strings.HasPrefix(trimmedLine, cmdPrefix) {
			switch option := strings.TrimPrefix(trimmedLine, cmdPrefix); option {
			case "NoTransaction":
				if len(p.Stmts) > 0 || buf.Len() > 0 {
					return nil, fmt.Errorf("adapt/core.Parse: NoTransaction option must be in the first line of the file")
				}
				p.UseTx = false
			case "BeginStatement":
				inStatement = true
			case "EndStatement":
				p.Stmts = append(p.Stmts, buf.String())
				buf.Reset()
				inStatement = false
			default:
				return nil, fmt.Errorf("adapt/core.Parse: unknown option at start of line: %q", option)
			}
		} else {
			// when we are in a statement just write everything to the current buffer
			if inStatement || !strings.ContainsRune(line, ';') {
				_, _ = buf.WriteString(line) // error is always nil according to Go documentation
			} else {
				split := strings.SplitAfter(line, ";")

				// add first element to buffer and finish this statement, as it's suffixed with a semicolon
				_, _ = buf.WriteString(split[0]) // error is always nil according to Go documentation
				p.Stmts = append(p.Stmts, buf.String())
				buf.Reset()

				// write all non-empty split elements, except the first and last
				if len(split) > 2 {
					for _, part := range split[1 : len(split)-1] {
						if len(strings.TrimSpace(part)) > 0 {
							p.Stmts = append(p.Stmts, part)
						}
					}
				}

				// add last split element to buffer, as it's not suffixed with a semicolon
				last := split[len(split)-1]
				if len(strings.TrimSpace(last)) > 0 {
					_, _ = buf.WriteString(last) // error is always nil according to Go documentation
				}
			}
		}
	}

	// finish buffer as last statement if non-empty
	if buf.Len() > 0 && len(strings.TrimSpace(buf.String())) > 0 {
		p.Stmts = append(p.Stmts, buf.String())
		buf.Reset()
	}

	// trim space around all finished statements
	for i, s := range p.Stmts {
		p.Stmts[i] = strings.TrimSpace(s)
	}

	return p, nil
}

func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[0 : i+1], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func dropCR(data string) string {
	l := len(data)
	if l > 0 && data[l-1] == '\r' {
		return data[:l-1]
	}
	return data
}
