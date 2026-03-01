1. **Define Message Device Interfaces**
   - Create `pkg/cpu/message_device.go`
   - Define `ReplyFunc` type: `type ReplyFunc func(target string, body []byte) error`
   - Define `MessageDevice` interface with methods `HandleMessage(reply ReplyFunc, sender string, body []byte)` and `Type() string`
   - Define `StatefulMessageDevice` interface embedding `MessageDevice` and adding `SaveState() []byte` and `LoadState(data []byte) error`
   - Create registry `type MessageDeviceFactory func() MessageDevice`, `var msgDeviceRegistry = make(map[string]MessageDeviceFactory)`, `func RegisterMessageDevice(name string, factory MessageDeviceFactory)`
   - Create `pkg/cpu/message_device_test.go` with registry tests.

2. **Integrate Message Bus into the CPU**
   - Update `pkg/cpu/cpu.go`: Add `MessageDevices map[string]MessageDevice` and `MessagePusher func(target string, body []byte) error` to `CPU` struct.
   - Initialize `MessageDevices` in `NewCPU()`.
   - Add `MountMessageDevice(address string, dev MessageDevice)` method to `CPU`.
   - Add `DispatchMessage(target string, body []byte)` method to `CPU`.
   - Create `pkg/cpu/cpu_message_test.go` and add the specified unit tests.

3. **Add Hibernation Support for Message Devices**
   - Update `pkg/cpu/hibernate.go`:
     - Add `MessageDevices map[string]string \`json:"message_devices"\`` to `humanReadableState`.
     - In `HibernateToBytes()`, initialize and populate `state.MessageDevices`, saving state for `StatefulMessageDevice` instances to hex-encoded bin files.
     - In `RestoreFromBytes()`, initialize `c.MessageDevices` and restore devices from `msgDeviceRegistry`, loading their state.
   - Create `pkg/cpu/hibernate_message_test.go` and add the specified unit tests.

4. **Create the Example Navigation Device**
   - Create directory `pkg/devices`.
   - Create `pkg/devices/navigation.go`.
   - Implement `NavigationDevice` with fields `X, Y, Z float64`.
   - Implement `Type()`, `HandleMessage()`, `SaveState()`, and `LoadState()`.
   - Define `NavigationDeviceType = "NavigationDevice"` and `NewNavigationDevice()`.
   - Create `pkg/devices/navigation_test.go` and add unit tests.

5. **Wire the Bus in Main Apps**
   - Update `cmd/desktop/main.go` and `cmd/console/main.go` (if it exists) to integrate the new system.
   - Ensure imports `gocpu/pkg/devices`.
   - Register `NavigationDevice` factory.
   - Mount `MessageReceiver` to slot 1, set `vm.MessagePusher`.
   - Mount `MessageSender` to slot 0, pass `vm.DispatchMessage`.
   - Mount `NavigationDevice` to "navigation@local".
   - Make sure peripherals are registered with `cpu.RegisterPeripheral`.
   - Provide integration test in `cmd/desktop/main_test.go`.

6. **Pre-commit Steps**
   - Ensure proper testing, verification, review, and reflection are done.
