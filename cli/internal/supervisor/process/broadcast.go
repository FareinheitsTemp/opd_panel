package process

import "sync"

// broadcast fans out log lines from one writer to N subscribers.
// Each subscriber gets its own buffered channel.
// Slow subscribers drop messages instead of blocking the writer.
type broadcast struct {
	mu     sync.Mutex
	subs   []chan string
	closed bool
}

func newBroadcast() *broadcast {
	return &broadcast{}
}

func (b *broadcast) subscribe() <-chan string {
	ch := make(chan string, 512)
	b.mu.Lock()
	if b.closed {
		close(ch)
	} else {
		b.subs = append(b.subs, ch)
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
