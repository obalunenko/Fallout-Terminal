package player

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/coder/websocket"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/live"
)

func TestServerConvergesFourThroughSevenClients(t *testing.T) {
	for clientTotal := 4; clientTotal <= 7; clientTotal++ {
		t.Run(fmt.Sprintf("%d_clients", clientTotal), func(t *testing.T) {
			service := live.New(&serverRandom{values: []int{0, 1, 2, 3}}, serverWords{})
			service.Set("terminal-1", "Overseer", serverTree(), 1, "WELCOME")
			counts := &countRecorder{}
			server := startTestServer(t, service, counts.Add, 16)

			clients := make([]*websocket.Conn, 0, clientTotal)
			patternID := ""
			for range clientTotal {
				connection := dialPlayer(t, server.Info().URL)
				clients = append(clients, connection)
				message := readMessage(t, connection)
				if message.Type != MessageTerminalLive || message.Nav == nil || message.Hack == nil || len(message.Hack.Patterns) == 0 || !reflect.DeepEqual(message.Nav.Path, []string{"root"}) {
					t.Fatalf("initial message = %#v", message)
				}
				if patternID == "" {
					patternID = message.Hack.Patterns[0].ID
				}
			}
			waitForCount(t, counts, clientTotal)

			writeJSON(t, clients[0], map[string]any{"type": MessageHackPattern, "patternId": patternID})
			patternState := readConvergedMessages(t, clients, MessageHackState)
			if !publicPatternUsed(patternState.Hack, patternID) {
				t.Fatalf("pattern state did not converge as used: %#v", patternState.Hack)
			}
			candidateID := firstCandidateID(patternState.Hack)
			if candidateID == "" {
				t.Fatal("pattern state contained no guessable candidate")
			}
			writeJSON(t, clients[1], map[string]any{"type": MessageHackGuess, "targetId": candidateID})
			readConvergedMessages(t, clients, MessageHackState)

			inDocs := false
			for actionIndex := 0; actionIndex < 23; actionIndex++ {
				request := map[string]any{"type": MessageNavAction, "action": "enter", "nodeId": "docs"}
				wantPath := []string{"root", "docs"}
				if inDocs {
					request = map[string]any{"type": MessageNavAction, "action": "back"}
					wantPath = []string{"root"}
				}
				writeJSON(t, clients[actionIndex%len(clients)], request)
				message := readConvergedMessages(t, clients, MessageNavState)
				if message.Nav == nil || !reflect.DeepEqual(message.Nav.Path, wantPath) {
					t.Fatalf("action %d navigation = %#v, want path %v", actionIndex+3, message, wantPath)
				}
				inDocs = !inDocs
			}
		})
	}
}

func TestReconnectReceivesCurrentStateAndClearRemovesIt(t *testing.T) {
	service := live.New(nil, nil)
	service.Set("terminal-1", "Overseer", serverTree(), 0, "WELCOME")
	server := startTestServer(t, service, nil, 8)

	first := dialPlayer(t, server.Info().URL)
	second := dialPlayer(t, server.Info().URL)
	readMessage(t, first)
	readMessage(t, second)
	writeJSON(t, first, map[string]any{"type": MessageNavAction, "action": "enter", "nodeId": "docs"})
	readMessage(t, first)
	readMessage(t, second)
	_ = second.Close(websocket.StatusNormalClosure, "reconnect")

	writeJSON(t, first, map[string]any{"type": MessageNavAction, "action": "entry", "nodeId": "report"})
	if message := readMessage(t, first); message.Nav == nil || message.Nav.Mode != "entry" {
		t.Fatalf("current client navigation = %#v", message)
	}

	reconnected := dialPlayer(t, server.Info().URL)
	defer reconnected.CloseNow()
	snapshot := readMessage(t, reconnected)
	if snapshot.Type != MessageTerminalLive || snapshot.Nav == nil || snapshot.Nav.Mode != "entry" || !reflect.DeepEqual(snapshot.Nav.Path, []string{"root", "docs"}) {
		t.Fatalf("reconnect snapshot = %#v", snapshot)
	}

	service.Clear()
	server.PublishClear()
	if message := readMessage(t, first); message.Type != MessageTerminalClear {
		t.Fatalf("first clear = %#v", message)
	}
	if message := readMessage(t, reconnected); message.Type != MessageTerminalClear {
		t.Fatalf("reconnected clear = %#v", message)
	}

	afterClear := dialPlayer(t, server.Info().URL)
	defer afterClear.CloseNow()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, _, err := afterClear.Read(ctx); err == nil {
		t.Fatal("connection after clear received a stale live snapshot")
	}
}

func TestSlowClientDoesNotBlockBroadcastOrFastClient(t *testing.T) {
	service := live.New(nil, nil)
	service.Set("terminal-1", "Overseer", serverTreeWithLargeBody(), 0, "WELCOME")
	server := startTestServer(t, service, nil, 8)

	slow := dialPlayer(t, server.Info().URL)
	defer slow.CloseNow()
	readMessage(t, slow)
	fast := dialPlayer(t, server.Info().URL)
	defer fast.CloseNow()
	readMessage(t, fast)

	fastMessages := make(chan serverMessage, 64)
	fastDone := make(chan struct{})
	go func() {
		defer close(fastDone)
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, payload, err := fast.Read(ctx)
			cancel()
			if err != nil {
				return
			}
			var message serverMessage
			if json.Unmarshal(payload, &message) == nil {
				fastMessages <- message
			}
		}
	}()

	for index := 0; index < 16; index++ {
		intro := fmt.Sprintf("update-%d", index)
		if _, ok := service.Update(serverTreeWithLargeBody(), &intro); !ok {
			t.Fatal("live update unexpectedly failed")
		}
		started := time.Now()
		server.PublishUpdate()
		// The race detector makes the deliberately large JSON fixture's local
		// marshal cost substantially slower on loaded CI hosts. A blocked socket
		// write would take the connection write timeout, so this ceiling still
		// proves broadcast is queue-bound without conflating instrumentation cost.
		if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
			t.Fatalf("broadcast %d waited %s for a slow client", index, elapsed)
		}
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case message := <-fastMessages:
			if message.Type == MessageTerminalUpdate {
				return
			}
		case <-deadline:
			t.Fatal("fast client did not receive an update while a slow client was isolated")
		}
	}
}

func TestGracefulShutdownClosesClientsAndListenerIdempotently(t *testing.T) {
	service := live.New(nil, nil)
	service.Set("terminal-1", "Overseer", serverTree(), 0, "WELCOME")
	server := startTestServer(t, service, nil, 4)
	info := server.Info()
	connection := dialPlayer(t, info.URL)
	readMessage(t, connection)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err := server.Stop(ctx); err != nil {
		t.Fatalf("second Stop() error = %v", err)
	}
	if _, _, err := connection.Read(ctx); err == nil {
		t.Fatal("client remained open after graceful shutdown")
	}
	if _, response, err := websocket.Dial(ctx, websocketURL(info.URL), nil); err == nil {
		t.Fatal("listener accepted a connection after shutdown")
	} else if response != nil {
		response.Body.Close()
	}
}

func TestServerRejectsPortAlreadyOwnedOnConfiguredIPv4Interface(t *testing.T) {
	holder, err := net.Listen("tcp4", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	port := holder.Addr().(*net.TCPAddr).Port
	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")},
	}
	server, err := NewServer(Config{
		Address: fmt.Sprintf("0.0.0.0:%d", port), Assets: fs.FS(assets), Live: live.New(nil, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.Start(context.Background()); err == nil {
		_ = server.Stop(context.Background())
		t.Fatal("Start() acquired an IPv6 wildcard while the configured IPv4 port was already owned")
	}
}

func TestPlayerPatternActionPublishesPublicCallback(t *testing.T) {
	service := live.New(&serverRandom{values: []int{0}}, serverWords{})
	service.Set("terminal-1", "Overseer", serverTree(), 1, "WELCOME")
	states := make(chan *domain.PublicHackState, 1)
	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")},
	}
	server, err := NewServer(Config{
		Address: "127.0.0.1:0", Assets: fs.FS(assets), Live: service,
		OnHackState: func(state *domain.PublicHackState) { states <- state },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Stop(ctx)
	})

	connection := dialPlayer(t, server.Info().URL)
	snapshot := readMessage(t, connection)
	if snapshot.Type != MessageTerminalLive || snapshot.Hack == nil || len(snapshot.Hack.Patterns) == 0 {
		t.Fatalf("initial message = %#v", snapshot)
	}
	patternID := snapshot.Hack.Patterns[0].ID
	writeJSON(t, connection, map[string]any{"type": MessageHackPattern, "patternId": patternID})
	if message := readMessage(t, connection); message.Type != MessageHackState {
		t.Fatalf("hack broadcast = %#v", message)
	}
	select {
	case state := <-states:
		if state == nil || state.Level != 1 || !publicPatternUsed(state, patternID) {
			t.Fatalf("hack callback = %#v", state)
		}
	case <-time.After(time.Second):
		t.Fatal("player hack action did not reach public callback")
	}
}

func TestRejectedPatternDoesNotBroadcastAndReconnectReceivesCurrentState(t *testing.T) {
	service := live.New(&serverRandom{values: []int{1, 0}}, serverWords{})
	service.Set("terminal-1", "Overseer", serverTree(), 1, "WELCOME")
	server := startTestServer(t, service, nil, 8)

	first := dialPlayer(t, server.Info().URL)
	defer first.CloseNow()
	second := dialPlayer(t, server.Info().URL)
	initialFirst := readMessage(t, first)
	readMessage(t, second)
	if initialFirst.Hack == nil || len(initialFirst.Hack.Patterns) == 0 {
		t.Fatalf("initial pattern state = %#v", initialFirst)
	}
	patternID := initialFirst.Hack.Patterns[0].ID
	writeJSON(t, first, map[string]any{"type": MessageHackPattern, "patternId": patternID})
	accepted := readConvergedMessages(t, []*websocket.Conn{first, second}, MessageHackState)
	if !publicPatternUsed(accepted.Hack, patternID) {
		t.Fatalf("accepted state = %#v", accepted.Hack)
	}

	_ = second.Close(websocket.StatusNormalClosure, "reconnect")
	reconnected := dialPlayer(t, server.Info().URL)
	defer reconnected.CloseNow()
	snapshot := readMessage(t, reconnected)
	if snapshot.Type != MessageTerminalLive || !reflect.DeepEqual(snapshot.Hack, accepted.Hack) {
		t.Fatalf("reconnect hack state = %#v, want %#v", snapshot.Hack, accepted.Hack)
	}

	if _, ok := service.ForceHackSuccess(); !ok {
		t.Fatal("ForceHackSuccess() rejected active puzzle")
	}
	server.PublishHack()
	readMessage(t, first)
	readMessage(t, reconnected)
	writeJSON(t, first, map[string]any{"type": MessageHackPattern, "patternId": patternID})
	assertNoPlayerMessage(t, first)
	assertNoPlayerMessage(t, reconnected)
}

func assertNoPlayerMessage(t *testing.T, connection *websocket.Conn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, _, err := connection.Read(ctx); err == nil {
		t.Fatal("unexpected player broadcast")
	}
}

func publicPatternUsed(state *domain.PublicHackState, patternID string) bool {
	if state == nil {
		return false
	}
	for _, pattern := range state.Patterns {
		if pattern.ID == patternID {
			return pattern.Used
		}
	}
	return false
}

type serverMessage struct {
	Type string                  `json:"type"`
	Nav  *domain.NavState        `json:"nav,omitempty"`
	Hack *domain.PublicHackState `json:"hack,omitempty"`
}

func readConvergedMessages(t *testing.T, clients []*websocket.Conn, wantType string) serverMessage {
	t.Helper()
	var canonical serverMessage
	for index, connection := range clients {
		message := readMessage(t, connection)
		if message.Type != wantType {
			t.Fatalf("client %d message type = %q, want %q", index, message.Type, wantType)
		}
		if index == 0 {
			canonical = message
			continue
		}
		if !reflect.DeepEqual(message, canonical) {
			t.Fatalf("client %d diverged: got %#v, canonical %#v", index, message, canonical)
		}
	}
	return canonical
}

func firstCandidateID(state *domain.PublicHackState) string {
	if state == nil {
		return ""
	}
	for _, column := range state.Columns {
		for _, word := range column.Words {
			return word.ID
		}
	}
	return ""
}

func startTestServer(t *testing.T, service *live.Service, onCount func(int), queueSize int) *Server {
	t.Helper()
	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")},
	}
	server, err := NewServer(Config{
		Address:       "127.0.0.1:0",
		Assets:        fs.FS(assets),
		Live:          service,
		QueueSize:     queueSize,
		OnClientCount: onCount,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Stop(ctx)
	})
	return server
}

func dialPlayer(t *testing.T, rawURL string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	origin, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	connection, response, err := websocket.Dial(ctx, websocketURL(rawURL), &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{origin.Scheme + "://" + origin.Host}},
	})
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		t.Fatal(err)
	}
	connection.SetReadLimit(8 << 20)
	t.Cleanup(func() { connection.CloseNow() })
	return connection
}

func readMessage(t *testing.T, connection *websocket.Conn) serverMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	messageType, payload, err := connection.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageText {
		t.Fatalf("message type = %v, want text", messageType)
	}
	var message serverMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		t.Fatalf("decode %s: %v", payload, err)
	}
	return message
}

func writeJSON(t *testing.T, connection *websocket.Conn, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := connection.Write(ctx, websocket.MessageText, payload); err != nil {
		t.Fatal(err)
	}
}

func websocketURL(rawURL string) string {
	return "ws" + rawURL[len("http"):]
}

type countRecorder struct {
	mu     sync.Mutex
	counts []int
}

func (recorder *countRecorder) Add(count int) {
	recorder.mu.Lock()
	recorder.counts = append(recorder.counts, count)
	recorder.mu.Unlock()
}

func (recorder *countRecorder) Last() int {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if len(recorder.counts) == 0 {
		return 0
	}
	return recorder.counts[len(recorder.counts)-1]
}

func waitForCount(t *testing.T, recorder *countRecorder, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if recorder.Last() == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("client count = %d, want %d", recorder.Last(), want)
}

func serverTree() domain.ContentNode {
	return domain.ContentNode{
		ID: "root", Type: domain.NodeFolder, Name: "ROOT",
		Children: []domain.ContentNode{{
			ID: "docs", Type: domain.NodeFolder, Name: "DOCS",
			Children: []domain.ContentNode{
				{ID: "report", Type: domain.NodeEntry, Name: "REPORT", Description: "Report"},
			},
		}},
	}
}

func serverTreeWithLargeBody() domain.ContentNode {
	tree := serverTree()
	tree.Children[0].Children = append(tree.Children[0].Children, domain.ContentNode{
		ID: "large", Type: domain.NodeEntry, Name: "LARGE", Description: string(make([]byte, 32*1024)),
	})
	return tree
}

type serverRandom struct {
	values []int
	index  int
}

func (random *serverRandom) Intn(limit int) int {
	if limit <= 0 || len(random.values) == 0 {
		return 0
	}
	value := random.values[random.index%len(random.values)]
	random.index++
	if value < 0 {
		value = -value
	}
	return value % limit
}

type serverWords struct{}

func (serverWords) PickWords(length, count int) []string {
	words := make([]string, count)
	for index := range words {
		word := fmt.Sprintf("%0*d", length, index)
		words[index] = word[len(word)-length:]
	}
	return words
}
