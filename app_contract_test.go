package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	privatev1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/private/v1"
	sessionservice "github.com/obalunenko/Fallout-Terminal/internal/session"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/prototest"
)

func TestPrivateDescriptorFieldsAndEnumsHaveExplicitAdapterCoverage(t *testing.T) {
	coverage := []struct {
		message proto.Message
		fields  []string
	}{
		{&privatev1.CommandResult{}, []string{"ok", "error"}},
		{&privatev1.SessionOperationResult{}, []string{"ok", "canceled", "error", "file_path", "session"}},
		{&privatev1.SaveSessionResult{}, []string{"ok", "error", "requested_revision", "saved_revision"}},
		{&privatev1.PlayerConfigOperationResult{}, []string{"ok", "canceled", "error", "player_config", "session", "state", "player_config_metadata"}},
		{&privatev1.CoordinationResult{}, []string{"ok", "error", "state"}},
		{&privatev1.TerminalSwitchResult{}, []string{"ok", "error", "status", "switch_id", "state"}},
		{&privatev1.TerminalActivationRequest{}, []string{"terminal_id", "terminal_name", "tree", "hack_level", "intro_text"}},
		{&privatev1.LiveTerminalUpdateRequest{}, []string{"tree", "intro_text"}},
		{&privatev1.TerminalSwitchDecisionRequest{}, []string{"switch_id", "choice"}},
		{&privatev1.ResetFailedHackRequest{}, []string{"terminal_id", "terminal_name", "tree", "hack_level", "intro_text"}},
		{&privatev1.AddCharacterRequest{}, []string{"display_name"}},
		{&privatev1.RenameCharacterRequest{}, []string{"character_id", "display_name"}},
		{&privatev1.DeleteCharacterRequest{}, []string{"character_id"}},
		{&privatev1.RenameLogicalSessionRequest{}, []string{"logical_session_id", "fallback_name"}},
		{&privatev1.AssignCharacterRequest{}, []string{"logical_session_id", "character_id"}},
		{&privatev1.ReleaseCharacterRequest{}, []string{"logical_session_id"}},
		{&privatev1.MoveCharacterRequest{}, []string{"character_id", "destination_session_id"}},
		{&privatev1.SetActiveControllerRequest{}, []string{"logical_session_id"}},
		{&privatev1.OpenUrlRequest{}, []string{"url"}},
		{&privatev1.ServerInformation{}, []string{"local_url", "public_url", "tunnel_enabled", "ip", "port", "tunnel_error", "url"}},
		{&privatev1.RuntimeStatus{}, []string{"server_info", "client_count", "hack_state", "startup_error", "save_state", "requested_revision", "saved_revision", "coordination_state"}},
		{&privatev1.ServerInformationEvent{}, []string{"server_info"}},
		{&privatev1.ClientCountEvent{}, []string{"client_count"}},
		{&privatev1.HackStateEvent{}, []string{"hack_state"}},
		{&privatev1.CoordinationStateEvent{}, []string{"coordination_state"}},
		{&privatev1.CharacterState{}, []string{"character_id", "display_name", "logical_session_id"}},
		{&privatev1.LogicalSessionState{}, []string{"logical_session_id", "fallback_name", "connected", "active_streams", "character_id", "role"}},
		{&privatev1.BroadcastState{}, []string{"broadcast_id", "active_controller_session_id", "active_terminal_id", "revision"}},
		{&privatev1.PendingTerminalSwitch{}, []string{"switch_id", "terminal_id", "terminal_name", "requested_terminal", "broadcast_id", "source_terminal_id", "target_terminal_id"}},
		{&privatev1.PlayerConfigMetadata{}, []string{"status", "file_path", "version", "name"}},
		{&privatev1.CoordinationState{}, []string{"roster", "logical_sessions", "broadcast", "pending_terminal_switch", "revision", "player_config"}},
	}
	for _, test := range coverage {
		descriptor := test.message.ProtoReflect().Descriptor()
		prototest.Message{}.Test(t, test.message.ProtoReflect().Type())
		actual := make([]string, 0, descriptor.Fields().Len())
		for index := 0; index < descriptor.Fields().Len(); index++ {
			actual = append(actual, string(descriptor.Fields().Get(index).Name()))
		}
		require.ElementsMatch(t, test.fields, actual, "adapter coverage drifted for %s", descriptor.FullName())
	}
	require.Equal(t, privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_UNSPECIFIED, terminalSwitchStatusToPrivate("unknown"))
	require.Empty(t, terminalSwitchStatusFromPrivate(privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_UNSPECIFIED))
	require.Empty(t, terminalSwitchChoiceFromPrivate(privatev1.TerminalSwitchChoice_TERMINAL_SWITCH_CHOICE_UNSPECIFIED))
}

func TestRuntimeStatusDescriptorRemainsFeature005Compatible(t *testing.T) {
	t.Parallel()

	descriptor := (&privatev1.RuntimeStatus{}).ProtoReflect().Descriptor()
	require.Equal(t, "fallout.terminal.private.v1.RuntimeStatus", string(descriptor.FullName()))

	wantFields := []string{
		"server_info", "client_count", "hack_state", "startup_error",
		"save_state", "requested_revision", "saved_revision", "coordination_state",
	}
	gotFields := make([]string, 0, descriptor.Fields().Len())
	for index := range descriptor.Fields().Len() {
		field := descriptor.Fields().Get(index)
		gotFields = append(gotFields, string(field.Name()))
		require.Equal(t, protoreflect.FieldNumber(index+1), field.Number())
	}
	require.Equal(t, wantFields, gotFields)
	require.Nil(t, descriptor.Fields().ByName("phase"))
	require.Nil(t, descriptor.Fields().ByJSONName("phase"))
	require.Zero(t, descriptor.ParentFile().Enums().Len())
}

func TestPrivateStatusResultAndEventAdaptersRoundTripEveryNativeSemantic(t *testing.T) {
	controller := domain.LogicalSessionID("session-1")
	terminal := "terminal-1"
	target := "terminal-2"
	state := &domain.MasterCoordinationState{
		Revision:      17,
		PlayerConfig:  &domain.PlayerConfigMetadata{Status: "loaded", FilePath: "/private/players.json", Version: 1, Name: "Vault 33"},
		Roster:        []domain.MasterRosterEntry{{ID: "character-1", Name: "Lucy", ClaimedBySessionID: &controller}},
		Sessions:      []domain.MasterSessionEntry{{ID: controller, FallbackName: "PLAYER 1", Connected: true, Character: &domain.PlayerCharacter{ID: "character-1", Name: "Lucy"}, Role: domain.PlayerRoleActive}},
		Broadcast:     &domain.MasterBroadcastState{ID: "broadcast-1", ControllerSessionID: &controller, ActiveTerminalID: &terminal},
		PendingSwitch: &domain.MasterPendingSwitch{SwitchID: "switch-1", BroadcastID: "broadcast-1", SourceTerminalID: terminal, TargetTerminalID: &target},
	}
	status := RuntimeStatus{
		ServerInfo:  &domain.ServerInfo{URL: "https://fallout.example", LocalURL: "http://127.0.0.1:3690", Tunnel: true},
		ClientCount: 2, StartupError: "startup", SaveState: "saved", RequestedRevision: 19, SavedRevision: 18,
		CoordinationState: state,
	}
	routed := routeRuntimeStatus(status)
	require.Equal(t, status, routed)
	require.Equal(t, state, routeCoordinationEvent(state))
	require.Equal(t, domain.ServerInfo{URL: "https://fallout.example", LocalURL: "http://127.0.0.1:3690", Tunnel: true}, routeServerInfoEvent(*status.ServerInfo))
	require.Equal(t, 2, routeClientCountEvent(2))
	require.Equal(t, CommandResult{OK: true}, routeCommandResult(CommandResult{OK: true}))
	require.Equal(t, CoordinationCommandResult{OK: true, State: state}, routeCoordinationResult(CoordinationCommandResult{OK: true, State: state}))
}

func TestDesktopServiceInventoryAndNativeEventsAreExactlyAllowlisted(t *testing.T) {
	requiredMethods := []string{
		"GetRuntimeStatus", "NewSession", "OpenSession", "CopyDemo", "SaveSession", "LoadReferencedPlayerConfig", "NewPlayerConfig", "OpenPlayerConfig",
		"RequestTerminalActivation", "UpdateLiveTerminal", "RequestTerminalClear", "ResolveTerminalSwitch", "ForceHackSuccess", "ResetFailedHack",
		"AddCharacter", "RenameCharacter", "DeleteCharacter", "RenameLogicalSession", "AssignCharacter", "ReleaseCharacter", "MoveCharacter", "SetActiveController",
		"StartBroadcast", "EndBroadcast", "OpenURL",
	}
	serviceType := reflect.TypeOf((*desktopService)(nil))
	actualMethods := make([]string, 0, serviceType.NumMethod())
	for index := range serviceType.NumMethod() {
		actualMethods = append(actualMethods, serviceType.Method(index).Name)
	}
	require.Len(t, actualMethods, 25)
	require.ElementsMatch(t, requiredMethods, actualMethods)

	for _, forbidden := range []string{
		"Start", "Shutdown", "ServiceStartup", "ServiceShutdown",
		"Dispatch", "Call", "Capabilities", "Reflect",
		"ReadFile", "WriteFile", "Exec", "Environment", "OpenDialog", "Browser",
		"PlayerService", "Subscribe", "SelectCharacter", "Navigate", "Guess", "ActivatePattern", "SoundManifest",
	} {
		require.NotContains(t, actualMethods, forbidden)
	}

	require.Equal(t, []string{"server-info", "client-count", "hack-state", "coordination-state"}, []string{serverInfoEvent, clientCountEvent, hackStateEvent, coordinationStateEvent})
}

func TestDesktopServiceMethodsAreTransparentCoreForwards(t *testing.T) {
	t.Parallel()

	file, err := parser.ParseFile(token.NewFileSet(), "desktop_service.go", nil, 0)
	require.NoError(t, err)

	forwarded := make(map[string]string)
	for _, declaration := range file.Decls {
		method, ok := declaration.(*ast.FuncDecl)
		if !ok || method.Recv == nil || method.Name.Name == "newDesktopService" {
			continue
		}
		require.Len(t, method.Body.List, 1, "%s must remain a transparent forward", method.Name.Name)
		returned, ok := method.Body.List[0].(*ast.ReturnStmt)
		require.True(t, ok, "%s must return the core result directly", method.Name.Name)
		require.Len(t, returned.Results, 1, "%s must return exactly one core call", method.Name.Name)
		call, ok := returned.Results[0].(*ast.CallExpr)
		require.True(t, ok, "%s must return a core call", method.Name.Name)
		selector, ok := call.Fun.(*ast.SelectorExpr)
		require.True(t, ok, "%s must call an explicit core method", method.Name.Name)
		core, ok := selector.X.(*ast.SelectorExpr)
		require.True(t, ok, "%s must call through service.core", method.Name.Name)
		service, ok := core.X.(*ast.Ident)
		require.True(t, ok)
		require.Equal(t, "service", service.Name)
		require.Equal(t, "core", core.Sel.Name)
		forwarded[method.Name.Name] = selector.Sel.Name
	}

	require.Len(t, forwarded, 25)
	for exposed, core := range forwarded {
		require.Equal(t, exposed, core, "%s must not translate into an authored capability", exposed)
	}
}

func TestDetachedDesktopResultShapesPreserveCancellationErrorsAndStatusFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		keys  []string
	}{
		{"runtime status", RuntimeStatus{}, []string{"serverInfo", "clientCount", "hackState", "saveState", "requestedRevision", "savedRevision", "coordinationState"}},
		{"command", CommandResult{Error: "safe"}, []string{"ok", "error"}},
		{"session cancellation", sessionservice.SessionResult{Canceled: true}, []string{"ok", "canceled"}},
		{"save", sessionservice.SaveResult{Error: "safe"}, []string{"ok", "error", "requestedRevision"}},
		{"player config cancellation", PlayerConfigCommandResult{Canceled: true}, []string{"ok", "canceled", "state"}},
		{"coordination", CoordinationCommandResult{Error: "safe"}, []string{"ok", "error", "state"}},
		{"terminal switch", TerminalSwitchCommandResult{Error: "safe"}, []string{"ok", "error", "state"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw, err := json.Marshal(test.value)
			require.NoError(t, err)
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(raw, &fields))
			actual := make([]string, 0, len(fields))
			for key := range fields {
				actual = append(actual, key)
			}
			require.ElementsMatch(t, test.keys, actual)
			require.NotContains(t, fields, "phase")
		})
	}
}
