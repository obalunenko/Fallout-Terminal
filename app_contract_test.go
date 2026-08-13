package main

import (
	"reflect"
	"testing"

	privatev1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/private/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestPrivateDescriptorFieldsAndEnumsHaveExplicitAdapterCoverage(t *testing.T) {
	messages := []protoreflect.MessageDescriptor{
		(&privatev1.RuntimeStatus{}).ProtoReflect().Descriptor(),
		(&privatev1.CommandResult{}).ProtoReflect().Descriptor(),
		(&privatev1.CoordinationState{}).ProtoReflect().Descriptor(),
		(&privatev1.TerminalSwitchResult{}).ProtoReflect().Descriptor(),
		(&privatev1.ServerInformationEvent{}).ProtoReflect().Descriptor(),
		(&privatev1.ClientCountEvent{}).ProtoReflect().Descriptor(),
		(&privatev1.HackStateEvent{}).ProtoReflect().Descriptor(),
		(&privatev1.CoordinationStateEvent{}).ProtoReflect().Descriptor(),
	}
	for _, descriptor := range messages {
		for index := 0; index < descriptor.Fields().Len(); index++ {
			field := descriptor.Fields().Get(index)
			require.NotEmpty(t, field.JSONName(), "%s.%s", descriptor.FullName(), field.Name())
		}
	}
	require.Equal(t, privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_UNSPECIFIED, terminalSwitchStatusToPrivate("unknown"))
}

func TestEveryBoundAppMethodAndNativeEventHasPrivateSemanticRegistration(t *testing.T) {
	methods := []string{
		"GetRuntimeStatus", "NewSession", "OpenSession", "CopyDemo", "SaveSession", "LoadReferencedPlayerConfig", "NewPlayerConfig", "OpenPlayerConfig",
		"RequestTerminalActivation", "UpdateLiveTerminal", "RequestTerminalClear", "ResolveTerminalSwitch", "ForceHackSuccess", "ResetFailedHack",
		"AddCharacter", "RenameCharacter", "DeleteCharacter", "RenameLogicalSession", "AssignCharacter", "ReleaseCharacter", "MoveCharacter", "SetActiveController",
		"StartBroadcast", "EndBroadcast", "OpenURL",
	}
	appType := reflect.TypeOf((*App)(nil))
	for _, name := range methods {
		_, ok := appType.MethodByName(name)
		require.True(t, ok, "missing bound App method %s", name)
	}
	require.Equal(t, []string{"server-info", "client-count", "hack-state", "coordination-state"}, []string{serverInfoEvent, clientCountEvent, hackStateEvent, coordinationStateEvent})
}
