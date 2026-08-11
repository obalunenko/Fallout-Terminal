// Package live owns the process-local, server-authoritative live terminal.
package live

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/hack"
	"github.com/obalunenko/Fallout-Terminal/internal/nav"
)

// Service serializes every canonical live-state transition. Values returned
// across this boundary are detached projections and may be freely mutated by
// callers without changing the canonical aggregate.
type Service struct {
	mu            sync.RWMutex
	live          *domain.LiveState
	random        hack.Random
	words         hack.WordSource
	generationIDs generationIDSource
}

type generationIDSource interface {
	Next() string
}

type cryptoGenerationIDSource struct {
	fallback atomic.Uint64
}

func (source *cryptoGenerationIDSource) Next() string {
	var value [16]byte
	if _, err := cryptorand.Read(value[:]); err == nil {
		return hex.EncodeToString(value[:])
	}
	return fmt.Sprintf("%x-%x", time.Now().UnixNano(), source.fallback.Add(1))
}

// New returns an empty live-state service. Randomness and the word source are
// injectable to keep puzzle generation deterministic in tests.
func New(random hack.Random, words hack.WordSource) *Service {
	return &Service{random: random, words: words, generationIDs: &cryptoGenerationIDSource{}}
}

// Set installs a fresh live terminal, resets navigation, and creates a new
// puzzle when hackLevel is greater than zero.
func (service *Service) Set(terminalID, terminalName string, tree domain.ContentNode, hackLevel int, introText string) *domain.PublicLiveState {
	service.mu.Lock()
	defer service.mu.Unlock()

	state := &domain.LiveState{
		TerminalID:   terminalID,
		TerminalName: terminalName,
		Tree:         cloneNode(tree),
		HackLevel:    hackLevel,
		IntroText:    introText,
		Nav:          nav.Default(),
	}
	if hackLevel > 0 {
		state.Hack = hack.GenerateBoard(service.generationIDs.Next(), hackLevel, service.random, service.words)
	}
	service.live = state
	return publicLiveState(state)
}

// Update replaces published content while retaining the current puzzle and
// repairing shared navigation against the new tree. A nil introText preserves
// the current introduction.
func (service *Service) Update(tree domain.ContentNode, introText *string) (*domain.PublicLiveState, bool) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.live == nil {
		return nil, false
	}

	service.live.Tree = cloneNode(tree)
	if introText != nil {
		service.live.IntroText = *introText
	}
	service.live.Nav = nav.Revalidate(service.live.Nav, service.live.Tree)
	return publicLiveState(service.live), true
}

// Clear removes the canonical live terminal. It is safe to call repeatedly.
func (service *Service) Clear() {
	service.mu.Lock()
	defer service.mu.Unlock()
	service.live = nil
}

// Snapshot returns the current immutable player projection, or nil when no
// terminal is live.
func (service *Service) Snapshot() *domain.PublicLiveState {
	service.mu.RLock()
	defer service.mu.RUnlock()
	return publicLiveState(service.live)
}

// ApplyNav applies a player navigation request. The boolean reports whether a
// live terminal existed; valid no-op requests remain observable for protocol
// compatibility.
func (service *Service) ApplyNav(action, nodeID string) (*domain.NavState, bool) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.live == nil {
		return nil, false
	}

	service.live.Nav = nav.ApplyAction(service.live.Nav, service.live.Tree, action, nodeID)
	projection := cloneNav(service.live.Nav)
	return &projection, true
}

// ApplyHackGuess applies a candidate or filler guess to an active puzzle.
func (service *Service) ApplyHackGuess(targetID string) (*domain.PublicHackState, bool) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if !activePuzzle(service.live) {
		return nil, false
	}

	hack.ApplyGuess(service.live.Hack, targetID)
	return hack.PublicState(service.live.Hack), true
}

// ApplyHackPattern atomically validates and consumes one current pattern.
func (service *Service) ApplyHackPattern(patternID string, publish func(*domain.PublicHackState)) bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	if !activePuzzle(service.live) {
		return false
	}

	if !hack.ApplyPattern(service.live.Hack, patternID, service.random) {
		return false
	}
	projection := hack.PublicState(service.live.Hack)
	if publish != nil {
		publish(projection)
	}
	return true
}

// ForceHackSuccess completes an active puzzle without spending an attempt.
func (service *Service) ForceHackSuccess() (*domain.PublicHackState, bool) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if !activePuzzle(service.live) {
		return nil, false
	}

	hack.ForceSuccess(service.live.Hack)
	return hack.PublicState(service.live.Hack), true
}

func activePuzzle(state *domain.LiveState) bool {
	return state != nil && state.Hack != nil && !state.Hack.Solved && !state.Hack.Failed
}

func publicLiveState(state *domain.LiveState) *domain.PublicLiveState {
	if state == nil {
		return nil
	}
	return &domain.PublicLiveState{
		TerminalID:   state.TerminalID,
		TerminalName: state.TerminalName,
		Tree:         cloneNode(state.Tree),
		HackLevel:    state.HackLevel,
		IntroText:    state.IntroText,
		Nav:          cloneNav(state.Nav),
		Hack:         hack.PublicState(state.Hack),
	}
}

func cloneNav(state domain.NavState) domain.NavState {
	clone := state
	clone.Path = append([]string(nil), state.Path...)
	if state.ViewEntryID != nil {
		value := *state.ViewEntryID
		clone.ViewEntryID = &value
	}
	if state.CommandNodeID != nil {
		value := *state.CommandNodeID
		clone.CommandNodeID = &value
	}
	return clone
}

func cloneNode(node domain.ContentNode) domain.ContentNode {
	clone := node
	clone.Children = make([]domain.ContentNode, len(node.Children))
	for index := range node.Children {
		clone.Children[index] = cloneNode(node.Children[index])
	}
	clone.Extra = cloneRawMap(node.Extra)
	return clone
}

func cloneRawMap(values map[string]json.RawMessage) map[string]json.RawMessage {
	if values == nil {
		return nil
	}
	clone := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		clone[key] = append(json.RawMessage(nil), value...)
	}
	return clone
}
