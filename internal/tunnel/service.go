package tunnel

import (
	"context"
	"io"
	"io/fs"
	"net/url"
	"time"
)

// PlayerUpstreamAddress is the one existing player listener targeted by the
// embedded endpoint. Public access never starts a second server or proxy.
const PlayerUpstreamAddress = "127.0.0.1:3690"

// TunnelStartRequest is ephemeral provider input. AccountToken must be cleared
// immediately after the provider has constructed its owned agent.
type TunnelStartRequest struct {
	UpstreamURL    string
	ReservedDomain string
	AccountToken   []byte
	PlayerUsername []byte
	PlayerPassword []byte
	Timeout        time.Duration
}

func (request *TunnelStartRequest) Clear() {
	if request == nil {
		return
	}
	clear(request.AccountToken)
	clear(request.PlayerUsername)
	clear(request.PlayerPassword)
	request.AccountToken = nil
	request.PlayerUsername = nil
	request.PlayerPassword = nil
}

type TunnelService interface {
	Start(context.Context, TunnelStartRequest) (TunnelEndpoint, error)
}

type TunnelEndpoint interface {
	URL() *url.URL
	Done() <-chan struct{}
	Close(context.Context) error
}

type Clock interface {
	Now() time.Time
	After(time.Duration) <-chan time.Time
}

type SnapshotPublisher func(PublicAccessSnapshot)

type SyncWriteCloser interface {
	io.WriteCloser
	Sync() error
	Chmod(fs.FileMode) error
	Name() string
}

type SyncCloser interface {
	io.Closer
	Sync() error
}

type FileSystem interface {
	ReadFile(string) ([]byte, error)
	MkdirAll(string, fs.FileMode) error
	CreateTemp(string, string) (SyncWriteCloser, error)
	Rename(string, string) error
	Remove(string) error
	Chmod(string, fs.FileMode) error
	Open(string) (SyncCloser, error)
}
