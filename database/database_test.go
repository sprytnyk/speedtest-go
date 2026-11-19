package database

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/librespeed/speedtest-go/database/bolt"
	"github.com/librespeed/speedtest-go/database/memory"
	"github.com/librespeed/speedtest-go/database/schema"
	"github.com/librespeed/speedtest-go/database/sqlite"
)

// TestDelete runs against the backends that need no database server.
func TestDelete(t *testing.T) {
	for name, open := range map[string]func(dir string) DataAccess{
		"memory": func(string) DataAccess { return memory.Open("") },
		"bolt":   func(dir string) DataAccess { return bolt.Open(filepath.Join(dir, "speedtest.db")) },
		"sqlite": func(dir string) DataAccess { return sqlite.Open(filepath.Join(dir, "speedtest.sqlite")) },
	} {
		t.Run(name, func(t *testing.T) {
			db := open(t.TempDir())
			uuids := func() []string {
				t.Helper()
				records, err := db.FetchPage(0, 100)
				if err != nil {
					t.Fatalf("FetchPage: %s", err)
				}
				var got []string
				for _, r := range records {
					got = append(got, r.UUID)
				}
				return got
			}

			if n, err := db.DeleteAll(); err != nil || n != 0 {
				t.Fatalf("DeleteAll on an empty database = %d, %v", n, err)
			}
			for _, id := range []string{"01A", "01B", "01C"} {
				if err := db.Insert(&schema.TelemetryData{UUID: id}); err != nil {
					t.Fatal(err)
				}
			}

			if n, err := db.DeleteByUUIDs([]string{"01B", "unknown"}); err != nil || n != 1 {
				t.Fatalf("DeleteByUUIDs = %d, %v; want 1", n, err)
			}
			// SQLite times tests to the second, so these may come in any order
			got := uuids()
			sort.Strings(got)
			if !reflect.DeepEqual(got, []string{"01A", "01C"}) {
				t.Fatalf("after DeleteByUUIDs: %v", got)
			}
			if _, err := db.FetchByUUID("01B"); err == nil {
				t.Error("a deleted test is still found")
			}

			if n, err := db.DeleteAll(); err != nil || n != 2 {
				t.Fatalf("DeleteAll = %d, %v; want 2", n, err)
			}
			if got := uuids(); len(got) != 0 {
				t.Fatalf("after DeleteAll: %v", got)
			}
			// the database still takes new tests
			if err := db.Insert(&schema.TelemetryData{UUID: "01D"}); err != nil {
				t.Fatal(err)
			}
			if got := uuids(); !reflect.DeepEqual(got, []string{"01D"}) {
				t.Fatalf("after a new insert: %v", got)
			}
		})
	}
}

func TestFetchPage(t *testing.T) {
	for _, tc := range []struct {
		name     string
		open     func(dir string) DataAccess
		retained int
	}{
		{"memory", func(string) DataAccess { return memory.Open("") }, 100},
		{"bolt", func(dir string) DataAccess { return bolt.Open(filepath.Join(dir, "speedtest.db")) }, 103},
		{"sqlite", func(dir string) DataAccess { return sqlite.Open(filepath.Join(dir, "speedtest.sqlite")) }, 103},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := tc.open(t.TempDir())
			testID := func(i int) string { return fmt.Sprintf("%026d", i) }
			for i := 1; i <= 103; i++ {
				if err := db.Insert(&schema.TelemetryData{UUID: testID(i)}); err != nil {
					t.Fatal(err)
				}
			}
			for _, page := range []struct{ offset, limit int }{{0, 100}, {100, 100}, {0, 3}, {2, 4}, {103, 100}} {
				rows, err := db.FetchPage(page.offset, page.limit)
				wantLen := min(page.limit, max(0, tc.retained-page.offset))
				if err != nil || len(rows) != wantLen {
					t.Fatalf("FetchPage(%d, %d) returned %d rows, want %d: %v", page.offset, page.limit, len(rows), wantLen, err)
				}
				for i, row := range rows {
					if want := testID(103 - page.offset - i); row.UUID != want {
						t.Fatalf("offset %d, row %d: ID %s, want %s", page.offset, i, row.UUID, want)
					}
				}
			}
		})
	}
}
