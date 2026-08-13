package player

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"testing"
	"testing/fstest"
	"time"

	"connectrpc.com/connect"
	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	playerv1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1"
	"github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1/playerv1connect"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionQueueIsBoundedNonblockingAndRecoversOnlyFromANewSnapshot(t *testing.T) {
	snapshot := subscriptionSnapshot(1)
	stream := NewSubscription(t.Context(), "physical-1", "logical-1", snapshot, 0)
	require.Equal(t, defaultSubscriptionQueueSize, cap(stream.updates))
	for revision := uint64(2); revision <= defaultSubscriptionQueueSize+1; revision++ {
		require.True(t, stream.Offer(subscriptionUpdate(revision)), "revision %d", revision)
	}

	started := time.Now()
	require.False(t, stream.Offer(subscriptionUpdate(defaultSubscriptionQueueSize+2)))
	require.Less(t, time.Since(started), 50*time.Millisecond)
	require.Eventually(t, func() bool {
		select {
		case <-stream.Done():
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	require.False(t, stream.Offer(subscriptionUpdate(defaultSubscriptionQueueSize+3)))

	recovered := NewSubscription(t.Context(), "physical-2", "logical-1", subscriptionSnapshot(50), 2)
	require.False(t, recovered.Offer(subscriptionUpdate(49)))
	select {
	case <-recovered.Done():
	case <-time.After(time.Second):
		t.Fatal("stale increment did not terminate the physical recovery stream")
	}
	fresh := NewSubscription(t.Context(), "physical-3", "logical-1", subscriptionSnapshot(50), 2)
	require.True(t, fresh.Offer(subscriptionUpdate(51)))
	require.Equal(t, uint64(51), (<-fresh.Updates()).GetUpdate().GetRevision())
}

func TestSubscriptionHubIsolatesOverflowingAndCanceledPhysicalSiblings(t *testing.T) {
	hub := NewSubscriptionHub()
	blocked := NewSubscription(t.Context(), "blocked", "logical-1", subscriptionSnapshot(1), 1)
	healthy := NewSubscription(t.Context(), "healthy", "logical-1", subscriptionSnapshot(1), 2)
	canceledContext, cancel := context.WithCancel(t.Context())
	canceled := NewSubscription(canceledContext, "canceled", "logical-1", subscriptionSnapshot(1), 1)
	hub.Register(blocked)
	hub.Register(healthy)
	hub.Register(canceled)
	cancel()
	select {
	case <-canceled.Done():
	case <-time.After(time.Second):
		t.Fatal("canceled physical stream remained active")
	}

	hub.Offer("logical-1", subscriptionUpdate(2))
	require.Equal(t, uint64(2), (<-healthy.Updates()).GetUpdate().GetRevision())
	hub.Offer("logical-1", subscriptionUpdate(3))
	require.Equal(t, uint64(3), (<-healthy.Updates()).GetUpdate().GetRevision())
	select {
	case <-blocked.Done():
	case <-time.After(time.Second):
		t.Fatal("overflowing sibling was not isolated")
	}

	hub.mu.Lock()
	_, blockedRegistered := hub.byID["blocked"]
	_, canceledRegistered := hub.byID["canceled"]
	_, healthyRegistered := hub.byID["healthy"]
	hub.mu.Unlock()
	require.False(t, blockedRegistered)
	require.False(t, canceledRegistered)
	require.True(t, healthyRegistered)

	hub.Offer("logical-1", subscriptionUpdate(4))
	require.Equal(t, uint64(4), (<-healthy.Updates()).GetUpdate().GetRevision())
	hub.Unregister("healthy")
	hub.Unregister("healthy")
}

func TestRepresentativeThreeHourStreamReconnectSoak(t *testing.T) {
	t.Parallel()

	const simulatedSeconds = 3 * 60 * 60
	hub := NewSubscriptionHub()
	stream := NewSubscription(t.Context(), "physical-0", "logical-1", subscriptionSnapshot(1), 2)
	hub.Register(stream)
	currentRevision := uint64(1)
	reconnects := 0
	for second := 1; second <= simulatedSeconds; second++ {
		// One authoritative heartbeat-equivalent projection per simulated second
		// exercises long-run ordering without making the suite wait three hours.
		currentRevision++
		hub.Offer("logical-1", subscriptionUpdate(currentRevision))
		select {
		case update := <-stream.Updates():
			require.Equal(t, currentRevision, update.GetUpdate().GetRevision())
		case <-time.After(time.Second):
			t.Fatalf("simulated second %d did not deliver revision %d", second, currentRevision)
		}

		// Interrupt and recover every five simulated minutes. Recovery begins
		// from one complete current snapshot and never retries old increments.
		if second%(5*60) == 0 {
			hub.Unregister(stream.ID())
			reconnects++
			stream = NewSubscription(t.Context(), domain.ConnectionID(fmt.Sprintf("physical-%d", reconnects)), "logical-1", subscriptionSnapshot(currentRevision), 2)
			hub.Register(stream)
			require.Equal(t, currentRevision, stream.Snapshot().GetSnapshot().GetRevision())
		}
	}
	require.Equal(t, 36, reconnects)
	require.Equal(t, 1, hub.Count())
	hub.CloseAll()
	require.Equal(t, 0, hub.Count())
}

func TestConnectServerShutdownIsBoundedWithBlockedAndCanceledPhysicalStreams(t *testing.T) {
	coordinator := newConnectTestCoordinator(t)
	hub := NewSubscriptionHub()
	connectPlayer, err := NewConnectService(ConnectServiceConfig{Coordinator: coordinator, QueueSize: 1, Hub: hub})
	require.NoError(t, err)
	assets := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")}}
	server, err := NewServer(Config{
		Address: "127.0.0.1:0", Assets: fs.FS(assets), Connect: connectPlayer,
	})
	require.NoError(t, err)
	_, err = server.Start(t.Context())
	require.NoError(t, err)

	client := playerv1connect.NewPlayerServiceClient(http.DefaultClient, server.Info().LocalURL)
	blockedContext, cancelBlocked := context.WithCancel(t.Context())
	defer cancelBlocked()
	blocked, err := client.Subscribe(blockedContext, connect.NewRequest(&playerv1.SubscribeRequest{}))
	require.NoError(t, err)
	require.True(t, blocked.Receive(), "stream error: %v", blocked.Err())
	snapshot := blocked.Msg().GetSnapshot()
	require.NotNil(t, snapshot)

	healthyContext, cancelHealthy := context.WithCancel(t.Context())
	healthy, err := client.Subscribe(healthyContext, connect.NewRequest(&playerv1.SubscribeRequest{RecognitionHandle: &snapshot.RecognitionHandle}))
	require.NoError(t, err)
	require.True(t, healthy.Receive())
	healthySnapshot := healthy.Msg().GetSnapshot()
	sessionID := healthySnapshot.GetPlayerState().GetLogicalSessionId()
	require.NotEmpty(t, sessionID)

	hub.Offer(domain.LogicalSessionID(sessionID), subscriptionUpdate(healthySnapshot.GetRevision()+1))
	require.True(t, healthy.Receive())

	cancelHealthy()
	require.Eventually(t, func() bool {
		hub.mu.Lock()
		defer hub.mu.Unlock()
		return len(hub.byID) == 1
	}, time.Second, time.Millisecond)

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	started := time.Now()
	require.NoError(t, server.Stop(shutdownContext))
	require.Less(t, time.Since(started), 5*time.Second)
	require.NoError(t, server.Stop(shutdownContext))
	require.Eventually(t, func() bool {
		hub.mu.Lock()
		defer hub.mu.Unlock()
		return len(hub.byID) == 0
	}, time.Second, time.Millisecond)
}

func subscriptionSnapshot(revision uint64) *playerv1.SubscriptionMessage {
	return &playerv1.SubscriptionMessage{Payload: &playerv1.SubscriptionMessage_Snapshot{Snapshot: &playerv1.PersonalizedSnapshot{Revision: revision}}}
}

func subscriptionUpdate(revision uint64) *playerv1.SubscriptionMessage {
	return &playerv1.SubscriptionMessage{Payload: &playerv1.SubscriptionMessage_Update{Update: &playerv1.CompoundUpdate{Revision: revision}}}
}
