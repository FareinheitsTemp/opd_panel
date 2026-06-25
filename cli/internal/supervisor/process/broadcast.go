package process

import "sync"

// broadcast is a fan-out channel: one sender, multiple receivers.
// Each subscriber gets its own buffered channel.
type broadcast struct {
	mu   sync.Mutex
	subs []chan string
	closed bool
}

func newBroadcast(buf int) *broadcast {
	_ = buf
	return &broadcast{}
}

func (b *broadcast) subscribe() <-chan string {
	ch := make(chan string, 256)
	b.mu.Lock()
	if !b.closed {
		b.subs = append(b.subs, ch)
	} else {
		close(ch)
	}
	b.mu.Unlock()
	return ch
}

func (b *broadcast) send(msg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subs {
		select {
		case ch <- msg:
		default:
			// Drop if subscriber is slow
		}
	}
}

func (b *broadcast) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for _, ch := range b.subs {
		close(ch)
	}
	b.subs = nil
}
