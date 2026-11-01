package mail

import (
	"errors"
	"sync"
)

// Mailer defines an interface for sending emails.
// It is useful for dependency injection and testing.
type Mailer interface {
	Send(messages ...*Message) error
}

// MockClient is a mock implementation of Client for testing. Send, SetError,
// and Messages may be called concurrently. Direct access to the deprecated
// compatibility fields is not concurrency-safe.
type MockClient struct {
	mu sync.RWMutex

	// SentMessages contains messages recorded by Send. Deprecated: use Messages
	// to retrieve a snapshot. Direct reads and writes are not concurrency-safe.
	SentMessages []*Message
	// Err is returned by Send when non-nil. Deprecated: use SetError instead.
	// Direct reads and writes are not concurrency-safe.
	Err error
}

// NewMockClient creates a new MockClient.
func NewMockClient() *MockClient {
	return &MockClient{
		SentMessages: make([]*Message, 0),
	}
}

// SetError configures the error returned by Send.
func (m *MockClient) SetError(err error) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Err = err
}

// Messages returns a snapshot of messages recorded by Send.
func (m *MockClient) Messages() []*Message {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]*Message(nil), m.SentMessages...)
}

// Send records the sent messages and returns any configured error.
func (m *MockClient) Send(messages ...*Message) error {
	if m == nil {
		return errors.New("mock mail client must not be nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	m.SentMessages = append(m.SentMessages, messages...)
	return nil
}
