package player

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestSameHostOrigin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		origin string
		host   string
		want   bool
	}{
		{name: "non-browser client", host: "127.0.0.1:3690", want: true},
		{name: "local same host", origin: "http://127.0.0.1:3690", host: "127.0.0.1:3690", want: true},
		{name: "public same host", origin: "https://players.example.test", host: "players.example.test", want: true},
		{name: "different host", origin: "https://attacker.example", host: "players.example.test", want: false},
		{name: "non HTTP scheme", origin: "file://players.example.test", host: "players.example.test", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequest("GET", "http://"+test.host+"/", nil)
			request.Host = test.host
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			if got := sameHostOrigin(request); got != test.want {
				t.Fatalf("sameHostOrigin() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestPlayerConnectionQueueIsBoundedAndNilSocketStops(t *testing.T) {
	t.Parallel()
	connection := NewPlayerConnection("slow", nil, 1)
	if !connection.Send([]byte("first")) {
		t.Fatal("first payload was not queued")
	}
	if connection.Send([]byte("overflow")) {
		t.Fatal("overflow payload was accepted")
	}
	connection.Start(context.Background(), nil)
	<-connection.Done()
	connection.Close()
}
