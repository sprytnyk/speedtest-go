package memory

import (
	"errors"
	"sync"
	"time"

	"github.com/librespeed/speedtest-go/database/schema"
)

const (
	// just enough records to return for FetchLast100
	maxRecords = 100
)

type Memory struct {
	lock    sync.RWMutex
	records []schema.TelemetryData
}

func Open(_ string) *Memory {
	return &Memory{}
}

func (mem *Memory) Insert(data *schema.TelemetryData) error {
	mem.lock.Lock()
	defer mem.lock.Unlock()
	data.Timestamp = time.Now()
	mem.records = append(mem.records, *data)
	if len(mem.records) > maxRecords {
		mem.records = mem.records[len(mem.records)-maxRecords:]
	}
	return nil
}

func (mem *Memory) FetchByUUID(uuid string) (*schema.TelemetryData, error) {
	mem.lock.RLock()
	defer mem.lock.RUnlock()
	for _, record := range mem.records {
		if record.UUID == uuid {
			return &record, nil
		}
	}
	return nil, errors.New("record not found")
}

func (mem *Memory) FetchLast100() ([]schema.TelemetryData, error) {
	mem.lock.RLock()
	defer mem.lock.RUnlock()
	// newest first, like the other backends
	records := make([]schema.TelemetryData, len(mem.records))
	for i, record := range mem.records {
		records[len(records)-1-i] = record
	}
	return records, nil
}

func (mem *Memory) DeleteByUUIDs(uuids []string) (int64, error) {
	mem.lock.Lock()
	defer mem.lock.Unlock()
	remove := make(map[string]bool, len(uuids))
	for _, uuid := range uuids {
		remove[uuid] = true
	}
	var deleted int64
	kept := mem.records[:0]
	for _, record := range mem.records {
		if remove[record.UUID] {
			deleted++
			continue
		}
		kept = append(kept, record)
	}
	mem.records = kept
	return deleted, nil
}

func (mem *Memory) DeleteAll() (int64, error) {
	mem.lock.Lock()
	defer mem.lock.Unlock()
	deleted := int64(len(mem.records))
	mem.records = nil
	return deleted, nil
}
