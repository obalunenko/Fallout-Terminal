package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	_ "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/config/v1"
	_ "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/persistence/v1"
	playerv1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1"
	_ "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/private/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestProtobufContractShapeAndSeparation(t *testing.T) {
	t.Parallel()

	service := playerv1.File_fallout_terminal_player_v1_player_proto.Services().ByName("PlayerService")
	if service == nil {
		t.Fatal("public descriptor is missing PlayerService")
	}
	wantMethods := []string{"Subscribe", "SelectCharacter", "Navigate", "Guess", "ActivatePattern", "SoundManifest"}
	if service.Methods().Len() != len(wantMethods) {
		t.Fatalf("PlayerService methods = %d, want %d", service.Methods().Len(), len(wantMethods))
	}
	for index, want := range wantMethods {
		method := service.Methods().Get(index)
		if got := string(method.Name()); got != want {
			t.Errorf("PlayerService method %d = %q, want %q", index, got, want)
		}
		if want == "Subscribe" {
			if method.IsStreamingClient() || !method.IsStreamingServer() {
				t.Errorf("Subscribe must be server-streaming only")
			}
		} else if method.IsStreamingClient() || method.IsStreamingServer() {
			t.Errorf("%s must be unary", want)
		}
	}

	var publicFiles []protoreflect.FileDescriptor
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if strings.HasPrefix(string(file.Package()), "fallout.terminal.player.v1") {
			publicFiles = append(publicFiles, file)
		}
		return true
	})
	if len(publicFiles) == 0 {
		t.Fatal("generated public descriptor graph is empty")
	}
	for _, file := range publicFiles {
		imports := file.Imports()
		for index := 0; index < imports.Len(); index++ {
			imported := imports.Get(index)
			if !strings.HasPrefix(imported.Path(), "fallout/terminal/player/v1/") {
				t.Errorf("public descriptor %s imports non-public schema %s", file.Path(), imported.Path())
			}
		}
	}

	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if !strings.HasPrefix(string(file.Package()), "fallout.terminal.") {
			return true
		}
		checkEnumZeroValues(t, file.Enums())
		checkMessageContractShape(t, file.Messages())
		return true
	})

	optionalFields := map[protoreflect.FullName]bool{
		"fallout.terminal.player.v1.PlayerState.broadcast_id":                     true,
		"fallout.terminal.player.v1.PlayerState.active_terminal_id":               true,
		"fallout.terminal.player.v1.SubscribeRequest.recognition_handle":          true,
		"fallout.terminal.player.v1.NavigationState.view_entry_id":                true,
		"fallout.terminal.player.v1.NavigationState.command_node_id":              true,
		"fallout.terminal.persistence.v1.Session.player_config":                   true,
		"fallout.terminal.config.v1.TunnelConfig.policy_parent":                   true,
		"fallout.terminal.private.v1.CharacterState.logical_session_id":           true,
		"fallout.terminal.private.v1.LogicalSessionState.character_id":            true,
		"fallout.terminal.private.v1.BroadcastState.active_controller_session_id": true,
		"fallout.terminal.private.v1.BroadcastState.active_terminal_id":           true,
	}
	for name := range optionalFields {
		descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
		if err != nil {
			t.Errorf("semantic optional field %s missing: %v", name, err)
			continue
		}
		field, ok := descriptor.(protoreflect.FieldDescriptor)
		if !ok || !field.HasOptionalKeyword() {
			t.Errorf("%s must use proto3 optional presence", name)
		}
	}

	for _, name := range []protoreflect.FullName{
		"fallout.terminal.player.v1.SubscriptionMessage.payload",
		"fallout.terminal.player.v1.NavigateRequest.action",
		"fallout.terminal.player.v1.GuessRequest.target",
		"fallout.terminal.player.v1.ContentNode.content",
		"fallout.terminal.player.v1.TerminalPresentation.presentation",
		"fallout.terminal.persistence.v1.ContentNode.content",
	} {
		descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
		if err != nil {
			t.Errorf("required oneof %s missing: %v", name, err)
			continue
		}
		if _, ok := descriptor.(protoreflect.OneofDescriptor); !ok {
			t.Errorf("%s is not a oneof descriptor", name)
		}
	}
}

func TestProtobufSchemaRevisionMatchesSources(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	paths, err := filepath.Glob(filepath.Join(root, "proto", "fallout", "terminal", "*", "v1", "*.proto"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	outer := sha256.New()
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		inner := sha256.Sum256(raw)
		relative, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		outer.Write([]byte(hex.EncodeToString(inner[:]) + "  " + filepath.ToSlash(relative) + "\n"))
	}
	wantRaw, err := os.ReadFile(filepath.Join(root, "proto", "schema-revision.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := hex.EncodeToString(outer.Sum(nil)), strings.TrimSpace(string(wantRaw)); got != want {
		t.Fatalf("schema revision = %s, want %s; run scripts/proto-generate.sh after schema edits", got, want)
	}
}

func checkEnumZeroValues(t *testing.T, enums protoreflect.EnumDescriptors) {
	t.Helper()
	for index := 0; index < enums.Len(); index++ {
		enum := enums.Get(index)
		if enum.Values().Len() == 0 || enum.Values().Get(0).Number() != 0 || !strings.HasSuffix(string(enum.Values().Get(0).Name()), "_UNSPECIFIED") {
			t.Errorf("enum %s must define an UNSPECIFIED zero value", enum.FullName())
		}
	}
}

func checkMessageContractShape(t *testing.T, messages protoreflect.MessageDescriptors) {
	t.Helper()
	for index := 0; index < messages.Len(); index++ {
		message := messages.Get(index)
		checkEnumZeroValues(t, message.Enums())
		checkMessageContractShape(t, message.Messages())
		fields := message.Fields()
		for fieldIndex := 0; fieldIndex < fields.Len(); fieldIndex++ {
			field := fields.Get(fieldIndex)
			if message.ReservedNames().Has(field.Name()) || message.ReservedRanges().Has(field.Number()) {
				t.Errorf("active field %s collides with a reserved identifier", field.FullName())
			}
		}
	}
}

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

func TestPlayerHackingOutcomeAudioUsesEligibleAuthoritativeTransitions(t *testing.T) {
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

	soundScript := read("client/sound.js")
	for _, required := range []string{
		"let webAudioEligible = false;",
		"const folderLoads = new Map();",
		"const rawLoads    = new Map();",
		"const oneShotFolders = ['single', 'multiple', 'enter', 'hack-good', 'hack-bad', 'menu-focus', 'charscroll'];",
		"function enableWebAudio()",
		"function reportPlayback(url)",
		"if (!webAudioReady || !await webAudioReady || !webAudioEligible) {",
		"await Promise.all(oneShotFolders.map(loadFolder));",
		"const buffer = await context.decodeAudioData(raw.slice(0));",
		"if (!rawBufs.has(url)) await prefetch(url);",
		"await Promise.all(supported.map(prefetch));",
		"enableWebAudio();",
		"reportPlayback(url);",
		"playFromFolder('single', 0.55)",
		"playFromFolder('multiple', 0.55)",
		"playFromFolder('enter', 0.65)",
		"playFirst('menu-focus', 0.5)",
		"playFirst('hack-good', 0.8)",
		"playFirst('hack-bad', 0.7)",
		"playFromFolder('charscroll', 0.4)",
	} {
		if !strings.Contains(soundScript, required) {
			t.Errorf("player sound adapter is missing outcome-audio contract %q", required)
		}
	}

	playerScript := read("client/client.js")
	outcomeStart := strings.Index(playerScript, "function playHackOutcomeTransition(previousHack, nextHack, revision = appliedSharedRevision) {")
	if outcomeStart < 0 {
		t.Fatal("player script is missing the common authoritative hacking-outcome boundary")
	}
	outcomeEnd := strings.Index(playerScript[outcomeStart:], "function scheduleHackSolvedNavigation() {")
	if outcomeEnd < 0 {
		t.Fatal("player script is missing the common authoritative hacking-outcome boundary")
	}
	outcomeBoundary := playerScript[outcomeStart : outcomeStart+outcomeEnd]
	for _, required := range []string{
		"nextHack.attemptsLeft < previousHack.attemptsLeft",
		"playHackBad();",
		"nextHack.solved && !previousHack.solved",
		"playHackGood();",
	} {
		if !strings.Contains(outcomeBoundary, required) {
			t.Errorf("common authoritative outcome handler is missing transition guard %q", required)
		}
	}
	for _, required := range []string{
		"let terminalLiveBaselinePending = true;",
		"const isContinuousTerminalUpdate = !terminalLiveBaselinePending && hasLive",
		"if (isContinuousTerminalUpdate) playHackOutcomeTransition(previousHack, hack);",
		"terminalLiveBaselinePending = false;",
		"playHackOutcomeTransition(previousHack, hack);",
	} {
		if !strings.Contains(playerScript, required) {
			t.Errorf("player script is missing revisioned live/hack outcome guard %q", required)
		}
	}

	actionResultStart := strings.Index(playerScript, "async function applyMutationResult(operation) {")
	if actionResultStart < 0 {
		t.Fatal("player script is missing typed mutation-result boundary")
	}
	actionResultEnd := strings.Index(playerScript[actionResultStart:], "function actionReasonName(reason) {")
	if actionResultEnd < 0 {
		t.Fatal("player script is missing typed mutation-result boundary")
	}
	if strings.Contains(playerScript[actionResultStart:actionResultStart+actionResultEnd], "playHack") {
		t.Error("ACTION_RESULT must not optimistically play hacking outcome audio")
	}

	beginActionStart := strings.Index(playerScript, "function beginSharedMutation(procedure, invoke) {")
	if beginActionStart < 0 {
		t.Fatal("player script is missing the shared-action presentation boundary")
	}
	beginActionEnd := strings.Index(playerScript[beginActionStart:], "function selectCharacter(characterID) {")
	if beginActionEnd < 0 {
		t.Fatal("player script is missing the shared-action presentation boundary")
	}
	beginAction := playerScript[beginActionStart : beginActionStart+beginActionEnd]
	if !strings.Contains(beginAction, "renderPlayerContext();") || strings.Contains(beginAction, "render();") {
		t.Error("beginSharedAction must update pending presentation without rebuilding the hacking board")
	}
	for _, required := range []string{
		"pattern.id !== hackHoverKey",
		"setHackHover(hoveredCells.length ? hackHoverKey : null, true);",
	} {
		if !strings.Contains(playerScript, required) {
			t.Errorf("player script is missing preview-audio replay guard %q", required)
		}
	}
}

func TestPlayerAmbientAudioUsesExplicitRetryableLifecycleState(t *testing.T) {
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
	functionBoundary := func(script, start, end string) string {
		t.Helper()
		startIndex := strings.Index(script, start)
		if startIndex < 0 {
			t.Fatalf("missing JavaScript boundary %q", start)
		}
		endIndex := strings.Index(script[startIndex+len(start):], end)
		if endIndex < 0 {
			t.Fatalf("missing JavaScript boundary %q after %q", end, start)
		}
		return script[startIndex : startIndex+len(start)+endIndex]
	}

	soundScript := read("client/sound.js")
	for _, state := range []string{"ambientRequested", "ambientPlayAttempt", "webAudioEligible", "webAudioReady"} {
		if !strings.Contains(soundScript, state) {
			t.Errorf("player sound adapter is missing explicit audio state %q", state)
		}
	}
	activation := functionBoundary(soundScript, "function enableWebAudio()", "async function prefetch")
	for _, boundary := range []string{
		"if (webAudioEligible)",
		"if (webAudioReady)",
		"webAudioReady = attempt",
		"webAudioReady = null",
	} {
		if !strings.Contains(activation, boundary) {
			t.Errorf("Web Audio activation is missing retry boundary %q", boundary)
		}
	}
	for _, event := range []string{"'pointerdown'", "'keydown'"} {
		if strings.Count(soundScript, "document.addEventListener("+event+", handleAudioGesture)") != 1 {
			t.Errorf("player sound adapter must observe qualifying gesture %s exactly once", event)
		}
	}

	reconcile := functionBoundary(soundScript, "function reconcileAmbient()", "function setAmbientActive(active)")
	for _, boundary := range []string{
		"ambientRequested",
		"ambientReady",
		"ambientAudio.paused",
		"ambientPlayAttempt",
		"ambientAudio.play()",
		".catch(",
		".finally(",
	} {
		if !strings.Contains(reconcile, boundary) {
			t.Errorf("ambient reconciliation is missing state boundary %q", boundary)
		}
	}
	ambientState := functionBoundary(soundScript, "function setAmbientActive(active)", "function stopAmbient()")
	for _, boundary := range []string{"Boolean(active)", "ambientRequested = requested", "reconcileAmbient()", "stopAmbient()"} {
		if !strings.Contains(ambientState, boundary) {
			t.Errorf("ambient state API is missing transition boundary %q", boundary)
		}
	}

	playerScript := read("client/client.js")
	if strings.Count(playerScript, "setAmbientActive(true);") != 1 {
		t.Error("accepted terminal-live dispatch must be the sole ambient activation boundary")
	}
	if strings.Count(playerScript, "setAmbientActive(false);") < 3 {
		t.Error("player lifecycle must revoke ambient state across clear, reset, and reconnect boundaries")
	}
	for _, obsolete := range []string{"tryStartAmbient", "stopAmbient"} {
		if strings.Contains(playerScript, obsolete) {
			t.Errorf("player lifecycle must use the explicit ambient state API, found %q", obsolete)
		}
	}
}

func TestBrowserJavaScriptUsesSpacesInsteadOfTabs(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	paths := []string{
		"client/client.js",
		"client/sound.js",
		"frontend/src/desktop-api.js",
		"frontend/src/master.js",
	}
	for _, relative := range paths {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.ContainsRune(string(raw), '\t') {
			t.Errorf("%s contains a tab; browser JavaScript uses two-space indentation", relative)
		}
	}
}

func TestMasterTerminalCommandsCannotBypassCoordinator(t *testing.T) {
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

	facade := read("frontend/src/desktop-api.js")
	app := read("app.go")
	for _, required := range []string{
		"requestTerminalActivation: 'RequestTerminalActivation'",
		"requestTerminalClear: 'RequestTerminalClear'",
		"updateLiveTerminal: 'UpdateLiveTerminal'",
		"forceHackSuccess: 'ForceHackSuccess'",
	} {
		if !strings.Contains(facade, required) {
			t.Errorf("master desktop facade is missing coordinator-owned command %q", required)
		}
	}
	for _, forbidden := range []string{"SetLiveTerminal", "ClearLiveTerminal", "setLiveTerminal", "clearLiveTerminal"} {
		if strings.Contains(facade, forbidden) {
			t.Errorf("master desktop facade exposes legacy terminal command %q", forbidden)
		}
	}
	for _, required := range []string{
		"func (app *App) RequestTerminalActivation(",
		"func (app *App) RequestTerminalClear(",
		"func (app *App) UpdateLiveTerminal(",
		"func (app *App) ForceHackSuccess(",
	} {
		if !strings.Contains(app, required) {
			t.Errorf("Wails App is missing required terminal command %q", required)
		}
	}
	for _, forbidden := range []string{
		"func (app *App) SetLiveTerminal(",
		"func (app *App) ClearLiveTerminal(",
	} {
		if strings.Contains(app, forbidden) {
			t.Errorf("Wails App still binds legacy terminal command %q", forbidden)
		}
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

func TestPlayerSessionSelectionAssetContract(t *testing.T) {
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
		`class="player-identity" id="playerIdentity" hidden`,
		`class="role-badge" id="roleBadge"`,
		`class="character-select" id="characterSelect" hidden`,
		`class="assigned-waiting" id="assignedWaiting" hidden`,
		`class="player-notice" id="playerNotice"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Errorf("player markup is missing session-selection region %q", fragment)
		}
	}
	for _, forbidden := range []string{"browserToken", "ForceHackSuccess", "forceHackSuccess", "HACK_ADMIN"} {
		if strings.Contains(html, forbidden) {
			t.Errorf("player markup exposes private/session capability %q", forbidden)
		}
	}

	css := read("client/client.css")
	compactCSS := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "").Replace(css)
	for _, fragment := range []string{
		".character-select{height:100%;min-height:0;display:flex;",
		`.character-option[data-status="claimed"]{`,
		".character-select.pending{",
		".assigned-waiting{height:100%;display:flex;",
		"[hidden]{display:none!important;}",
	} {
		if !strings.Contains(compactCSS, fragment) {
			t.Errorf("player stylesheet is missing bounded selection contract %q", fragment)
		}
	}
	for _, forbidden := range []string{"overflow:auto", "overflow-y:auto", "overflow:scroll", "overflow-y:scroll", "scrollbar"} {
		if strings.Contains(compactCSS, forbidden) {
			t.Errorf("session selection must remain within the no-scroll player layout; found %q", forbidden)
		}
	}

	js := read("client/client.js")
	for _, fragment := range []string{
		"const PLAYER_TOKEN_KEY = 'fallout-terminal.player-token'",
		"localStorage.getItem(PLAYER_TOKEN_KEY)",
		"localStorage.setItem(PLAYER_TOKEN_KEY",
		"option.textContent = entry.name",
		"playerRPC.subscribe(request",
		"playerRPC.selectCharacter({",
	} {
		if !strings.Contains(js, fragment) {
			t.Errorf("player script is missing safe selection/identity contract %q", fragment)
		}
	}
	for _, forbidden := range []string{
		"searchParams.set('browserToken'",
		`searchParams.set("browserToken"`,
		"innerHTML = entry.name",
		"ForceHackSuccess",
		"forceHackSuccess",
		"HACK_ADMIN",
	} {
		if strings.Contains(js, forbidden) {
			t.Errorf("player script exposes unsafe session/privileged path %q", forbidden)
		}
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
	if !strings.Contains(string(playerScript), "beginGuess(cell.dataset.target)") {
		t.Error("ordinary candidate and filler cells must continue through the typed Guess procedure")
	}
}

func TestPlayerSharedActionPathsAreRoleAndPendingGated(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	playerScript, err := os.ReadFile(filepath.Join(root, "client", "client.js"))
	if err != nil {
		t.Fatal(err)
	}
	js := string(playerScript)

	for _, required := range []string{
		"let pendingSharedAction = null",
		"function canControlSharedTerminal()",
		"playerState.role === 'active'",
		"playerState.phase === 'controlling'",
		"pendingSharedAction === null",
		"function beginSharedMutation(procedure, invoke)",
		"beginNavigation('enter', node.id)",
		"beginNavigation('command', node.id)",
		"beginNavigation('entry', node.id)",
		"beginNavigation('back')",
		"beginPattern(pattern.id)",
		"beginGuess(cell.dataset.target)",
		"activateRow(kids[selIndex])",
		"goBack()",
	} {
		if !strings.Contains(js, required) {
			t.Errorf("player script is missing shared-action gate %q", required)
		}
	}
	for _, forbidden := range []string{
		"send({ type: 'NAV_ACTION'",
		"send({ type: 'HACK_GUESS'",
		"send({ type: 'HACK_PATTERN'",
		"ForceHackSuccess",
		"forceHackSuccess",
		"HACK_ADMIN",
	} {
		if strings.Contains(js, forbidden) {
			t.Errorf("player script bypasses authority or exposes a privileged path %q", forbidden)
		}
	}

	stylesheet, err := os.ReadFile(filepath.Join(root, "client", "client.css"))
	if err != nil {
		t.Fatal(err)
	}
	compactCSS := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "").Replace(string(stylesheet))
	for _, forbidden := range []string{
		"#screen.observer-read-only:is(.term-row,.back-btn,.hcell){pointer-events:none",
		"#screen.shared-input-pending:is(.term-row,.back-btn,.hcell){pointer-events:none",
		"#screen.observer-read-only.hcell{pointer-events:none",
		"#screen.shared-input-pending.hcell{pointer-events:none",
	} {
		if strings.Contains(compactCSS, forbidden) {
			t.Errorf("read-only/pending presentation disables target hit-testing through %q", forbidden)
		}
	}
	for _, required := range []string{
		`class="hcell word" data-target="${esc(wid)}" tabindex="0"`,
		`class="hcell filler" data-target="${colIndex}:${i}" data-row="${rowBase + r}" data-offset="${i - rowStart}" tabindex="0"`,
		"const lines = Array.isArray(hack.log) ? hack.log : []",
	} {
		if !strings.Contains(js, required) {
			t.Errorf("active hacking target is not rendered as a focusable hit target %q", required)
		}
	}

	protocol, err := os.ReadFile(filepath.Join(root, "internal", "player", "handler.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"ForceHackSuccess", "forceHackSuccess", "HACK_ADMIN"} {
		if strings.Contains(string(protocol), forbidden) {
			t.Errorf("player protocol exposes trusted hacking capability %q", forbidden)
		}
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
		"pattern.row === row && pattern.start === offset",
		"const pattern = patternAtCell(cell)",
		"const relatedPattern = patternAtCell(related)",
		"offset >= pattern.start && offset <= pattern.end",
		"`[data-row=\"${pattern.row}\"][data-offset]`",
		"if (pattern.used) setHackPatternHover(null)",
		"beginPattern(pattern.id)",
		"beginGuess(cell.dataset.target)",
	} {
		if !strings.Contains(playerScript, required) {
			t.Errorf("bundled player is missing pattern interaction contract %q", required)
		}
	}
	for _, forbidden := range []string{
		"offset >= pattern.start && offset <= pattern.end\n  )",
		"matches.find(pattern => pattern.start === offset)",
		"pattern.start > nearest.start",
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
		"const pattern = patternAtCell(cell)",
		"pattern.row === row && pattern.start === offset",
		"if (pattern && !pattern.used)",
		"if (beginPattern(pattern.id)) playEnter();",
		"if (pattern) return;",
		"if (beginGuess(cell.dataset.target)) playEnter();",
		"class=\"hcell word\"",
		"class=\"hcell filler\"",
	} {
		if !strings.Contains(playerScript, required) {
			t.Errorf("bundled player is missing camouflage interaction contract %q", required)
		}
	}
	for _, forbidden := range []string{
		"function isDelimiterCell(cell)",
		"HACK_DELIMITERS.includes(cell.textContent)",
		"pattern || isDelimiterCell(cell)",
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
	playerProtocol := read("internal", "player", "handler.go")
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

func TestGameMasterRetainsExclusiveFailedHackResetControl(t *testing.T) {
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
	masterHTML := read("frontend/src/index.html")
	masterCSS := read("frontend/src/master.css")
	masterJS := read("frontend/src/master.js")
	desktopAPI := read("frontend/src/desktop-api.js")
	appBoundary := read("app.go")
	contract := read("specs/004-player-sessions-control/contracts/desktop-coordination.md")
	for _, required := range []string{
		`id="btnResetFailedHack"`,
		`ПОВТОРИТЬ ВЗЛОМ`,
		`desktopAPI.resetFailedHack(`,
		`resetFailedHack: 'ResetFailedHack'`,
		`func (app *App) ResetFailedHack(`,
		`ResetFailedHack`,
	} {
		if !strings.Contains(masterHTML+masterCSS+masterJS+desktopAPI+appBoundary+contract, required) {
			t.Errorf("game-master bundle is missing failed-hack reset contract %q", required)
		}
	}
	playerSurface := strings.Join([]string{
		read("client/index.html"), read("client/client.css"), read("client/client.js"),
		read("internal/player/handler.go"), read("internal/player/server.go"),
	}, "\n")
	for _, forbidden := range []string{"ResetFailedHack", "resetFailedHack", "btnResetFailedHack", "HACK_RESET", "URLSearchParams", "location.search"} {
		if strings.Contains(playerSurface, forbidden) {
			t.Errorf("player surface gained failed-hack reset authority %q", forbidden)
		}
	}
}

func TestGameMasterTerminalSwitchDecisionDialogIsAccessibleAndPrivate(t *testing.T) {
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
	openingTag := func(document, id string) string {
		t.Helper()
		marker := `id="` + id + `"`
		markerAt := strings.Index(document, marker)
		if markerAt < 0 {
			return ""
		}
		start := strings.LastIndex(document[:markerAt], "<")
		endOffset := strings.Index(document[markerAt:], ">")
		if start < 0 || endOffset < 0 {
			return ""
		}
		return document[start : markerAt+endOffset+1]
	}

	masterHTML := read("frontend/src/index.html")
	dialogTag := openingTag(masterHTML, "terminalSwitchDialog")
	for _, fragment := range []string{
		"<dialog",
		`id="terminalSwitchDialog"`,
		`aria-modal="true"`,
		`aria-labelledby="terminalSwitchDialogTitle"`,
		`aria-describedby="terminalSwitchDialogDescription terminalSwitchStatus terminalSwitchError"`,
		"hidden",
	} {
		if !strings.Contains(dialogTag, fragment) {
			t.Errorf("terminal-switch dialog opening tag is missing %q; got %q", fragment, dialogTag)
		}
	}
	for _, fragment := range []string{
		`id="terminalSwitchDialogTitle"`,
		`id="terminalSwitchDialogDescription"`,
		`id="terminalSwitchStatus" role="status" aria-live="polite"`,
		`id="terminalSwitchError" role="alert" aria-live="assertive"`,
	} {
		if !strings.Contains(masterHTML, fragment) {
			t.Errorf("terminal-switch dialog is missing accessible feedback contract %q", fragment)
		}
	}
	if errorTag := openingTag(masterHTML, "terminalSwitchError"); !strings.Contains(errorTag, "hidden") {
		t.Errorf("terminal-switch error must be hidden until populated; got %q", errorTag)
	}

	for id, decision := range map[string]string{
		"btnPreserveTerminalSwitch": "preserve",
		"btnDiscardTerminalSwitch":  "discard",
		"btnCancelTerminalSwitch":   "cancel",
	} {
		tag := openingTag(masterHTML, id)
		for _, fragment := range []string{
			"<button",
			`type="button"`,
			`data-switch-decision="` + decision + `"`,
		} {
			if !strings.Contains(tag, fragment) {
				t.Errorf("terminal-switch %s control is missing %q; got %q", decision, fragment, tag)
			}
		}
	}
	discardTag := openingTag(masterHTML, "btnDiscardTerminalSwitch")
	for _, className := range []string{"btn-danger", "terminal-switch-discard"} {
		if !strings.Contains(discardTag, className) {
			t.Errorf("discard must carry destructive emphasis class %q; got %q", className, discardTag)
		}
	}

	masterCSS := read("frontend/src/master.css")
	compactCSS := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "").Replace(masterCSS)
	for _, fragment := range []string{
		".terminal-switch-dialog{",
		".terminal-switch-dialog[hidden]{display:none;}",
		".terminal-switch-discard{",
	} {
		if !strings.Contains(compactCSS, fragment) {
			t.Errorf("master stylesheet is missing terminal-switch visibility/destructive contract %q", fragment)
		}
	}

	playerSurface := strings.Join([]string{
		read("client/index.html"),
		read("client/client.css"),
		read("client/client.js"),
		read("internal/player/handler.go"),
	}, "\n")
	for _, forbidden := range []string{
		"terminalSwitchDialog",
		"btnPreserveTerminalSwitch",
		"btnDiscardTerminalSwitch",
		"btnCancelTerminalSwitch",
		"data-switch-decision",
		"resolveTerminalSwitch",
		"ResolveTerminalSwitch",
		"switchId",
		"ForceHackSuccess",
		"forceHackSuccess",
	} {
		if strings.Contains(playerSurface, forbidden) {
			t.Errorf("player surface exposes game-master switch/private-puzzle capability %q", forbidden)
		}
	}
}

func TestGameMasterEndBroadcastDialogIsAccessibleAndAvoidsNativeConfirm(t *testing.T) {
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
	openingTag := func(document, id string) string {
		t.Helper()
		marker := `id="` + id + `"`
		markerAt := strings.Index(document, marker)
		if markerAt < 0 {
			return ""
		}
		start := strings.LastIndex(document[:markerAt], "<")
		endOffset := strings.Index(document[markerAt:], ">")
		if start < 0 || endOffset < 0 {
			return ""
		}
		return document[start : markerAt+endOffset+1]
	}

	masterHTML := read("frontend/src/index.html")
	masterJS := read("frontend/src/master.js")
	masterCSS := read("frontend/src/master.css")
	dialogTag := openingTag(masterHTML, "endBroadcastDialog")
	for _, fragment := range []string{
		"<dialog",
		`id="endBroadcastDialog"`,
		`aria-modal="true"`,
		`aria-labelledby="endBroadcastDialogTitle"`,
		`aria-describedby="endBroadcastDialogDescription"`,
		"hidden",
	} {
		if !strings.Contains(dialogTag, fragment) {
			t.Errorf("end-broadcast dialog opening tag is missing %q; got %q", fragment, dialogTag)
		}
	}
	for _, id := range []string{"endBroadcastDialogTitle", "endBroadcastDialogDescription", "btnCancelEndBroadcast", "btnConfirmEndBroadcast"} {
		if !strings.Contains(masterHTML, `id="`+id+`"`) {
			t.Errorf("end-broadcast dialog is missing %q", id)
		}
	}
	for _, id := range []string{"btnCancelEndBroadcast", "btnConfirmEndBroadcast"} {
		if tag := openingTag(masterHTML, id); !strings.Contains(tag, `type="button"`) {
			t.Errorf("end-broadcast control %q must be an explicit button; got %q", id, tag)
		}
	}
	for _, fragment := range []string{
		"showEndBroadcastConfirmation()",
		"desktopAPI.endBroadcast()",
		"!result.state || result.state.broadcast",
		"btnConfirmEndBroadcast.disabled = true",
	} {
		if !strings.Contains(masterJS, fragment) {
			t.Errorf("master script is missing end-broadcast contract %q", fragment)
		}
	}
	if strings.Contains(masterJS, "window.confirm('Завершить текущую трансляцию") {
		t.Error("end-broadcast action still depends on the native window.confirm gate")
	}
	compactCSS := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "").Replace(masterCSS)
	if !strings.Contains(compactCSS, ".end-broadcast-actions{") {
		t.Error("master stylesheet is missing end-broadcast confirmation layout")
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
		"//go:embed all:client/dist\nvar playerSource embed.FS",
		`fs.Sub(frontendSource, "frontend/dist")`,
		`fs.Sub(playerSource, "client/dist")`,
		"Assets: frontendAssets",
		"composeApplication(playerAssets)",
		"Assets:  playerAssets",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(source, fragment) {
			t.Errorf("main.go is missing production asset wiring %q", fragment)
		}
	}
	if strings.Contains(source, "//go:embed all:frontend/dist all:client/dist") ||
		strings.Contains(source, "//go:embed all:client/dist all:frontend/dist") {
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

func TestPackagedPlayerBuildIsCompleteAndOffline(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	dist := filepath.Join(root, "client", "dist")
	entries, err := os.ReadDir(dist)
	if err != nil {
		t.Fatalf("client/dist/.keep must preserve the go:embed root on a clean checkout: %v", err)
	}
	if len(entries) == 1 && entries[0].Name() == ".keep" {
		return
	}
	assertNonEmptyFiles(t, dist, []string{"index.html"})

	var bundleFiles []string
	err = filepath.WalkDir(dist, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(dist, path)
		if err != nil {
			return err
		}
		bundleFiles = append(bundleFiles, filepath.ToSlash(relative))
		if extension := strings.ToLower(filepath.Ext(path)); extension == ".html" || extension == ".js" || extension == ".css" {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, forbidden := range []string{"https://cdn.", "http://localhost:", "http://127.0.0.1:5173", "@vite/client"} {
				if strings.Contains(string(raw), forbidden) {
					t.Errorf("packaged player asset %s depends on %q", relative, forbidden)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for extension, description := range map[string]string{".js": "generated player bundle", ".css": "player stylesheet", ".ttf": "Fixedsys font"} {
		if !containsExtension(bundleFiles, extension) {
			t.Errorf("packaged player is missing %s", description)
		}
	}
	for _, category := range []string{"ambient", "charscroll", "enter", "hack-bad", "hack-good", "menu-focus", "multiple", "single"} {
		entries, err := os.ReadDir(filepath.Join(dist, "sounds", category))
		if err != nil || len(entries) == 0 {
			t.Errorf("packaged player sound category %s is missing: %v", category, err)
		}
	}

	mainSource, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mainSource), "//go:embed all:client/dist") || !strings.Contains(string(mainSource), `fs.Sub(playerSource, "client/dist")`) {
		t.Error("production does not embed only the complete built player application")
	}
}

func TestPlayerSessionsControlCrossCuttingAssetContract(t *testing.T) {
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

	masterHTML := read("frontend/src/index.html")
	masterJS := read("frontend/src/master.js")
	masterCSS := read("frontend/src/master.css")
	playerHTML := read("client/index.html")
	playerJS := read("client/client.js")
	playerCSS := read("client/client.css")
	playerHTTP := read("internal/player/http.go")
	playerProtocol := read("internal/player/handler.go")

	for _, directive := range []string{
		`default-src 'self'`,
		`script-src 'self'`,
		`object-src 'none'`,
		`base-uri 'none'`,
		`form-action 'none'`,
	} {
		if !strings.Contains(masterHTML, directive) {
			t.Errorf("master CSP is missing restrictive directive %q", directive)
		}
	}
	for _, fragment := range []string{
		`response.Header().Set("Content-Security-Policy", playerContentSecurityPolicy)`,
		`default-src 'self'`,
		`script-src 'self'`,
		`object-src 'none'`,
		`base-uri 'none'`,
		`frame-ancestors 'none'`,
	} {
		if !strings.Contains(playerHTTP, fragment) {
			t.Errorf("player HTTP boundary is missing CSP contract %q", fragment)
		}
	}

	for _, fragment := range []string{
		`row.querySelector('.roster-name').textContent = character.name || '—'`,
		`row.querySelector('.session-primary-name').textContent = assigned`,
		`row.querySelector('.session-character-name').textContent = assigned`,
		`row.querySelector('.session-fallback-label').textContent = `,
	} {
		if !strings.Contains(masterJS, fragment) {
			t.Errorf("master asset is missing text-only name rendering %q", fragment)
		}
	}
	for _, fragment := range []string{
		`option.textContent = entry.name`,
		`playerCharacterName.textContent = playerState.character.name`,
		`playerFallbackName.textContent = playerState.fallbackName`,
	} {
		if !strings.Contains(playerJS, fragment) {
			t.Errorf("player asset is missing text-only name rendering %q", fragment)
		}
	}
	for _, forbidden := range []string{
		"innerHTML = character.name",
		"innerHTML = session.fallbackName",
		"innerHTML = entry.name",
		"innerHTML = playerState.fallbackName",
	} {
		if strings.Contains(masterJS, forbidden) || strings.Contains(playerJS, forbidden) {
			t.Errorf("coordination assets interpolate an unescaped name through %q", forbidden)
		}
	}

	masterSurface := masterHTML + "\n" + masterJS + "\n" + masterCSS
	for _, forbidden := range []string{"browserToken", "PLAYER_TOKEN_KEY", "fallout-terminal.player-token"} {
		if strings.Contains(masterSurface, forbidden) {
			t.Errorf("master assets expose player resume-token detail %q", forbidden)
		}
	}
	for _, fragment := range []string{
		"const PLAYER_TOKEN_KEY = 'fallout-terminal.player-token'",
		"localStorage.getItem(PLAYER_TOKEN_KEY)",
		"localStorage.setItem(PLAYER_TOKEN_KEY",
		"playerRPC.subscribe(request",
	} {
		if !strings.Contains(playerJS, fragment) {
			t.Errorf("player token is not confined to the private handshake/storage path %q", fragment)
		}
	}
	for _, forbidden := range []string{"browserToken", "PLAYER_TOKEN_KEY", "?token", "?session"} {
		if strings.Contains(playerHTML+"\n"+playerCSS, forbidden) {
			t.Errorf("player document/style surface leaks token detail %q", forbidden)
		}
	}
	for _, forbidden := range []string{"URLSearchParams", "location.search", "searchParams.set('browserToken'", `searchParams.set("browserToken"`} {
		if strings.Contains(playerJS, forbidden) {
			t.Errorf("player script exposes resume tokens through the URL via %q", forbidden)
		}
	}

	for _, fragment := range []string{
		"const observerReadOnly = hasState && playerState.role === 'observer'",
		"screen.classList.toggle('observer-read-only', observerReadOnly)",
		"screen.setAttribute('aria-readonly', String(observerReadOnly))",
		"function canControlSharedTerminal()",
		"playerState.role === 'active'",
		"pendingSharedAction === null",
	} {
		if !strings.Contains(playerJS, fragment) {
			t.Errorf("player asset is missing observer/local-only action gate %q", fragment)
		}
	}
	for _, fragment := range []string{
		"#screen.observer-read-only :is(.term-row, .back-btn, .hcell)",
		"#screen.shared-input-pending :is(.term-row, .back-btn, .hcell)",
	} {
		if !strings.Contains(playerCSS, fragment) {
			t.Errorf("player stylesheet is missing read-only/pending presentation %q", fragment)
		}
	}

	for _, fragment := range []string{
		"pendingSharedAction.acceptedRevision = Number(result.revision) || 0",
		"if (appliedSharedRevision < pendingSharedAction.acceptedRevision) return",
		"playerState.revision >= pendingSelection.acceptedRevision",
	} {
		if !strings.Contains(playerJS, fragment) {
			t.Errorf("player asset clears pending input without authoritative revision evidence %q", fragment)
		}
	}
	for _, fragment := range []string{
		"pendingTerminalSwitch = result?.switchId || null",
		"desktopAPI.resolveTerminalSwitch({ switchId: pendingTerminalSwitch, decision })",
		"if (!pendingTerminalSwitch || coordinationCommandPending) return",
	} {
		if !strings.Contains(masterJS, fragment) {
			t.Errorf("master asset is missing terminal-switch resolution contract %q", fragment)
		}
	}
	for _, fragment := range []string{
		`id="playerConfigStatus"`,
		`id="btnOpenPlayerConfig"`,
		`id="btnNewPlayerConfig"`,
		`id="playerConfigError"`,
	} {
		if !strings.Contains(masterHTML, fragment) {
			t.Errorf("master asset is missing player-config recovery control %q", fragment)
		}
	}
	for _, fragment := range []string{
		`.coord-panel[data-player-config-active="false"] .roster-management`,
		`.player-config-error[hidden]`,
	} {
		if !strings.Contains(masterCSS, fragment) {
			t.Errorf("master stylesheet is missing player-config gating contract %q", fragment)
		}
	}

	compactPlayerCSS := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "").Replace(playerCSS)
	compactMasterCSS := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "").Replace(masterCSS)
	for _, fragment := range []string{
		"--terminal-scale:clamp(",
		".screen{position:relative;width:100%;max-width:1500px;height:100%;",
		"@media(max-width:760px){",
		"@media(max-height:720px){",
	} {
		if !strings.Contains(compactPlayerCSS, fragment) {
			t.Errorf("player stylesheet is missing responsive layout boundary %q", fragment)
		}
	}
	for _, fragment := range []string{
		"@media(max-width:1050px){",
		"@media(max-width:820px){",
		"@media(max-width:620px),(max-height:560px){",
		".terminal-switch-dialog{width:calc(100vw-20px);max-height:calc(100dvh-20px);",
	} {
		if !strings.Contains(compactMasterCSS, fragment) {
			t.Errorf("master stylesheet is missing responsive layout boundary %q", fragment)
		}
	}

	playerSurface := strings.Join([]string{playerHTML, playerCSS, playerJS, playerProtocol}, "\n")
	for _, forbidden := range []string{
		"ForceHackSuccess",
		"forceHackSuccess",
		"HACK_ADMIN",
		"btnHackSuccess",
		"resolveTerminalSwitch",
		"ResolveTerminalSwitch",
	} {
		if strings.Contains(playerSurface, forbidden) {
			t.Errorf("player surface exposes private game-master capability %q", forbidden)
		}
	}
}

func TestPlayerBundleImportsOnlyPublicGeneratedContractsAndNoGenericPrivateCarriers(t *testing.T) {
	t.Parallel()
	root := assetRepositoryRoot(t)
	for _, relative := range []string{"client/client.js", "client/sound.js", "client/index.html"} {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		source := strings.ToLower(string(raw))
		for _, forbidden := range []string{
			"fallout/terminal/private", "fallout/terminal/persistence", "protojson", "base64",
			"genericdispatch", "generic-dispatch", "forcehacksuccess", "resetfailedhack",
		} {
			if strings.Contains(source, forbidden) {
				t.Errorf("%s imports or carries private desktop semantic %q", relative, forbidden)
			}
		}
	}
}

func TestOneProtocolCutoverHasNoActiveLegacyPlayerSurface(t *testing.T) {
	t.Parallel()

	root := assetRepositoryRoot(t)
	paths := []string{
		"client/client.js", "client/sound.js", "client/index.html",
		"internal/player", "internal/testutil/testdata", "tests/browser/fixture-server",
		"go.mod", "README.md", "docs",
	}
	legacyIdentifiers := []string{
		"SESSION_HELLO", "CHARACTER_SELECT", "NAV_ACTION", "HACK_GUESS", "HACK_PATTERN",
		"SESSION_WELCOME", "PLAYER_STATE", "ACTION_RESULT", "TERMINAL_LIVE", "TERMINAL_UPDATE",
		"TERMINAL_CLEAR", "NAV_STATE", "HACK_STATE", "HACK_ADMIN",
	}
	for _, relative := range paths {
		path := filepath.Join(root, filepath.FromSlash(relative))
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		var files []string
		if info.IsDir() {
			err = filepath.WalkDir(path, func(candidate string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if !entry.IsDir() && !strings.Contains(filepath.ToSlash(candidate), "/gen/") && !strings.Contains(filepath.ToSlash(candidate), "/dist/") && !strings.Contains(filepath.ToSlash(candidate), "/node_modules/") {
					files = append(files, candidate)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		} else {
			files = []string{path}
		}
		for _, candidate := range files {
			raw, err := os.ReadFile(candidate)
			if err != nil {
				t.Fatal(err)
			}
			content := string(raw)
			lower := strings.ToLower(content)
			for _, forbidden := range []string{"github.com/coder/websocket", "new websocket", "fakewebsocket", "/api/sounds/", "connect-src 'self' ws:", "connect-src 'self' wss:"} {
				if strings.Contains(lower, forbidden) {
					t.Errorf("active cutover surface %s contains %q", candidate, forbidden)
				}
			}
			for _, forbidden := range legacyIdentifiers {
				if strings.Contains(content, forbidden) {
					t.Errorf("active cutover surface %s contains legacy identifier %q", candidate, forbidden)
				}
			}
			if strings.Contains(content, "dispatch(msg") || strings.Contains(content, "send({ type:") {
				t.Errorf("active player surface %s retains a generic message dispatcher", candidate)
			}
		}
	}

	if _, err := os.Stat(filepath.Join(root, "internal", "player", "protocol.go")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("legacy handwritten protocol implementation still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "testutil", "testdata", "protocol")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("legacy JSON protocol fixture directory still exists: %v", err)
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
