package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Load reads migrations from dir.
// Supported files:
// - 0001_init.up.sql
// - 0001_init.down.sql
// - 0001_init.pg.up.sql (dialect-specific)
func Load(dir, dialect string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	byVersion := map[int]*Migration{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ver, name, dirn, ok := ParseMigrationFileName(entry.Name(), dialect)
		if !ok {
			continue
		}
		m := byVersion[ver]
		if m == nil {
			m = &Migration{Version: ver, Name: name}
			byVersion[ver] = m
		}
		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if dirn == "up" {
			m.UpSQL = string(content)
		} else {
			m.DownSQL = string(content)
		}
	}

	versions := make([]int, 0, len(byVersion))
	for v := range byVersion {
		versions = append(versions, v)
	}
	sort.Ints(versions)
	out := make([]Migration, 0, len(versions))
	for _, v := range versions {
		m := byVersion[v]
		if m.UpSQL == "" {
			return nil, fmt.Errorf("migrate: missing up migration for version %d", v)
		}
		out = append(out, *m)
	}
	return out, nil
}
