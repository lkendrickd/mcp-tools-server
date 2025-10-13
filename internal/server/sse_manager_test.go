package server

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func setupSSEManager() *SSEManager {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	return NewSSEManager(logger)
}

func TestSSEManager_AddAndRemoveClient(t *testing.T) {
	m := setupSSEManager()

	if len(m.clients) != 0 {
		t.Fatalf("Expected 0 clients, got %d", len(m.clients))
	}

	client := m.AddClient()
	if len(m.clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(m.clients))
	}
	if _, ok := m.clients[client.id]; !ok {
		t.Error("Client not found in map after adding")
	}

	m.RemoveClient(client.id)
	if len(m.clients) != 0 {
		t.Errorf("Expected 0 clients after removal, got %d", len(m.clients))
	}

	// Test that the channel is closed
	select {
	case _, ok := <-client.send:
		if ok {
			t.Error("Expected client channel to be closed")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for channel close check")
	}
}

func TestSSEManager_Send(t *testing.T) {
	m := setupSSEManager()
	client := m.AddClient()
	msg := []byte("hello")

	t.Run("send to valid client", func(t *testing.T) {
		err := m.Send(client.id, msg)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		select {
		case received := <-client.send:
			if string(received) != string(msg) {
				t.Errorf("Expected '%s', got '%s'", msg, received)
			}
		case <-time.After(1 * time.Second):
			t.Error("Timed out waiting for message")
		}
	})

	t.Run("send to non-existent client", func(t *testing.T) {
		err := m.Send("non-existent-id", msg)
		if err == nil {
			t.Fatal("Expected error for non-existent client, got nil")
		}
	})

	t.Run("send to removed client", func(t *testing.T) {
		m.RemoveClient(client.id)
		err := m.Send(client.id, msg)
		if err == nil {
			t.Fatal("Expected error for removed client, got nil")
		}
	})
}

func TestSSEManager_Broadcast(t *testing.T) {
	m := setupSSEManager()
	client1 := m.AddClient()
	client2 := m.AddClient()
	msg := []byte("broadcast")

	m.Broadcast(msg)

	// Check client 1
	select {
	case received := <-client1.send:
		if string(received) != string(msg) {
			t.Errorf("Client 1 expected '%s', got '%s'", msg, received)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for message on client 1")
	}

	// Check client 2
	select {
	case received := <-client2.send:
		if string(received) != string(msg) {
			t.Errorf("Client 2 expected '%s', got '%s'", msg, received)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for message on client 2")
	}

	// Test with one client removed
	m.RemoveClient(client1.id)
	m.Broadcast([]byte("after removal"))

	// A receive on a closed channel returns immediately with a zero value and ok=false.
	// We check to make sure nothing was sent *before* the channel was closed.
	if msg, ok := <-client1.send; ok {
		t.Errorf("Removed client should not have received a message, but got: %s", msg)
	}
}
