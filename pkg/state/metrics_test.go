package state

import "testing"

func TestMetrics_IncreaseHoneyTrapped(t *testing.T) {
	type fields struct {
		honeyTrapped uint64
	}
	tests := []struct {
		name             string
		fields           fields
		wantHoneyTrapped uint64
	}{
		{
			name:             "should increase by one if initialized with 0",
			fields:           fields{honeyTrapped: 0},
			wantHoneyTrapped: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Metrics{
				honeyTrapped: tt.fields.honeyTrapped,
			}
			m.IncreaseHoneyTrapped()
			if m.HoneyTrapped() != tt.wantHoneyTrapped {
				t.Errorf("IncreaseHoneyTrapped() = %v, want %v", m.HoneyTrapped(), tt.wantHoneyTrapped)
			}
		})
	}
}
