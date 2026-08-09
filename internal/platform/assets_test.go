package platform

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
)

func TestMasterAssetManifestSupportsCleanCheckoutAndBuiltOutput(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	assertNonEmptyFiles(t, root, []string{
		"frontend/src/index.html",
		"frontend/src/master.css",
		"frontend/src/master.js",
		"frontend/src/desktop-api.js",
		"frontend/src/Fixedsys.ttf",
	})

	distRoot := filepath.Join(root, "frontend", "dist")
	if info, err := os.Stat(filepath.Join(distRoot, ".keep")); err != nil || info.IsDir() {
		t.Fatalf("frontend/dist/.keep must preserve the go:embed root on a clean checkout: %v", err)
	}

	builtFiles := make([]string, 0)
	err := filepath.WalkDir(distRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() == ".keep" {
			return nil
		}
		relative, err := filepath.Rel(distRoot, path)
		if err != nil {
			return err
		}
		builtFiles = append(builtFiles, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(builtFiles) == 0 {
		return
	}

	assertNonEmptyFiles(t, distRoot, []string{"index.html"})
	for extension, description := range map[string]string{
		".js":  "JavaScript bundle",
		".css": "stylesheet bundle",
		".ttf": "Fixedsys font bundle",
	} {
		if !containsExtension(builtFiles, extension) {
			t.Errorf("built master output is missing a %s (%s); files: %v", description, extension, builtFiles)
		}
	}
}

func TestRetainedPlayerAssetAndSoundManifest(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	assertNonEmptyFiles(t, root, []string{
		"client/index.html",
		"client/client.css",
		"client/client.js",
		"client/sound.js",
		"client/fonts/Fixedsys.ttf",
	})

	requiredCategories := []string{
		"ambient",
		"charscroll",
		"enter",
		"hack-bad",
		"hack-good",
		"menu-focus",
		"multiple",
		"single",
	}
	allowedExtensions := map[string]struct{}{
		".mp3": {}, ".wav": {}, ".ogg": {}, ".m4a": {}, ".webm": {},
	}
	for _, category := range requiredCategories {
		category := category
		t.Run(category, func(t *testing.T) {
			directory := filepath.Join(root, "client", "sounds", category)
			entries, err := os.ReadDir(directory)
			if err != nil {
				t.Fatalf("read required sound category %q: %v", category, err)
			}
			files := make([]string, 0)
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if _, allowed := allowedExtensions[strings.ToLower(filepath.Ext(entry.Name()))]; !allowed {
					continue
				}
				info, err := entry.Info()
				if err != nil {
					t.Fatal(err)
				}
				if info.Size() > 0 {
					files = append(files, entry.Name())
				}
			}
			sort.Strings(files)
			if len(files) == 0 {
				t.Fatalf("required sound category %q has no non-empty supported audio asset", category)
			}
		})
	}
}

func TestPlayerHiddenStatesStayOutOfScrollableLayout(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	read := func(path string) string {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}

	html := read("client/index.html")
	for _, fragment := range []string{
		`class="term-body" id="termBody"`,
		`class="term-idle" id="termIdle"`,
		`class="hack-board" id="hackBoard" hidden`,
		`class="hack-blocked" id="hackBlocked" hidden`,
	} {
		if !strings.Contains(html, fragment) {
			t.Errorf("player markup is missing visibility fixture %q", fragment)
		}
	}

	css := read("client/client.css")
	compactCSS := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "").Replace(css)
	for _, fragment := range []string{
		".term-body{flex:11auto;overflow-y:auto;min-height:0;}",
		".term-idle{height:100%;display:flex;",
		".hack-board{height:100%;display:flex;",
		".hack-blocked{height:100%;display:flex;",
	} {
		if !strings.Contains(compactCSS, fragment) {
			t.Errorf("player stylesheet no longer exercises the overflow/display regression fixture %q", fragment)
		}
	}
	if !strings.Contains(compactCSS, "[hidden]{display:none!important;}") {
		t.Error("player stylesheet must make the hidden attribute authoritative so inactive state containers occupy no scrollable layout space")
	}
}

func TestBundledDemoManifestIsValidAndResolvesFromResources(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	demoPath := filepath.Join(root, "sessions", "demo.json")
	raw, err := os.ReadFile(demoPath)
	if err != nil {
		t.Fatal(err)
	}
	session, err := domain.DecodeSession(raw)
	if err != nil {
		t.Fatalf("bundled demo is not a valid version-1 session: %v", err)
	}
	if session.Version != 1 || len(session.Terminals) == 0 {
		t.Fatalf("bundled demo = version %d with %d terminals, want version 1 with content", session.Version, len(session.Terminals))
	}

	locations, err := NewSessionLocations(filepath.Join(root, ".manifest-home"), root)
	if err != nil {
		t.Fatal(err)
	}
	if locations.BundledDemo != demoPath {
		t.Fatalf("BundledDemo = %q, want manifest path %q", locations.BundledDemo, demoPath)
	}
}

func TestProductionEmbedsMasterAndPlayerAsSeparateFilesystems(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	requiredFragments := []string{
		"//go:embed all:frontend/dist\nvar frontendSource embed.FS",
		"//go:embed all:client\nvar playerSource embed.FS",
		`fs.Sub(frontendSource, "frontend/dist")`,
		`fs.Sub(playerSource, "client")`,
		"Assets: frontendAssets",
		"composeApplication(playerAssets)",
		"Assets: playerAssets",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(source, fragment) {
			t.Errorf("main.go is missing production asset wiring %q", fragment)
		}
	}
	if strings.Contains(source, "//go:embed all:frontend/dist all:client") ||
		strings.Contains(source, "//go:embed all:client all:frontend/dist") {
		t.Error("master and remote-player assets share one embed directive; their serving boundaries must remain separate")
	}

	viteConfig, err := os.ReadFile(filepath.Join(root, "frontend", "vite.config.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(viteConfig), `./dist/.keep`) {
		t.Error("Vite build does not restore frontend/dist/.keep after emptyOutDir")
	}
}

func assertNonEmptyFiles(t *testing.T, root string, paths []string) {
	t.Helper()
	for _, path := range paths {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Errorf("required asset %q: %v", path, err)
			continue
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			t.Errorf("required asset %q is not a non-empty regular file", path)
		}
	}
}

func containsExtension(paths []string, extension string) bool {
	for _, path := range paths {
		if strings.EqualFold(filepath.Ext(path), extension) {
			return true
		}
	}
	return false
}

func assetRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve asset-manifest test location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
