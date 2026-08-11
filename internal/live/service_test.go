package live

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
)

func TestSetSnapshotIsDetachedAndSecretFree(t *testing.T) {
	service := New(&constantRandom{}, fixedWords{})
	tree := testTree()

	first := service.Set("terminal-1", "Overseer", tree, 1, "WELCOME")
	if first == nil || first.Hack == nil {
		t.Fatalf("Set() = %#v, want live terminal with puzzle", first)
	}

	tree.Name = "MUTATED INPUT"
	first.Tree.Name = "MUTATED RESULT"
	first.Nav.Path[0] = "mutated"
	first.Hack.Log = append(first.Hack.Log, "private mutation")

	snapshot := service.Snapshot()
	if snapshot == nil {
		t.Fatal("Snapshot() returned nil")
	}
	if snapshot.Tree.Name != "ROOT" || !reflect.DeepEqual(snapshot.Nav.Path, []string{"root"}) || len(snapshot.Hack.Log) != 0 {
		t.Fatalf("canonical state was mutated through a boundary: %#v", snapshot)
	}

	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, privateField := range []string{"secretWord", "wordsById"} {
		if strings.Contains(string(raw), privateField) {
			t.Errorf("public snapshot leaked %q: %s", privateField, raw)
		}
	}
}

func TestUpdateRevalidatesNavigationAndPreservesPuzzle(t *testing.T) {
	service := New(&constantRandom{}, fixedWords{})
	service.Set("terminal-1", "Overseer", testTree(), 1, "OLD")
	if _, ok := service.ApplyNav("enter", "docs"); !ok {
		t.Fatal("ApplyNav() rejected active live terminal")
	}
	if _, ok := service.ApplyNav("entry", "report"); !ok {
		t.Fatal("ApplyNav() rejected active entry")
	}
	before := service.Snapshot()
	intro := "NEW"

	updated, ok := service.Update(treeWithoutReport(), &intro)
	if !ok {
		t.Fatal("Update() rejected active live terminal")
	}
	if updated.IntroText != intro || updated.Nav.Mode != "list" || updated.Nav.ViewEntryID != nil {
		t.Fatalf("Update() did not revalidate navigation: %#v", updated)
	}
	if !reflect.DeepEqual(updated.Hack, before.Hack) {
		t.Fatal("Update() reset or changed the active puzzle")
	}
}

func TestApplyHackPatternReturnsDetachedAcceptedState(t *testing.T) {
	service := New(&constantRandom{}, fixedWords{})
	initial := service.Set("terminal-1", "Overseer", testTree(), 1, "WELCOME")
	if initial == nil || initial.Hack == nil || len(initial.Hack.Patterns) == 0 {
		t.Fatalf("Set() pattern state = %#v", initial)
	}
	patternID := initial.Hack.Patterns[0].ID

	result, ok := activatePattern(service, patternID)
	if !ok || result == nil {
		t.Fatalf("ApplyHackPattern(%q) = %#v, %t", patternID, result, ok)
	}
	used := false
	for _, pattern := range result.Patterns {
		if pattern.ID == patternID {
			used = pattern.Used
		}
	}
	if !used {
		t.Fatalf("accepted pattern %q was not marked used: %#v", patternID, result.Patterns)
	}
	result.Patterns[0].ID = "mutated"
	snapshot := service.Snapshot()
	if snapshot == nil || snapshot.Hack == nil || snapshot.Hack.Patterns[0].ID == "mutated" {
		t.Fatal("public pattern projection mutated canonical state")
	}
}

func TestClearAndAbsentActions(t *testing.T) {
	service := New(&constantRandom{}, fixedWords{})
	if service.Snapshot() != nil {
		t.Fatal("new service unexpectedly has live state")
	}
	if _, ok := service.Update(testTree(), nil); ok {
		t.Fatal("Update() succeeded without live state")
	}
	if _, ok := service.ApplyNav("back", ""); ok {
		t.Fatal("ApplyNav() succeeded without live state")
	}
	if _, ok := service.ApplyHackGuess("A1"); ok {
		t.Fatal("ApplyHackGuess() succeeded without a puzzle")
	}
	if service.ApplyHackPattern("opaque-stale-pattern", nil) {
		t.Fatal("ApplyHackPattern() succeeded without a puzzle")
	}

	service.Set("terminal-1", "Overseer", testTree(), 0, "")
	service.Clear()
	service.Clear()
	if service.Snapshot() != nil {
		t.Fatal("Clear() left stale live state")
	}
}

func TestConcurrentTransitionsAndSnapshots(t *testing.T) {
	service := New(&constantRandom{}, fixedWords{})
	service.Set("terminal-1", "Overseer", testTree(), 1, "WELCOME")

	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			for iteration := 0; iteration < 100; iteration++ {
				switch worker % 4 {
				case 0:
					service.ApplyNav("enter", "docs")
					service.ApplyNav("back", "")
				case 1:
					service.ApplyHackGuess("missing")
				case 2:
					snapshot := service.Snapshot()
					if snapshot != nil && snapshot.Hack != nil && len(snapshot.Hack.Patterns) > 0 {
						service.ApplyHackPattern(snapshot.Hack.Patterns[0].ID, nil)
					}
				case 3:
					snapshot := service.Snapshot()
					if snapshot != nil {
						snapshot.Tree.Name = "external"
					}
				}
			}
		}(worker)
	}
	workers.Wait()

	snapshot := service.Snapshot()
	if snapshot == nil || snapshot.Tree.Name != "ROOT" || len(snapshot.Nav.Path) == 0 || snapshot.Nav.Path[0] != "root" {
		t.Fatalf("concurrent use corrupted canonical state: %#v", snapshot)
	}
}

func TestConcurrentPatternUseAppliesOnceAndFreshSetResetsUsage(t *testing.T) {
	service := New(&constantRandom{}, fixedWords{})
	initial := service.Set("terminal-1", "Overseer", testTree(), 1, "WELCOME")
	if initial == nil || initial.Hack == nil || len(initial.Hack.Patterns) == 0 {
		t.Fatalf("Set() pattern state = %#v", initial)
	}
	patternID := initial.Hack.Patterns[0].ID

	var accepted atomic.Int32
	var workers sync.WaitGroup
	for range 16 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if service.ApplyHackPattern(patternID, nil) {
				accepted.Add(1)
			}
		}()
	}
	workers.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("accepted concurrent actions = %d, want 1", accepted.Load())
	}

	beforeRejected := service.Snapshot()
	if service.ApplyHackPattern(patternID, nil) {
		t.Fatal("repeated pattern was accepted")
	}
	if afterRejected := service.Snapshot(); !reflect.DeepEqual(afterRejected, beforeRejected) {
		t.Fatalf("repeated pattern changed state\ngot: %#v\nwant: %#v", afterRejected, beforeRejected)
	}

	fresh := service.Set("terminal-1", "Overseer", testTree(), 1, "WELCOME")
	if fresh == nil || fresh.Hack == nil || len(fresh.Hack.Patterns) == 0 {
		t.Fatalf("fresh Set() pattern state = %#v", fresh)
	}
	for _, pattern := range fresh.Hack.Patterns {
		if pattern.Used {
			t.Fatalf("fresh puzzle retained used pattern %#v", pattern)
		}
	}
}

func TestPatternGenerationRejectsStaleIDWithoutRandomnessOrPublication(t *testing.T) {
	random := newCountingRandom(1)
	service := New(random, fixedWords{})
	service.generationIDs = &sequenceGenerationIDs{values: []string{"generation-old", "generation-new"}}
	old := service.Set("terminal-1", "Overseer", testTree(), 1, "WELCOME")
	if old == nil || old.Hack == nil || len(old.Hack.Patterns) == 0 {
		t.Fatalf("old puzzle = %#v", old)
	}
	staleID := old.Hack.Patterns[0].ID
	current := service.Set("terminal-1", "Overseer", testTree(), 1, "WELCOME")
	if current == nil || current.Hack == nil || len(current.Hack.Patterns) == 0 || current.Hack.Patterns[0].ID == staleID {
		t.Fatalf("fresh generation did not replace opaque identities: old=%q current=%#v", staleID, current)
	}

	beforeCalls := random.calls.Load()
	publications := atomic.Int32{}
	if service.ApplyHackPattern(staleID, func(*domain.PublicHackState) { publications.Add(1) }) {
		t.Fatal("stale generation pattern was accepted")
	}
	if random.calls.Load() != beforeCalls || publications.Load() != 0 {
		t.Fatalf("stale request consumed RNG or published: calls=%d->%d publications=%d", beforeCalls, random.calls.Load(), publications.Load())
	}
	if after := service.Snapshot(); !reflect.DeepEqual(after, current) {
		t.Fatalf("stale request mutated current generation\ngot: %#v\nwant: %#v", after, current)
	}
}

func TestAcceptedPatternPublishesOnceAfterMutationAndDuplicatePublishesNever(t *testing.T) {
	random := newCountingRandom(1)
	service := New(random, fixedWords{})
	service.generationIDs = &sequenceGenerationIDs{values: []string{"generation-atomic"}}
	initial := service.Set("terminal-1", "Overseer", testTree(), 1, "WELCOME")
	patternID := initial.Hack.Patterns[0].ID
	random.value.Store(99)
	beforeCalls := random.calls.Load()

	var published []*domain.PublicHackState
	accepted := service.ApplyHackPattern(patternID, func(state *domain.PublicHackState) {
		published = append(published, state)
		if !publicPatternIsUsed(state, patternID) {
			t.Fatalf("callback ran before used marking: %#v", state.Patterns)
		}
	})
	if !accepted || len(published) != 1 || random.calls.Load() != beforeCalls+1 {
		t.Fatalf("accepted=%t publications=%d RNG=%d->%d, want true/1/+1", accepted, len(published), beforeCalls, random.calls.Load())
	}
	published[0].Patterns[0].ID = "mutated-return"
	if snapshot := service.Snapshot(); snapshot.Hack.Patterns[0].ID == "mutated-return" {
		t.Fatal("callback projection retained a canonical reference")
	}

	beforeCalls = random.calls.Load()
	if service.ApplyHackPattern(patternID, func(*domain.PublicHackState) { published = append(published, nil) }) {
		t.Fatal("duplicate pattern was accepted")
	}
	if len(published) != 1 || random.calls.Load() != beforeCalls {
		t.Fatalf("duplicate request published or consumed RNG: publications=%d calls=%d->%d", len(published), beforeCalls, random.calls.Load())
	}
}

func activatePattern(service *Service, patternID string) (*domain.PublicHackState, bool) {
	var result *domain.PublicHackState
	ok := service.ApplyHackPattern(patternID, func(state *domain.PublicHackState) { result = state })
	return result, ok
}

func publicPatternIsUsed(state *domain.PublicHackState, patternID string) bool {
	if state == nil {
		return false
	}
	for _, pattern := range state.Patterns {
		if pattern.ID == patternID {
			return pattern.Used
		}
	}
	return false
}

type sequenceGenerationIDs struct {
	values []string
	next   int
}

func (source *sequenceGenerationIDs) Next() string {
	if source.next >= len(source.values) {
		return "generation-overflow"
	}
	value := source.values[source.next]
	source.next++
	return value
}

type countingRandom struct {
	value atomic.Int32
	calls atomic.Int64
}

func newCountingRandom(value int32) *countingRandom {
	random := &countingRandom{}
	random.value.Store(value)
	return random
}

func (random *countingRandom) Intn(limit int) int {
	random.calls.Add(1)
	return int(random.value.Load()) % limit
}

type constantRandom struct{}

func (*constantRandom) Intn(limit int) int {
	if limit <= 1 {
		return 0
	}
	return 1
}

type fixedWords struct{}

func (fixedWords) PickWords(length, count int) []string {
	pools := map[int][]string{
		4: {"CODE", "CAVE", "DUST", "IRON", "GATE", "BOLT", "RAMP", "CORE", "FUSE", "GRID", "LAMP", "MASK", "NODE", "PIPE", "RING", "RUST"},
		5: {"ALLOY", "ARMOR", "ATLAS", "BASIN", "BLAST", "BRICK", "CABLE", "CACHE", "CARGO", "CLIFF", "CLOCK", "CRANE", "CRATE", "CREEK", "DRAIN", "DRONE"},
	}
	return append([]string(nil), pools[length][:count]...)
}

func testTree() domain.ContentNode {
	return domain.ContentNode{
		ID: "root", Type: domain.NodeFolder, Name: "ROOT",
		Children: []domain.ContentNode{
			{
				ID: "docs", Type: domain.NodeFolder, Name: "DOCS",
				Children: []domain.ContentNode{
					{ID: "report", Type: domain.NodeEntry, Name: "REPORT", Description: "Report"},
					{ID: "read", Type: domain.NodeCommand, Name: "READ", Text: "Reading"},
				},
			},
		},
	}
}

func treeWithoutReport() domain.ContentNode {
	tree := testTree()
	tree.Children[0].Children = tree.Children[0].Children[1:]
	return tree
}
