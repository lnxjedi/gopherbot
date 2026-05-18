package robot

import (
	"log"
	"sync"
)

type QueueMessage struct {
	ID         string
	Body       []byte
	Attributes map[string]string
}

type QueueDisposition int

const (
	QueueAck QueueDisposition = iota
	QueueRetry
)

type QueueProvider interface {
	Run(stop <-chan struct{})
}

type InitializedQueueProvider struct {
	Provider QueueProvider
}

type QueueProviderRegistration struct {
	Initialize func(QueueHandler, *log.Logger) (InitializedQueueProvider, error)
}

type QueueHandler interface {
	GetQueueConfig(interface{}) error
	HandleQueueMessage(QueueMessage) QueueDisposition
	ReadEncryptedFile(path string) ([]byte, error)
	Log(l LogLevel, m string, v ...interface{})
	GetInstallPath() string
	GetConfigPath() string
}

var queueProviderRegistry = struct {
	sync.RWMutex
	registrations map[string]QueueProviderRegistration
}{
	registrations: make(map[string]QueueProviderRegistration),
}

func RegisterQueueProvider(name string, initialize func(QueueHandler, *log.Logger) (InitializedQueueProvider, error)) {
	queueProviderRegistry.Lock()
	defer queueProviderRegistry.Unlock()

	validateNameOrFatal(name)

	if _, exists := queueProviderRegistry.registrations[name]; exists {
		log.Fatalf("Queue provider '%s' is already registered", name)
	}
	queueProviderRegistry.registrations[name] = QueueProviderRegistration{
		Initialize: initialize,
	}
}

func GetQueueProviderRegistration(name string) (QueueProviderRegistration, bool) {
	queueProviderRegistry.RLock()
	defer queueProviderRegistry.RUnlock()
	registration, ok := queueProviderRegistry.registrations[name]
	return registration, ok
}

func ListQueueProviderRegistrations() map[string]QueueProviderRegistration {
	queueProviderRegistry.RLock()
	defer queueProviderRegistry.RUnlock()
	out := make(map[string]QueueProviderRegistration, len(queueProviderRegistry.registrations))
	for name, registration := range queueProviderRegistry.registrations {
		out[name] = registration
	}
	return out
}
