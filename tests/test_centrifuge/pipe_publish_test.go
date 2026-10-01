package test_centrifuge

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/centrifugal/gocent"

	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/utility"
)

func TestPipePublishChannels(t *testing.T) {
	logger := utility.NewLogger()

	t.Run("batches entries into a single HTTP call", func(t *testing.T) {
		var (
			mu               sync.Mutex
			receivedCommands []map[string]any
			requestCount     int32
		)

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)

			dec := json.NewDecoder(r.Body)
			var cmds []map[string]any
			for {
				var cmd map[string]any
				if err := dec.Decode(&cmd); err == io.EOF {
					break
				} else if err != nil {
					t.Errorf("failed to decode command: %v", err)
					break
				}
				cmds = append(cmds, cmd)
			}

			mu.Lock()
			receivedCommands = append(receivedCommands, cmds...)
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
			enc := json.NewEncoder(w)
			for range cmds {
				enc.Encode(map[string]any{"result": map[string]any{}})
			}
		}))
		defer mockServer.Close()

		originalClient := centrifuge.Client.C
		defer func() { centrifuge.Client.C = originalClient }()
		centrifuge.Client.C = gocent.New(gocent.Config{
			Addr: mockServer.URL,
			Key:  "test-key",
		})

		entries := []centrifuge.PublishEntry{
			{Channel: "org1/user1", Payload: map[string]string{"type": "unread"}},
			{Channel: "org1/user2", Payload: map[string]string{"type": "unread"}},
			{Channel: "org1/user3", Payload: map[string]string{"type": "mention"}},
		}

		centrifuge.PipePublishChannels(logger, entries)

		if got := atomic.LoadInt32(&requestCount); got != 1 {
			t.Errorf("expected 1 HTTP request, got %d", got)
		}

		mu.Lock()
		defer mu.Unlock()
		if len(receivedCommands) != 3 {
			t.Errorf("expected 3 commands in pipe, got %d", len(receivedCommands))
		}
	})

	t.Run("chunks large entry sets into multiple batches", func(t *testing.T) {
		var requestCount int32

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)

			dec := json.NewDecoder(r.Body)
			var count int
			for {
				var cmd map[string]any
				if err := dec.Decode(&cmd); err == io.EOF {
					break
				} else if err != nil {
					break
				}
				count++
			}

			w.WriteHeader(http.StatusOK)
			enc := json.NewEncoder(w)
			for i := 0; i < count; i++ {
				enc.Encode(map[string]any{"result": map[string]any{}})
			}
		}))
		defer mockServer.Close()

		originalClient := centrifuge.Client.C
		defer func() { centrifuge.Client.C = originalClient }()
		centrifuge.Client.C = gocent.New(gocent.Config{
			Addr: mockServer.URL,
			Key:  "test-key",
		})

		entryCount := 300
		entries := make([]centrifuge.PublishEntry, entryCount)
		for i := range entries {
			entries[i] = centrifuge.PublishEntry{
				Channel: "org1/user",
				Payload: map[string]int{"i": i},
			}
		}

		centrifuge.PipePublishChannels(logger, entries)

		if got := atomic.LoadInt32(&requestCount); got != 2 {
			t.Errorf("expected 2 HTTP requests (batches), got %d", got)
		}
	})

	t.Run("handles empty entries without making requests", func(t *testing.T) {
		var requestCount int32

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)
			w.WriteHeader(http.StatusOK)
		}))
		defer mockServer.Close()

		originalClient := centrifuge.Client.C
		defer func() { centrifuge.Client.C = originalClient }()
		centrifuge.Client.C = gocent.New(gocent.Config{
			Addr: mockServer.URL,
			Key:  "test-key",
		})

		centrifuge.PipePublishChannels(logger, []centrifuge.PublishEntry{})

		if got := atomic.LoadInt32(&requestCount); got != 0 {
			t.Errorf("expected 0 HTTP requests for empty entries, got %d", got)
		}
	})

	t.Run("logs errors on server failure without panicking", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer mockServer.Close()

		originalClient := centrifuge.Client.C
		defer func() { centrifuge.Client.C = originalClient }()
		centrifuge.Client.C = gocent.New(gocent.Config{
			Addr: mockServer.URL,
			Key:  "test-key",
		})

		entries := []centrifuge.PublishEntry{
			{Channel: "org1/user1", Payload: map[string]string{"type": "unread"}},
		}

		centrifuge.PipePublishChannels(logger, entries)
	})
}
