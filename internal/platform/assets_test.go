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

func TestPlayerHiddenStatesStayOutOfInactiveLayout(t *testing.T) {
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
		".term-body{flex:11auto;display:flex;flex-direction:column;min-height:0;overflow:hidden;}",
		".term-idle{height:100%;display:flex;",
		".term-entry{height:100%;min-height:0;display:flex;flex-direction:column;overflow:hidden;",
		".hack-board{height:100%;display:flex;",
		".hack-blocked{height:100%;display:flex;",
	} {
		if !strings.Contains(compactCSS, fragment) {
			t.Errorf("player stylesheet no longer exercises the hidden-layout regression fixture %q", fragment)
		}
	}
	if !strings.Contains(compactCSS, "[hidden]{display:none!important;}") {
		t.Error("player stylesheet must make the hidden attribute authoritative so inactive state containers occupy no layout space")
	}
}

func TestPlayerDesktopResponsiveLayoutContract(t *testing.T) {
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
	if !strings.Contains(html, `content="width=device-width, initial-scale=1.0"`) {
		t.Error("player viewport must follow the browser width at the default zoom")
	}
	if strings.Contains(html, "maximum-scale") {
		t.Error("player viewport must not prevent accessibility zoom")
	}
	for _, fragment := range []string{
		`class="term-header" id="normalHeader"`,
		`class="term-body" id="termBody"`,
		`class="term-footer"`,
		`class="page-nav" id="pageNav"`,
		`class="page-btn" id="pagePrev"`,
		`class="page-indicator" id="pageIndicator"`,
		`class="page-btn" id="pageNext"`,
		`class="term-prompt" id="termPrompt"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Errorf("player markup is missing persistent responsive region %q", fragment)
		}
	}

	css := read("client/client.css")
	compactCSS := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "").Replace(css)
	for _, fragment := range []string{
		"--terminal-scale:clamp(8px,min(1.5625vw,2.6667vh),24px);",
		"--font-chrome:calc(var(--terminal-scale)*.875);",
		"--font-body:var(--terminal-scale);",
		"--font-menu:calc(var(--terminal-scale)*1.0625);",
		"--font-title:calc(var(--terminal-scale)*1.1875);",
		"--font-hack:var(--terminal-scale);",
		".screen{position:relative;width:100%;max-width:1500px;height:100%;max-height:920px;min-width:0;min-height:0;overflow:hidden;",
		".hdr-intro{max-height:min(18vh,8rem);overflow:hidden;overflow-wrap:anywhere;}",
		".term-output{flex:0030%;min-height:calc(var(--terminal-scale)*2.8);",
		".page-nav{min-width:0;display:flex;align-items:center;justify-content:flex-end;",
		".term-footer:has(.back-btn[hidden]):has(.page-nav[hidden]){min-height:0;margin-top:0;}",
		"@media(max-height:720px){:root{--screen-pad-y:max(4px,calc(var(--terminal-scale)*.5));",
		".hack-board.hack-compact.hack-stacked{",
		".hack-board.hack-stacked{flex-direction:column;",
	} {
		if !strings.Contains(compactCSS, fragment) {
			t.Errorf("player stylesheet is missing responsive layout contract %q", fragment)
		}
	}

	for _, forbidden := range []string{"overflow:auto", "overflow-y:auto", "overflow:scroll", "overflow-y:scroll", "scrollbar"} {
		if strings.Contains(compactCSS, forbidden) {
			t.Errorf("player stylesheet must not expose browser or localized scrolling; found %q", forbidden)
		}
	}

	js := read("client/client.js")
	for _, fragment := range []string{
		"function paginateText(container, text)",
		"function naturalPageBreak(text, start, fittedEnd)",
		"pagedView.index = Math.min(previousIndex, pagedView.pages.length - 1)",
		"pagePrev.hidden = pagedView.index === 0",
		"pageNext.hidden = pagedView.index >= pagedView.pages.length - 1",
		"pageIndicator.value = `${pagedView.index + 1} / ${pagedView.pages.length}`",
		"activatePagination('entry', viewEntryId",
		"activatePagination('command', currentCommandNodeId",
		"e.key === 'ArrowLeft' || e.key === 'PageUp'",
		"e.key === 'ArrowRight' || e.key === 'PageDown'",
		"window.addEventListener('resize', scheduleRepagination)",
		"new ResizeObserver(scheduleRepagination)",
		"document.fonts.ready.then(scheduleRepagination)",
		"function scheduleHackFit()",
		"hackBoard.classList.toggle('hack-stacked'",
		"hackBoard.classList.toggle('hack-compact'",
		"function renderHackScreen() {\n  deactivatePagination();",
		"renderHackInputPreview();\n  scheduleHackFit();",
	} {
		if !strings.Contains(js, fragment) {
			t.Errorf("player script is missing pagination contract %q", fragment)
		}
	}
	if strings.Contains(js, ".scrollTop") || strings.Contains(js, ".scrollTo(") {
		t.Error("player script must not navigate terminal content through scrolling")
	}
}

func TestPlayerHackingSingleScreenContract(t *testing.T) {
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
		`class="term-header" id="hackHeader" hidden`,
		`class="hack-board" id="hackBoard" hidden`,
		`class="hack-columns" id="hackColumns"`,
		`class="hack-log" id="hackLog"`,
		`class="hack-input-line"`,
		`class="page-nav" id="pageNav" aria-label="Навигация по страницам" hidden`,
	} {
		if !strings.Contains(html, fragment) {
			t.Errorf("player markup is missing single-screen hacking region %q", fragment)
		}
	}

	css := read("client/client.css")
	compactCSS := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "").Replace(css)
	for _, fragment := range []string{
		"--font-body:var(--terminal-scale);",
		"--font-hack:var(--terminal-scale);",
		".hack-board{height:100%;display:flex;align-items:stretch;",
		".hack-columns{flex:11auto;display:flex;",
		".hack-log-panel{flex:0132%;display:grid;grid-template-rows:1frauto;",
		".hack-board.hack-tight.hack-stacked{gap:0;}",
		".hack-board.hack-tight.hack-stacked.hack-columns{flex-shrink:0;}",
		".hack-board.hack-tight.hack-stacked.hack-log-panel{min-height:0;row-gap:0;padding-top:0;}",
	} {
		if !strings.Contains(compactCSS, fragment) {
			t.Errorf("player stylesheet is missing single-screen hacking contract %q", fragment)
		}
	}
	for _, forbidden := range []string{"overflow:auto", "overflow-y:auto", "overflow:scroll", "overflow-y:scroll", "scrollbar"} {
		if strings.Contains(compactCSS, forbidden) {
			t.Errorf("hacking layout must not rely on scrolling; found %q", forbidden)
		}
	}

	js := read("client/client.js")
	start := strings.Index(js, "function renderHackScreen()")
	end := strings.Index(js, "function buildColumnHtml")
	if start < 0 || end <= start {
		t.Fatal("player script is missing the hacking render boundary")
	}
	hackingRender := js[start:end]
	if !strings.Contains(hackingRender, "deactivatePagination();") {
		t.Error("hacking render must disable information-view pagination")
	}
	if strings.Contains(hackingRender, "\n  activatePagination(") || strings.Contains(hackingRender, "\n  paginateText(") {
		t.Error("hacking render must not paginate the board or activity log")
	}
	for _, fragment := range []string{
		"function regionContains(parent, child)",
		"hackColumns.querySelectorAll('.hack-row')",
		"Array.from(hackLog.children)",
		"regions.some(regionOverflows)",
		"containedRegions.some(([parent, child]) => !regionContains(parent, child))",
		"hackBoard.classList.add('hack-tight')",
	} {
		if !strings.Contains(js, fragment) {
			t.Errorf("player script is missing rendered hacking geometry contract %q", fragment)
		}
	}
}

func TestPlayerHackingCheatPathsAreRemoved(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	playerScript, err := os.ReadFile(filepath.Join(root, "client", "client.js"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"HACK_ADMIN",
		"val === '1'",
		"SUCCESS",
		"URLSearchParams",
		"location.search",
		"forceHackSuccess",
		"ForceHackSuccess",
	} {
		if strings.Contains(string(playerScript), forbidden) {
			t.Errorf("bundled player still exposes removed hacking shortcut %q", forbidden)
		}
	}
	if !strings.Contains(string(playerScript), "send({ type: 'HACK_GUESS', targetId: cell.dataset.target });") {
		t.Error("ordinary candidate and filler cells must continue through HACK_GUESS")
	}
}

func TestPlayerHackingPatternInteractionContract(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "client", "client.js"))
	if err != nil {
		t.Fatal(err)
	}
	playerScript := string(raw)
	for _, required := range []string{
		"function patternAtCell(cell)",
		"(hack.patterns || [])",
		"pattern.row === row && offset >= pattern.start && offset <= pattern.end",
		"matches.find(pattern => pattern.start === offset)",
		"pattern.start > nearest.start",
		"const pattern = patternAtCell(cell)",
		"if (patternAtCell(cell))",
		"offset >= pattern.start && offset <= pattern.end",
		"`[data-row=\"${pattern.row}\"][data-offset]`",
		"if (pattern.used) setHackPatternHover(null)",
		"send({ type: 'HACK_PATTERN', patternId: pattern.id })",
	} {
		if !strings.Contains(playerScript, required) {
			t.Errorf("bundled player is missing pattern interaction contract %q", required)
		}
	}
	for _, forbidden := range []string{
		"pattern.column",
		"pattern.pair",
		"column.text.slice(pattern.start",
		"data-column=\"${pattern.column}",
	} {
		if strings.Contains(playerScript, forbidden) {
			t.Errorf("bundled player depends on private pattern metadata %q", forbidden)
		}
	}
}

func TestPlayerHackingCamouflageAndDelimiterContract(t *testing.T) {
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

	playerScript := read("client/client.js")
	for _, required := range []string{
		"const HACK_DELIMITERS = '()[]{}<>';",
		"function isDelimiterCell(cell)",
		"HACK_DELIMITERS.includes(cell.textContent)",
		"const pattern = patternAtCell(cell)",
		"if (isDelimiterCell(cell))",
		"if (pattern && !pattern.used)",
		"send({ type: 'HACK_PATTERN', patternId: pattern.id });",
		"send({ type: 'HACK_GUESS', targetId: cell.dataset.target });",
		"class=\"hcell word\"",
		"class=\"hcell filler\"",
	} {
		if !strings.Contains(playerScript, required) {
			t.Errorf("bundled player is missing camouflage interaction contract %q", required)
		}
	}
	for _, forbidden := range []string{
		"classList.add('pattern')",
		"classList.add('valid-pattern')",
		"classList.add('delimiter-decoy')",
		"classList.add('decoy')",
		"data-pattern-valid",
		"data-delimiter-decoy",
	} {
		if strings.Contains(playerScript, forbidden) {
			t.Errorf("bundled player exposes persistent pattern validity through %q", forbidden)
		}
	}

	stylesheet := read("client/client.css")
	compactCSS := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "").Replace(stylesheet)
	for _, required := range []string{
		".hcell{cursor:pointer;}",
		".hcell.filler{opacity:.8;}",
		".hcell.hi{background:#57ff6e;color:#021002;text-shadow:none;}",
	} {
		if !strings.Contains(compactCSS, required) {
			t.Errorf("player stylesheet is missing static/transient cell styling contract %q", required)
		}
	}
	for _, forbidden := range []string{".pattern{", ".valid-pattern{", ".delimiter-decoy{", ".decoy{"} {
		if strings.Contains(compactCSS, forbidden) {
			t.Errorf("player stylesheet exposes persistent delimiter validity through %q", forbidden)
		}
	}
}

func TestGameMasterRetainsExclusiveHackSolveControl(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	read := func(parts ...string) string {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(append([]string{root}, parts...)...))
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	masterHTML := read("frontend", "src", "index.html")
	masterJS := read("frontend", "src", "master.js")
	desktopAPI := read("frontend", "src", "desktop-api.js")
	playerJS := read("client", "client.js")
	playerHTML := read("client", "index.html")
	playerProtocol := read("internal", "player", "protocol.go")
	appBoundary := read("app.go")
	for _, required := range []string{
		`id="btnHackSuccess"`,
		"desktopAPI.forceHackSuccess()",
		"h.solved || h.failed",
		"forceHackSuccess: 'ForceHackSuccess'",
		"func (app *App) ForceHackSuccess() CommandResult",
	} {
		if !strings.Contains(masterHTML+masterJS+desktopAPI+appBoundary, required) {
			t.Errorf("game-master bundle is missing solve-control contract %q", required)
		}
	}
	playerSurface := playerJS + playerHTML + playerProtocol
	for _, forbidden := range []string{
		"forceHackSuccess",
		"ForceHackSuccess",
		"btnHackSuccess",
		"HACK_ADMIN",
		"URLSearchParams",
		"location.search",
	} {
		if strings.Contains(playerSurface, forbidden) {
			t.Errorf("player surface gained game-master solve authority %q", forbidden)
		}
	}
}

func TestPlayerHackingColumnFontFitContract(t *testing.T) {
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

	css := read("client/client.css")
	compactCSS := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "").Replace(css)
	for _, fragment := range []string{
		"--font-hack:var(--terminal-scale);",
		"--hack-row-font:var(--font-hack);",
		".hack-row{display:flex;gap:clamp(4px,.8vw,12px);font-size:var(--hack-row-font);",
	} {
		if !strings.Contains(compactCSS, fragment) {
			t.Errorf("player stylesheet is missing shared hacking-row fit contract %q", fragment)
		}
	}
	if !strings.Contains(css, "'Fixedsys', 'Consolas', monospace") {
		t.Error("hacking-row fit must retain the production fallback font stack for metric remeasurement")
	}

	js := read("client/client.js")
	for _, fragment := range []string{
		"function hackRowsFitColumns()",
		"const tolerance = 0.5",
		"finalBounds.right <= columnBounds.right + tolerance",
		"rowBounds.bottom <= columnBounds.bottom + tolerance",
		"function fitHackRowFont()",
		"hackBoard.style.removeProperty('--hack-row-font')",
		"let low = baseSize",
		"Math.min(...columns.map(column => column.getBoundingClientRect().width))",
		"hackRowsFitColumns() && !hackContentOverflows()",
		"while (high - low > 0.25)",
		"hackBoard.style.setProperty('--hack-row-font', `${size}px`)",
		"fitHackRowFont();",
		"window.addEventListener('resize', scheduleHackFit)",
		"hackFitObserver.observe(termBody)",
		"document.fonts.ready.then(scheduleHackFit)",
		"if (hackFitFrame !== null) cancelAnimationFrame(hackFitFrame)",
	} {
		if !strings.Contains(js, fragment) {
			t.Errorf("player script is missing column-aware hacking-row fit contract %q", fragment)
		}
	}
	if strings.Contains(js, "hackFitObserver.observe(hackBoard)") || strings.Contains(js, "hackFitObserver.observe(hackColumns)") {
		t.Error("hacking-row fit must not observe its own font-sized descendants and create a resize feedback loop")
	}
}

func TestActiveFrontendUsesRuntimeNeutralDesktopFacade(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	adapter, err := os.ReadFile(filepath.Join(root, "frontend", "src", "desktop-api.js"))
	if err != nil {
		t.Fatal(err)
	}
	master, err := os.ReadFile(filepath.Join(root, "frontend", "src", "master.js"))
	if err != nil {
		t.Fatal(err)
	}

	activeSource := string(adapter) + "\n" + string(master)
	if strings.Contains(activeSource, "window.electronAPI") || strings.Contains(activeSource, "'electronAPI'") || strings.Contains(activeSource, `"electronAPI"`) {
		t.Error("active production frontend still defines or consumes the transitional Electron-specific bridge global")
	}
	for _, required := range []string{
		"window.desktopAPI",
		"Object.defineProperty(window, 'desktopAPI'",
	} {
		if !strings.Contains(activeSource, required) {
			t.Errorf("active production frontend is missing runtime-neutral facade contract %q", required)
		}
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
