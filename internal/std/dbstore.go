package std

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// validFieldName restricts document-field and identifier names used to
// build SQL fragments (table names, index names, json_extract paths) to a
// safe character set. Values (row data) never go through this path — those
// are always bound as query parameters ("?"). This is defense in depth on
// top of sqlIdent's quote-escaping, so a field name can't smuggle SQL
// syntax through a json_extract path or an unquoted context.
var validFieldName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const dbDirName = ".lunex/data"

type sqliteStore struct {
	path string
	sql  *sql.DB
	mu   sync.Mutex
}

var storeRegistry = struct {
	mu     sync.Mutex
	stores map[string]*sqliteStore
}{stores: make(map[string]*sqliteStore)}

func resolveDBPath(name string) (string, error) {
	if name == "" {
		name = "default"
	}
	if filepath.Ext(name) != ".db" {
		name = name + ".db"
	}
	if filepath.IsAbs(name) {
		return name, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cwd, dbDirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

func openStore(name string) (*sqliteStore, error) {
	path, err := resolveDBPath(name)
	if err != nil {
		return nil, err
	}
	storeRegistry.mu.Lock()
	defer storeRegistry.mu.Unlock()
	if s, ok := storeRegistry.stores[path]; ok {
		return s, nil
	}
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: failed to open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS __lunex_meta__ (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("db: failed to initialize %s: %w", path, err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS __lunex_seqs__ (
		name TEXT PRIMARY KEY,
		value INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("db: failed to initialize %s: %w", path, err)
	}
	s := &sqliteStore{path: path, sql: db}
	storeRegistry.stores[path] = s
	return s, nil
}

func dropStoreFile(name string) error {
	path, err := resolveDBPath(name)
	if err != nil {
		return err
	}
	storeRegistry.mu.Lock()
	if s, ok := storeRegistry.stores[path]; ok {
		s.sql.Close()
		delete(storeRegistry.stores, path)
	}
	storeRegistry.mu.Unlock()
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		_ = os.Remove(path + suffix)
	}
	return nil
}

func listStoreFiles() []string {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	dir := filepath.Join(cwd, dbDirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) == ".db" {
			out = append(out, e.Name()[:len(e.Name())-len(".db")])
		}
	}
	return out
}

func sqlIdent(name string) string {
	escaped := strings.ReplaceAll(name, `"`, `""`)
	return `"` + escaped + `"`
}

func (s *sqliteStore) ensureTable(name string) error {
	if !validFieldName.MatchString(name) {
		return fmt.Errorf("db: invalid table name %q", name)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id TEXT PRIMARY KEY,
		doc TEXT NOT NULL
	)`, sqlIdent(name))
	_, err := s.sql.Exec(stmt)
	return err
}

func (s *sqliteStore) dropTable(name string) error {
	if !validFieldName.MatchString(name) {
		return fmt.Errorf("db: invalid table name %q", name)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.sql.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, sqlIdent(name)))
	return err
}

func (s *sqliteStore) clearTable(name string) error {
	if !validFieldName.MatchString(name) {
		return fmt.Errorf("db: invalid table name %q", name)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.sql.Exec(fmt.Sprintf(`DELETE FROM %s`, sqlIdent(name)))
	return err
}

func (s *sqliteStore) listTables() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.sql.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE '__lunex_%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

func (s *sqliteStore) insertRow(table, id string, docJSON []byte) error {
	if !validFieldName.MatchString(table) {
		return fmt.Errorf("db: invalid table name %q", table)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.sql.Exec(fmt.Sprintf(`INSERT INTO %s (id, doc) VALUES (?, ?)`, sqlIdent(table)), id, string(docJSON))
	return err
}

func (s *sqliteStore) updateRow(table, id string, docJSON []byte) error {
	if !validFieldName.MatchString(table) {
		return fmt.Errorf("db: invalid table name %q", table)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.sql.Exec(fmt.Sprintf(`UPDATE %s SET doc = ? WHERE id = ?`, sqlIdent(table)), string(docJSON), id)
	return err
}

func (s *sqliteStore) deleteRow(table, id string) error {
	if !validFieldName.MatchString(table) {
		return fmt.Errorf("db: invalid table name %q", table)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.sql.Exec(fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, sqlIdent(table)), id)
	return err
}

func (s *sqliteStore) clearRows(table string) error {
	return s.clearTable(table)
}

type rawRow struct {
	ID  string
	Doc map[string]interface{}
}

func (s *sqliteStore) loadAll(table string) ([]rawRow, error) {
	if !validFieldName.MatchString(table) {
		return nil, fmt.Errorf("db: invalid table name %q", table)
	}
	s.mu.Lock()
	rows, err := s.sql.Query(fmt.Sprintf(`SELECT id, doc FROM %s ORDER BY rowid ASC`, sqlIdent(table)))
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []rawRow
	for rows.Next() {
		var id, docStr string
		if err := rows.Scan(&id, &docStr); err != nil {
			return nil, err
		}
		var doc map[string]interface{}
		if err := json.Unmarshal([]byte(docStr), &doc); err != nil {
			return nil, fmt.Errorf("db: corrupt row %s in table %s: %w", id, table, err)
		}
		out = append(out, rawRow{ID: id, Doc: doc})
	}
	return out, rows.Err()
}

func (s *sqliteStore) createIndex(table, indexName string, fields []string, unique bool) error {
	if !validFieldName.MatchString(table) {
		return fmt.Errorf("db: invalid table name %q", table)
	}
	if !validFieldName.MatchString(indexName) {
		return fmt.Errorf("db: invalid index name %q", indexName)
	}
	exprs := make([]string, len(fields))
	for i, f := range fields {
		if f == "_id" || f == "id" {
			exprs[i] = "id"
			continue
		}
		// Field names are validated against a strict identifier pattern
		// rather than merely quote-escaped, since they are interpolated
		// into a json_extract(...) path expression rather than bound as
		// a query parameter. This rules out any SQL syntax being smuggled
		// through a crafted field name.
		if !validFieldName.MatchString(f) {
			return fmt.Errorf("db: invalid field name %q", f)
		}
		exprs[i] = fmt.Sprintf("json_extract(doc, '$.%s')", f)
	}
	uniqueKw := ""
	if unique {
		uniqueKw = "UNIQUE "
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	stmt := fmt.Sprintf(`CREATE %sINDEX IF NOT EXISTS %s ON %s (%s)`,
		uniqueKw, sqlIdent("idx_"+table+"_"+indexName), sqlIdent(table), joinStrings(exprs, ", "))
	_, err := s.sql.Exec(stmt)
	return err
}

func (s *sqliteStore) nextSeq(name string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.sql.Begin()
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(`INSERT INTO __lunex_seqs__ (name, value) VALUES (?, 1)
		ON CONFLICT(name) DO UPDATE SET value = value + 1`, name)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	var val int64
	if err := tx.QueryRow(`SELECT value FROM __lunex_seqs__ WHERE name = ?`, name).Scan(&val); err != nil {
		tx.Rollback()
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return val, nil
}

func (s *sqliteStore) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sql.Close()
}

func joinStrings(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}
