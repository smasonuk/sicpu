package cpu

import (
	"encoding/json"
	"testing"
)

type mockStatefulDevice struct {
	Counter int `json:"counter"`
}

func (m *mockStatefulDevice) HandleMessage(reply ReplyFunc, sender string, body []byte) {}
func (m *mockStatefulDevice) Type() string                                              { return "mockStatefulDevice" }
func (m *mockStatefulDevice) SaveState() []byte {
	b, _ := json.Marshal(m)
	return b
}
func (m *mockStatefulDevice) LoadState(data []byte) error {
	return json.Unmarshal(data, m)
}

func TestHibernateMessageDevices(t *testing.T) {
	RegisterMessageDevice("mockStatefulDevice", func() MessageDevice {
		return &mockStatefulDevice{}
	})

	c1 := NewCPU()
	dev1 := &mockStatefulDevice{Counter: 42}
	c1.MountMessageDevice("test@local", dev1)

	payload, err := c1.HibernateToBytes()
	if err != nil {
		t.Fatalf("HibernateToBytes failed: %v", err)
	}

	c2 := NewCPU()
	if err := c2.RestoreFromBytes(payload); err != nil {
		t.Fatalf("RestoreFromBytes failed: %v", err)
	}

	restoredDev, ok := c2.MessageDevices["test@local"]
	if !ok {
		t.Fatalf("expected message device 'test@local' to be restored")
	}

	mockDev, ok := restoredDev.(*mockStatefulDevice)
	if !ok {
		t.Fatalf("expected restored device to be of type *mockStatefulDevice")
	}

	if mockDev.Counter != 42 {
		t.Errorf("expected counter 42, got %d", mockDev.Counter)
	}
}
