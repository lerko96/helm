package broker

import (
	"sync"
)

// Broker fan-outs SSE messages to all connected clients.
type Broker struct {
	mu      sync.Mutex
	clients map[string]chan string
}

func New() *Broker {
	return &Broker{clients: make(map[string]chan string)}
}

func (b *Broker) Subscribe(id string) chan string {
	ch := make(chan string, 8)
	b.mu.Lock()
	b.clients[id] = ch
	b.mu.Unlock()
	return ch
}

func (b *Broker) Unsubscribe(id string) {
	b.mu.Lock()
	if ch, ok := b.clients[id]; ok {
		close(ch)
		delete(b.clients, id)
	}
	b.mu.Unlock()
}

func (b *Broker) Publish(msg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}
