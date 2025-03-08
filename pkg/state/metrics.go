package state

import "sync"

type Metrics struct {
	honeyTrapped uint64
	mutex        sync.Mutex
}

func (m *Metrics) HoneyTrapped() uint64 {
	return m.honeyTrapped
}

func (m *Metrics) IncreaseHoneyTrapped() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.honeyTrapped++
}
