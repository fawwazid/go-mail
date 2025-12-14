package mail

// Mailer defines an interface for sending emails.
// It is useful for dependency injection and testing.
type Mailer interface {
	Send(messages ...*Message) error
}

// MockClient is a mock implementation of Client for testing purposes.
type MockClient struct {
	SentMessages []*Message
	Err          error
}

// NewMockClient creates a new MockClient.
func NewMockClient() *MockClient {
	return &MockClient{
		SentMessages: make([]*Message, 0),
	}
}

// Send records the sent messages and returns any configured error.
func (m *MockClient) Send(messages ...*Message) error {
	if m.Err != nil {
		return m.Err
	}
	m.SentMessages = append(m.SentMessages, messages...)
	return nil
}
