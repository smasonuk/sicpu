package cpu

// ReplyFunc is a callback that a message device can use to send a response message.
type ReplyFunc func(target string, body []byte) error

// MessageDevice defines the interface for a device on the message bus.
type MessageDevice interface {
	HandleMessage(reply ReplyFunc, sender string, body []byte)
	Type() string
}

// StatefulMessageDevice is a MessageDevice that can have its state saved and restored during hibernation.
type StatefulMessageDevice interface {
	MessageDevice
	SaveState() []byte
	LoadState(data []byte) error
}

// MessageDeviceFactory is a function that creates a MessageDevice.
type MessageDeviceFactory func() MessageDevice

// msgDeviceRegistry is the global registry of message device factories.
var msgDeviceRegistry = make(map[string]MessageDeviceFactory)

// RegisterMessageDevice registers a factory for a given message device type name.
func RegisterMessageDevice(name string, factory MessageDeviceFactory) {
	msgDeviceRegistry[name] = factory
}
