package none

import (
	"github.com/librespeed/speedtest-go/database/schema"
)

type None struct{}

func Open(_ string) *None {
	return &None{}
}

func (n *None) Insert(_ *schema.TelemetryData) error {
	return nil
}

func (n *None) FetchByUUID(_ string) (*schema.TelemetryData, error) {
	return &schema.TelemetryData{}, nil
}

func (n *None) FetchLast100() ([]schema.TelemetryData, error) {
	return []schema.TelemetryData{}, nil
}

func (n *None) DeleteByUUIDs(_ []string) (int64, error) {
	return 0, nil
}

func (n *None) DeleteAll() (int64, error) {
	return 0, nil
}
