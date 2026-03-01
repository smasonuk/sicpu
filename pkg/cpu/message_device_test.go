package cpu

import (
	"testing"
)

// DummyMessageDevice is a minimal implementation for testing the registry.
type DummyMessageDevice struct{}

func (d *DummyMessageDevice) HandleMessage(reply ReplyFunc, sender string, body []byte) {}
func (d *DummyMessageDevice) Type() string                                              { return "DummyMessageDevice" }

func TestRegisterMessageDevice(t *testing.T) {
	factory := func() MessageDevice {
		return &DummyMessageDevice{}
	}

	RegisterMessageDevice("DummyMessageDevice", factory)

	registeredFactory, ok := msgDeviceRegistry["DummyMessageDevice"]
	if !ok {
		t.Fatalf("expected 'DummyMessageDevice' to be registered")
	}

	dev := registeredFactory()
	if dev.Type() != "DummyMessageDevice" {
		t.Errorf("expected device type 'DummyMessageDevice', got '%s'", dev.Type())
	}
}
