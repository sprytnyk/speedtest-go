package database

import (
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
				records, err := db.FetchLast100()
				if err != nil {
					t.Fatalf("FetchLast100: %s", err)
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
