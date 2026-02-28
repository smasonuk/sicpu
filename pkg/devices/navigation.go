package devices

import (
	"encoding/json"
	"fmt"
	"gocpu/pkg/cpu"
)

const NavigationDeviceType = "NavigationDevice"

// NavigationDevice is an example stateful message device.
type NavigationDevice struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// NewNavigationDevice creates a new NavigationDevice.
func NewNavigationDevice() *NavigationDevice {
	return &NavigationDevice{}
}

// Type returns the unique type identifier for hibernation.
func (n *NavigationDevice) Type() string {
	return NavigationDeviceType
}

// HandleMessage processes incoming messages.
func (n *NavigationDevice) HandleMessage(reply cpu.ReplyFunc, sender string, body []byte) {
	cmd := string(body)
	if cmd == "get_coords" {
		coords := fmt.Sprintf("%.2f,%.2f,%.2f", n.X, n.Y, n.Z)
		_ = reply(sender, []byte(coords))
	} else if cmd == "move_forward" {
		n.X += 10.0
		_ = reply(sender, []byte("ok"))
	}
}

// SaveState serializes the device state.
func (n *NavigationDevice) SaveState() []byte {
	b, _ := json.Marshal(n)
	return b
}

// LoadState deserializes the device state.
func (n *NavigationDevice) LoadState(data []byte) error {
	return json.Unmarshal(data, n)
}
