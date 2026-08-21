package player

import (
	"errors"
	"net/http"
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestServerRecordsOnlyUnexpectedServeExit(t *testing.T) {
	t.Parallel()

	logs := testutil.NewRecordingLogger()
	server := &Server{log: logs}

	server.recordServeExit(http.ErrServerClosed)
	require.Empty(t, logs.Records(), "normal HTTP shutdown must remain silent")

	serveErr := errors.New("listener accept failed")
	server.recordServeExit(serveErr)
	records := logs.Records()
	require.Len(t, records, 1)
	require.Equal(t, "error", records[0].Level)
	require.Equal(t, "player server stopped unexpectedly", records[0].Message)
	require.Equal(t, "player.serve", records[0].Fields["operation"])
	require.ErrorIs(t, records[0].Fields["error"].(error), serveErr)
}
