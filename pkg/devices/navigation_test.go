package devices

import (
	"testing"
)

func TestNavigationDevice_HandleMessage_MoveForward(t *testing.T) {
	dev := NewNavigationDevice()

	var replied bool
	reply := func(target string, body []byte) error {
		replied = true
		if target != "sender" {
			t.Errorf("expected target 'sender', got '%s'", target)
		}
		if string(body) != "ok" {
			t.Errorf("expected reply 'ok', got '%s'", string(body))
		}
		return nil
	}

	dev.HandleMessage(reply, "sender", []byte("move_forward"))

	if !replied {
		t.Fatalf("expected reply callback to be called")
	}

	if dev.X != 10.0 {
		t.Errorf("expected X to be 10.0, got %f", dev.X)
	}
}

func TestNavigationDevice_HandleMessage_GetCoords(t *testing.T) {
	dev := NewNavigationDevice()
	dev.X = 1.0
	dev.Y = 2.0
	dev.Z = 3.0

	var replied bool
	reply := func(target string, body []byte) error {
		replied = true
		expected := "1.00,2.00,3.00"
		if string(body) != expected {
			t.Errorf("expected '%s', got '%s'", expected, string(body))
		}
		return nil
	}

	dev.HandleMessage(reply, "sender", []byte("get_coords"))

	if !replied {
		t.Fatalf("expected reply callback to be called")
	}
}

func TestNavigationDevice_State(t *testing.T) {
	dev1 := NewNavigationDevice()
	dev1.X = 42.5
	dev1.Y = -10.0
	dev1.Z = 0.0

	data := dev1.SaveState()

	dev2 := NewNavigationDevice()
	err := dev2.LoadState(data)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if dev2.X != 42.5 || dev2.Y != -10.0 || dev2.Z != 0.0 {
		t.Errorf("state mismatch after restore: %+v", dev2)
	}
}
