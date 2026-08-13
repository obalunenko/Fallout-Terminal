package player

import (
	"context"
	"sync"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	playerv1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1"
	"google.golang.org/protobuf/proto"
)

const defaultSubscriptionQueueSize = 32

// Subscription owns the bounded outbound queue and cancellation lifecycle for
// one physical Connect server stream. Its complete snapshot is sent directly
// before Updates is drained, so queued values can never precede it.
type Subscription struct {
	id        domain.ConnectionID
	sessionID domain.LogicalSessionID
	snapshot  *playerv1.SubscriptionMessage
	updates   chan *playerv1.SubscriptionMessage
	done      chan struct{}
	cancel    context.CancelFunc

	mu           sync.Mutex
	lastRevision uint64
	closeOnce    sync.Once
}

// NewSubscription constructs one physical stream with an immutable first
// snapshot and a bounded queue for strictly newer compound updates.
func NewSubscription(parent context.Context, id domain.ConnectionID, sessionID domain.LogicalSessionID, snapshot *playerv1.SubscriptionMessage, queueSize int) *Subscription {
	if parent == nil {
		parent = context.Background()
	}
	if queueSize <= 0 {
		queueSize = defaultSubscriptionQueueSize
	}
	ctx, cancel := context.WithCancel(parent)
	stream := &Subscription{
		id: id, sessionID: sessionID, snapshot: cloneSubscriptionMessage(snapshot),
		updates: make(chan *playerv1.SubscriptionMessage, queueSize), done: make(chan struct{}), cancel: cancel,
	}
	if messageSnapshot := stream.snapshot.GetSnapshot(); messageSnapshot != nil {
		stream.lastRevision = messageSnapshot.GetRevision()
	}
	go func() {
		<-ctx.Done()
		stream.close()
	}()
	return stream
}

// ID returns the physical stream identity.
func (stream *Subscription) ID() domain.ConnectionID {
	if stream == nil {
		return ""
	}
	return stream.id
}

// SessionID returns the process-local logical owner.
func (stream *Subscription) SessionID() domain.LogicalSessionID {
	if stream == nil {
		return ""
	}
	return stream.sessionID
}

// Snapshot returns a detached copy of the mandatory first message.
func (stream *Subscription) Snapshot() *playerv1.SubscriptionMessage {
	if stream == nil {
		return nil
	}
	return cloneSubscriptionMessage(stream.snapshot)
}

// Updates returns the bounded post-snapshot delivery queue.
func (stream *Subscription) Updates() <-chan *playerv1.SubscriptionMessage {
	if stream == nil {
		closed := make(chan *playerv1.SubscriptionMessage)
		close(closed)
		return closed
	}
	return stream.updates
}

// Offer enqueues one strictly newer compound update without blocking. An
// invalid/same-revision value or full queue closes only this stream.
func (stream *Subscription) Offer(message *playerv1.SubscriptionMessage) bool {
	if stream == nil || message == nil || message.GetUpdate() == nil {
		return false
	}
	revision := message.GetUpdate().GetRevision()
	stream.mu.Lock()
	if revision == stream.lastRevision {
		stream.mu.Unlock()
		return true
	}
	if revision < stream.lastRevision {
		stream.mu.Unlock()
		stream.Close()
		return false
	}
	select {
	case <-stream.done:
		stream.mu.Unlock()
		return false
	default:
	}
	copy := cloneSubscriptionMessage(message)
	select {
	case stream.updates <- copy:
		stream.lastRevision = revision
		stream.mu.Unlock()
		return true
	default:
		stream.mu.Unlock()
		stream.Close()
		return false
	}
}

// Count reports active physical subscriptions.
func (hub *SubscriptionHub) Count() int {
	if hub == nil {
		return 0
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	return len(hub.byID)
}

// Close cancels and releases this physical stream idempotently.
func (stream *Subscription) Close() {
	if stream == nil {
		return
	}
	stream.cancel()
	stream.close()
}

func (stream *Subscription) close() {
	stream.closeOnce.Do(func() { close(stream.done) })
}

// Done closes when this physical stream terminates.
func (stream *Subscription) Done() <-chan struct{} {
	if stream == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return stream.done
}

// SubscriptionHub indexes physical streams by logical session and offers one
// detached personalized update to each currently responsive sibling.
type SubscriptionHub struct {
	mu        sync.Mutex
	byID      map[domain.ConnectionID]*Subscription
	bySession map[domain.LogicalSessionID]map[domain.ConnectionID]*Subscription
}

// NewSubscriptionHub returns an empty physical-stream registry.
func NewSubscriptionHub() *SubscriptionHub {
	return &SubscriptionHub{
		byID:      make(map[domain.ConnectionID]*Subscription),
		bySession: make(map[domain.LogicalSessionID]map[domain.ConnectionID]*Subscription),
	}
}

// Register adds a stream and removes any older stream with the same physical ID.
func (hub *SubscriptionHub) Register(stream *Subscription) {
	if hub == nil || stream == nil || stream.ID() == "" || stream.SessionID() == "" {
		return
	}
	hub.mu.Lock()
	if previous := hub.byID[stream.ID()]; previous != nil {
		hub.removeLocked(previous)
		previous.Close()
	}
	hub.byID[stream.ID()] = stream
	siblings := hub.bySession[stream.SessionID()]
	if siblings == nil {
		siblings = make(map[domain.ConnectionID]*Subscription)
		hub.bySession[stream.SessionID()] = siblings
	}
	siblings[stream.ID()] = stream
	hub.mu.Unlock()
}

// Unregister removes and closes one physical stream.
func (hub *SubscriptionHub) Unregister(id domain.ConnectionID) {
	if hub == nil || id == "" {
		return
	}
	hub.mu.Lock()
	stream := hub.byID[id]
	if stream != nil {
		hub.removeLocked(stream)
	}
	hub.mu.Unlock()
	if stream != nil {
		stream.Close()
	}
}

// Offer sends one logical update to every currently responsive physical stream.
func (hub *SubscriptionHub) Offer(sessionID domain.LogicalSessionID, message *playerv1.SubscriptionMessage) {
	if hub == nil || sessionID == "" || message == nil {
		return
	}
	hub.mu.Lock()
	streams := make([]*Subscription, 0, len(hub.bySession[sessionID]))
	for _, stream := range hub.bySession[sessionID] {
		streams = append(streams, stream)
	}
	hub.mu.Unlock()
	for _, stream := range streams {
		if !stream.Offer(message) {
			hub.Unregister(stream.ID())
		}
	}
}

// CloseAll detaches and cancels every physical stream without holding the hub
// lock across cancellation callbacks. It is safe to call repeatedly.
func (hub *SubscriptionHub) CloseAll() {
	if hub == nil {
		return
	}
	hub.mu.Lock()
	streams := make([]*Subscription, 0, len(hub.byID))
	for _, stream := range hub.byID {
		streams = append(streams, stream)
	}
	hub.byID = make(map[domain.ConnectionID]*Subscription)
	hub.bySession = make(map[domain.LogicalSessionID]map[domain.ConnectionID]*Subscription)
	hub.mu.Unlock()
	for _, stream := range streams {
		stream.Close()
	}
}

func (hub *SubscriptionHub) removeLocked(stream *Subscription) {
	delete(hub.byID, stream.ID())
	siblings := hub.bySession[stream.SessionID()]
	delete(siblings, stream.ID())
	if len(siblings) == 0 {
		delete(hub.bySession, stream.SessionID())
	}
}

func cloneSubscriptionMessage(message *playerv1.SubscriptionMessage) *playerv1.SubscriptionMessage {
	if message == nil {
		return nil
	}
	return proto.Clone(message).(*playerv1.SubscriptionMessage)
}
