package platform

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingDialogs struct {
	openOptions OpenFileOptions
	saveOptions SaveFileOptions
	openResult  string
	openErr     error
	saveResult  string
	saveErr     error
}

func (manager *recordingDialogs) OpenFile(_ context.Context, options OpenFileOptions) (string, error) {
	manager.openOptions = options
	return manager.openResult, manager.openErr
}

func (manager *recordingDialogs) SaveFile(_ context.Context, options SaveFileOptions) (string, error) {
	manager.saveOptions = options
	return manager.saveResult, manager.saveErr
}

type recordingBrowser struct {
	urls []string
	err  error
}

func (manager *recordingBrowser) OpenURL(_ context.Context, rawURL string) error {
	manager.urls = append(manager.urls, rawURL)
	return manager.err
}

func TestDesktopDialogAdaptersPreserveNativeOptionsAndOutcomes(t *testing.T) {
	t.Parallel()

	existing := t.TempDir()
	missing := filepath.Join(existing, "Campaigns", "Vault 33")
	tests := []struct {
		name string
		run  func(*Desktop) (string, error)
		set  func(*recordingDialogs)
		want func(*testing.T, *recordingDialogs)
	}{
		{
			name: "open uses nearest existing ancestor and resolves aliases",
			run:  func(desktop *Desktop) (string, error) { return desktop.OpenFile(missing) },
			set:  func(dialogs *recordingDialogs) { dialogs.openResult = filepath.Join(existing, "campaign.json") },
			want: func(t *testing.T, dialogs *recordingDialogs) {
				assert.Equal(t, "Open Fallout Terminal Session", dialogs.openOptions.Title)
				assert.Equal(t, existing, dialogs.openOptions.DefaultDirectory)
				assert.Empty(t, dialogs.openOptions.DefaultFilename)
				assert.True(t, dialogs.openOptions.ResolvesAliases)
				assert.Equal(t, []FileFilter{{DisplayName: "Fallout Terminal sessions (*.json)", Pattern: "*.json"}}, dialogs.openOptions.Filters)
			},
		},
		{
			name: "save keeps filename and permits directory creation",
			run: func(desktop *Desktop) (string, error) {
				return desktop.SaveFile(filepath.Join(missing, "overseer.json"))
			},
			set: func(dialogs *recordingDialogs) { dialogs.saveResult = filepath.Join(existing, "overseer.json") },
			want: func(t *testing.T, dialogs *recordingDialogs) {
				assert.Equal(t, "Save Fallout Terminal Session", dialogs.saveOptions.Title)
				assert.Equal(t, existing, dialogs.saveOptions.DefaultDirectory)
				assert.Equal(t, "overseer.json", dialogs.saveOptions.DefaultFilename)
				assert.True(t, dialogs.saveOptions.CanCreateDirectories)
				assert.Equal(t, []FileFilter{{DisplayName: "Fallout Terminal sessions (*.json)", Pattern: "*.json"}}, dialogs.saveOptions.Filters)
			},
		},
		{
			name: "cancel remains empty without error",
			run:  func(desktop *Desktop) (string, error) { return desktop.OpenFile(existing) },
			set:  func(*recordingDialogs) {},
			want: func(t *testing.T, dialogs *recordingDialogs) {
				assert.NotEmpty(t, dialogs.openOptions.Title)
			},
		},
		{
			name: "native error is preserved",
			run: func(desktop *Desktop) (string, error) {
				return desktop.SaveFile(filepath.Join(existing, "session.json"))
			},
			set: func(dialogs *recordingDialogs) { dialogs.saveErr = errors.New("dialog unavailable") },
			want: func(t *testing.T, dialogs *recordingDialogs) {
				assert.NotEmpty(t, dialogs.saveOptions.Title)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dialogs := &recordingDialogs{}
			test.set(dialogs)
			desktop := NewDesktopWithManagers(t.Context(), dialogs, &recordingBrowser{})
			got, err := test.run(desktop)
			test.want(t, dialogs)
			switch test.name {
			case "cancel remains empty without error":
				require.NoError(t, err)
				assert.Empty(t, got)
			case "native error is preserved":
				require.EqualError(t, err, "dialog unavailable")
				assert.Empty(t, got)
			default:
				require.NoError(t, err)
				assert.NotEmpty(t, got)
			}
		})
	}
}

func TestWailsFileFilterPatternUsesDarwinExtensionsAndPreservesOtherGlobs(t *testing.T) {
	t.Parallel()

	got := wailsFileFilterPattern("*.json;*.yaml")
	if runtime.GOOS == "darwin" {
		require.Equal(t, "json;yaml", got)
		return
	}
	require.Equal(t, "*.json;*.yaml", got)
}

func TestDesktopBrowserAdapterAllowsOnlyAbsoluteHTTPAndHTTPS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rawURL  string
		wantURL string
		wantErr string
	}{
		{name: "http", rawURL: "http://127.0.0.1:3690/player", wantURL: "http://127.0.0.1:3690/player"},
		{name: "https", rawURL: "https://players.example.test/session", wantURL: "https://players.example.test/session"},
		{name: "file", rawURL: "file:///etc/passwd", wantErr: "external URL must be an absolute HTTP or HTTPS URL"},
		{name: "javascript", rawURL: "javascript:alert(1)", wantErr: "external URL must be an absolute HTTP or HTTPS URL"},
		{name: "relative", rawURL: "/player", wantErr: "external URL must be an absolute HTTP or HTTPS URL"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			browser := &recordingBrowser{}
			desktop := NewDesktopWithManagers(t.Context(), &recordingDialogs{}, browser)
			err := desktop.OpenURL(test.rawURL)
			if test.wantErr != "" {
				require.EqualError(t, err, test.wantErr)
				require.Empty(t, browser.urls)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []string{test.wantURL}, browser.urls)
		})
	}
}

func TestDesktopAdaptersRejectUseOutsideApplicationLifetime(t *testing.T) {
	t.Parallel()

	desktop := NewDesktopWithManagers(t.Context(), &recordingDialogs{}, &recordingBrowser{})
	require.NoError(t, desktop.Close(t.Context()))
	_, err := desktop.OpenFile("")
	require.ErrorIs(t, err, errDesktopNotReady)
	_, err = desktop.SaveFile("")
	require.ErrorIs(t, err, errDesktopNotReady)
	require.ErrorIs(t, desktop.OpenURL("https://example.test"), errDesktopNotReady)
}
