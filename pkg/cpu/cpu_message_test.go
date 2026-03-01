package cpu

import (
	"bytes"
	"testing"
)

type mockMessageDevice struct {
	called       bool
	lastSender   string
	lastBody     []byte
	replyPayload []byte
}

func (m *mockMessageDevice) HandleMessage(reply ReplyFunc, sender string, body []byte) {
	m.called = true
	m.lastSender = sender
	m.lastBody = body
	if m.replyPayload != nil {
		_ = reply(sender, m.replyPayload)
	}
}

func (m *mockMessageDevice) Type() string { return "mockMessageDevice" }

func TestCPUDispatchMessage(t *testing.T) {
	c := NewCPU()

	mockDev := &mockMessageDevice{
		replyPayload: []byte("pong"),
	}

	c.MountMessageDevice("test@local", mockDev)

	var pusherCalled bool
	var pusherTarget string
	var pusherBody []byte

	c.MessagePusher = func(target string, body []byte) error {
		pusherCalled = true
		pusherTarget = target
		pusherBody = body
		return nil
	}

	c.DispatchMessage("test@local", []byte("ping"))

	if !mockDev.called {
		t.Fatalf("expected mock device to have been called")
	}

	if string(mockDev.lastBody) != "ping" {
		t.Errorf("expected body 'ping', got '%s'", mockDev.lastBody)
	}

	if !pusherCalled {
		t.Fatalf("expected MessagePusher to be called")
	}

	if pusherTarget != "system@local" {
		t.Errorf("expected pusher target 'system@local', got '%s'", pusherTarget)
	}

	if !bytes.Equal(pusherBody, []byte("pong")) {
		t.Errorf("expected pusher body 'pong', got '%s'", string(pusherBody))
	}
}

func TestCPUDispatchMessageUnroutable(t *testing.T) {
	c := NewCPU()

	// Should not panic, but print to stdout
	c.DispatchMessage("unknown@local", []byte("ping"))
}
