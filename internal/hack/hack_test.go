package hack

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
)

func TestGenerateBoardUsesLevelRules(t *testing.T) {
	tests := []struct {
		level      int
		wordLength int
		wordCount  int
	}{
		{level: 1, wordLength: 4, wordCount: 12},
		{level: 2, wordLength: 5, wordCount: 13},
		{level: 3, wordLength: 6, wordCount: 14},
		{level: 4, wordLength: 7, wordCount: 15},
		{level: 5, wordLength: 8, wordCount: 16},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("level_%d", test.level), func(t *testing.T) {
			words := &recordingWordSource{}
			hack := GenerateBoard(test.level, newSequenceRandom(), words)

			if hack == nil {
				t.Fatal("GenerateBoard() returned nil")
			}
			if hack.Level != test.level || hack.WordLength != test.wordLength {
				t.Fatalf("level metadata = (%d, %d), want (%d, %d)", hack.Level, hack.WordLength, test.level, test.wordLength)
			}
			if words.length != test.wordLength || words.count != test.wordCount {
				t.Fatalf("PickWords() request = (%d, %d), want (%d, %d)", words.length, words.count, test.wordLength, test.wordCount)
			}
			if hack.AttemptsMax != 4 || hack.AttemptsLeft != 4 {
				t.Fatalf("attempts = %d/%d, want 4/4", hack.AttemptsLeft, hack.AttemptsMax)
			}
			if hack.Solved || hack.Failed || len(hack.UsedPatterns) != 0 || len(hack.Log) != 0 {
				t.Fatalf("new puzzle has progressed state: %#v", hack)
			}
			if len(hack.WordsByID) != test.wordCount {
				t.Fatalf("private lookup has %d entries, want %d candidates", len(hack.WordsByID), test.wordCount)
			}

			candidateCount := 0
			for _, candidate := range hack.WordsByID {
				candidateCount++
				if len(candidate.Text) != test.wordLength {
					t.Errorf("candidate %q has length %d, want %d", candidate.Text, len(candidate.Text), test.wordLength)
				}
			}
			if candidateCount != test.wordCount || candidateCount < 12 || candidateCount > 16 {
				t.Errorf("candidate count = %d, want %d in range 12..16", candidateCount, test.wordCount)
			}
			if !containsCandidate(hack, hack.SecretWord) {
				t.Errorf("secret word %q is not a visible candidate", hack.SecretWord)
			}

			if len(hack.Columns) != 2 {
				t.Fatalf("column count = %d, want 2", len(hack.Columns))
			}
			for columnIndex, column := range hack.Columns {
				if len(column.Text) != 16*12 {
					t.Errorf("column %d text length = %d, want 192", columnIndex, len(column.Text))
				}
				if len(column.Addresses) != 16 {
					t.Errorf("column %d address count = %d, want 16", columnIndex, len(column.Addresses))
				}
				for _, address := range column.Addresses {
					if len(address) != 6 || !strings.HasPrefix(address, "0x") {
						t.Errorf("column %d contains malformed address %q", columnIndex, address)
					}
				}
			}
		})
	}
}

func TestApplyGuessTransitions(t *testing.T) {
	t.Run("matching candidate solves without spending attempt", func(t *testing.T) {
		hack := testHackState()

		ApplyGuess(hack, "A1")

		if !hack.Solved || hack.Failed {
			t.Fatalf("result solved=%t failed=%t, want solved only", hack.Solved, hack.Failed)
		}
		if hack.AttemptsLeft != 4 {
			t.Fatalf("attempts left = %d, want 4", hack.AttemptsLeft)
		}
		assertLogContains(t, hack.Log, "> CODE", "> Точно!")
	})

	t.Run("wrong candidate reports likeness and spends attempt", func(t *testing.T) {
		hack := testHackState()

		ApplyGuess(hack, "A2")

		if hack.Solved || hack.Failed {
			t.Fatalf("result solved=%t failed=%t, want unfinished", hack.Solved, hack.Failed)
		}
		if hack.AttemptsLeft != 3 {
			t.Fatalf("attempts left = %d, want 3", hack.AttemptsLeft)
		}
		assertLogContains(t, hack.Log, "> CAVE", "> Отказ в доступе", "> 2/4 правильно.")
	})

	t.Run("filler clicks exhaust four attempts", func(t *testing.T) {
		hack := testHackState()

		for range 4 {
			ApplyGuess(hack, "0:4")
		}

		if !hack.Failed || hack.Solved {
			t.Fatalf("result solved=%t failed=%t, want failed only", hack.Solved, hack.Failed)
		}
		if hack.AttemptsLeft != 0 {
			t.Fatalf("attempts left = %d, want 0", hack.AttemptsLeft)
		}
		assertLogContains(t, hack.Log, "> !", "> 0/4 правильно.")
	})

	t.Run("unknown malformed out-of-range and in-word targets are ignored", func(t *testing.T) {
		targets := []string{"missing", "not:a-position", "2:0", "0:999", "0:0"}
		for _, target := range targets {
			t.Run(target, func(t *testing.T) {
				hack := testHackState()
				before := cloneHackState(t, hack)

				ApplyGuess(hack, target)

				if !reflect.DeepEqual(hack, before) {
					t.Fatalf("target %q mutated state\ngot:  %#v\nwant: %#v", target, hack, before)
				}
			})
		}
	})

	t.Run("terminal states ignore further guesses", func(t *testing.T) {
		for _, terminalState := range []struct {
			name   string
			mutate func(*domain.HackState)
		}{
			{name: "solved", mutate: func(h *domain.HackState) { h.Solved = true }},
			{name: "failed", mutate: func(h *domain.HackState) { h.Failed = true; h.AttemptsLeft = 0 }},
		} {
			t.Run(terminalState.name, func(t *testing.T) {
				hack := testHackState()
				terminalState.mutate(hack)
				before := cloneHackState(t, hack)

				ApplyGuess(hack, "A2")

				if !reflect.DeepEqual(hack, before) {
					t.Fatalf("terminal puzzle mutated\ngot:  %#v\nwant: %#v", hack, before)
				}
			})
		}
	})
}

func TestGeneratedBoardsContainThreeThroughSixValidPatterns(t *testing.T) {
	seenPairs := map[string]bool{}
	for level := 1; level <= 5; level++ {
		for iteration := 0; iteration < 200; iteration++ {
			state := GenerateBoard(level, newSequenceRandom(), &recordingWordSource{})
			if state == nil {
				t.Fatalf("level %d iteration %d returned nil", level, iteration)
			}
			patterns := discoverPatterns(state.Columns, state.UsedPatterns)
			if len(patterns) < 3 || len(patterns) > 6 {
				t.Fatalf("level %d iteration %d patterns = %d, want 3..6", level, iteration, len(patterns))
			}
			for _, pattern := range patterns {
				seenPairs[pattern.Pair] = true
				if pattern.Start/boardRowWidth != pattern.End/boardRowWidth {
					t.Fatalf("pattern %#v crosses a row", pattern)
				}
				text := state.Columns[pattern.Column].Text[pattern.Start : pattern.End+1]
				if strings.IndexFunc(text[1:len(text)-1], isASCIIAlpha) >= 0 {
					t.Fatalf("pattern %#v contains alphabetic interior %q", pattern, text)
				}
			}
		}
	}
	for _, pair := range []string{"()", "[]", "{}", "<>"} {
		if !seenPairs[pair] {
			t.Errorf("generated boards never exposed pair %s", pair)
		}
	}
}

func TestApplyPatternUsesExactOutcomeBuckets(t *testing.T) {
	dudRemovals := 0
	restores := 0
	for roll := 0; roll < 100; roll++ {
		state := patternTestState()
		state.AttemptsLeft = 1
		beforeCandidates := len(state.WordsByID)
		if !ApplyPattern(state, "0:0:3", &constantRandom{value: roll}) {
			t.Fatalf("roll %d rejected valid pattern", roll)
		}
		if len(state.WordsByID) == beforeCandidates-1 {
			dudRemovals++
			if !containsCandidate(state, state.SecretWord) || state.AttemptsLeft != 1 {
				t.Fatalf("roll %d corrupted dud-removal state: %#v", roll, state)
			}
		} else if state.AttemptsLeft == state.AttemptsMax {
			restores++
		} else {
			t.Fatalf("roll %d produced no defined effect: %#v", roll, state)
		}
	}
	if dudRemovals != 80 || restores != 20 {
		t.Fatalf("outcomes = %d dud removals/%d restores, want 80/20", dudRemovals, restores)
	}
}

func TestApplyPatternIsOneUseAndRestoresWhenNoDudRemains(t *testing.T) {
	state := patternTestState()
	state.AttemptsLeft = 2
	delete(state.WordsByID, "A2")
	state.Columns[0].Words = state.Columns[0].Words[:1]
	state.Columns[0].Text = state.Columns[0].Text[:9] + "...."

	if !ApplyPattern(state, "0:0:3", &constantRandom{value: 0}) {
		t.Fatal("valid pattern was rejected")
	}
	if state.AttemptsLeft != state.AttemptsMax {
		t.Fatalf("attempts = %d, want restored to %d", state.AttemptsLeft, state.AttemptsMax)
	}
	after := cloneHackState(t, state)
	if ApplyPattern(state, "0:0:3", &constantRandom{value: 0}) {
		t.Fatal("used pattern was accepted twice")
	}
	if !reflect.DeepEqual(state, after) {
		t.Fatalf("repeated pattern mutated state\ngot: %#v\nwant: %#v", state, after)
	}
}

func TestPatternDiscoveryHandlesStackedAndInvalidSpans(t *testing.T) {
	stacked := discoverPatternSpans([]domain.HackColumn{{Text: "((!!)", Words: []domain.HackWord{}}})
	wantIDs := []string{"0:0:4", "0:1:4"}
	if got := patternIDs(stacked); !reflect.DeepEqual(got, wantIDs) {
		t.Fatalf("stacked pattern IDs = %v, want %v", got, wantIDs)
	}

	invalid := []struct {
		name string
		text string
	}{
		{name: "alphabetic interior", text: "(A)"},
		{name: "mismatched closer", text: "[)"},
		{name: "cross row", text: "!!!!!!!!!!!()"},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			if patterns := discoverPatternSpans([]domain.HackColumn{{Text: test.text}}); len(patterns) != 0 {
				t.Fatalf("discoverPatternSpans(%q) = %#v, want none", test.text, patterns)
			}
		})
	}
}

func TestDudRemovalRevealsDynamicPatternImmediately(t *testing.T) {
	state := &domain.HackState{
		Level: 1, WordLength: 4, AttemptsMax: 4, AttemptsLeft: 4, SecretWord: "CODE",
		WordsByID:    map[string]domain.HackCandidate{"A1": {Text: "DUST"}, "B1": {Text: "CODE"}},
		UsedPatterns: map[string]struct{}{},
		Columns: []domain.HackColumn{
			{Text: "(DUST)!!!!!!", Words: []domain.HackWord{{ID: "A1", Start: 1, Length: 4}}},
			{Text: "[]CODE!!!!!!", Words: []domain.HackWord{{ID: "B1", Start: 2, Length: 4}}},
		},
	}
	if patterns := discoverPatternSpans(state.Columns); !reflect.DeepEqual(patternIDs(patterns), []string{"1:0:1"}) {
		t.Fatalf("initial patterns = %#v", patterns)
	}
	if !ApplyPattern(state, "1:0:1", &constantRandom{value: 0}) {
		t.Fatal("available pattern was rejected")
	}
	if got := patternIDs(discoverPatternSpans(state.Columns)); !reflect.DeepEqual(got, []string{"0:0:5", "1:0:1"}) {
		t.Fatalf("post-dud patterns = %v, want dynamic and used spans", got)
	}
	public := PublicState(state)
	if public == nil || len(public.Patterns) != 2 || public.Patterns[0].Used || !public.Patterns[1].Used {
		t.Fatalf("public dynamic/used pattern state = %#v", public)
	}
}

func TestGeneratedBoardHasNoPlayerAdministratorEntry(t *testing.T) {
	hack := GenerateBoard(1, newSequenceRandom(), &recordingWordSource{})
	if hack == nil {
		t.Fatal("GenerateBoard() returned nil")
	}
	for id, candidate := range hack.WordsByID {
		if candidate.Text == "SUCCESS" {
			t.Fatalf("private lookup retained administrator candidate %q", id)
		}
	}
	for _, column := range hack.Columns {
		if strings.Contains(column.Text, "SUCCESS") {
			t.Fatalf("public board retained administrator entry: %q", column.Text)
		}
	}
}

func TestForceSuccessOnlyCompletesEligiblePuzzle(t *testing.T) {
	t.Run("active puzzle", func(t *testing.T) {
		hack := testHackState()

		ForceSuccess(hack)

		if !hack.Solved || hack.Failed || hack.AttemptsLeft != 4 {
			t.Fatalf("forced puzzle state = %#v", hack)
		}
		assertLogContains(t, hack.Log, "> CODE", "> Точно!")
	})

	t.Run("nil and terminal puzzles are no-op", func(t *testing.T) {
		ForceSuccess(nil)

		for _, terminalState := range []struct {
			name   string
			mutate func(*domain.HackState)
		}{
			{name: "solved", mutate: func(h *domain.HackState) { h.Solved = true }},
			{name: "failed", mutate: func(h *domain.HackState) { h.Failed = true; h.AttemptsLeft = 0 }},
		} {
			t.Run(terminalState.name, func(t *testing.T) {
				hack := testHackState()
				terminalState.mutate(hack)
				before := cloneHackState(t, hack)

				ForceSuccess(hack)

				if !reflect.DeepEqual(hack, before) {
					t.Fatalf("terminal puzzle mutated\ngot:  %#v\nwant: %#v", hack, before)
				}
			})
		}
	})
}

func TestPublicStateExcludesPrivatePuzzleFields(t *testing.T) {
	if got := PublicState(nil); got != nil {
		t.Fatalf("PublicState(nil) = %#v, want nil", got)
	}

	hack := testHackState()
	public := PublicState(hack)
	if public == nil {
		t.Fatal("PublicState() returned nil")
	}
	if public.Level != hack.Level || public.WordLength != hack.WordLength || public.AttemptsLeft != hack.AttemptsLeft {
		t.Fatalf("public state dropped gameplay fields: %#v", public)
	}

	raw, err := json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"secretWord", "wordsById"} {
		if strings.Contains(string(raw), secret) {
			t.Errorf("public JSON contains private value %q: %s", secret, raw)
		}
	}
}

type sequenceRandom struct {
	next int
}

func newSequenceRandom() *sequenceRandom {
	return &sequenceRandom{next: 11}
}

func (r *sequenceRandom) Intn(limit int) int {
	value := r.next % limit
	r.next += 37
	return value
}

type constantRandom struct {
	value int
}

func (r *constantRandom) Intn(limit int) int {
	return r.value % limit
}

type recordingWordSource struct {
	length int
	count  int
}

func (s *recordingWordSource) PickWords(length, count int) []string {
	s.length = length
	s.count = count
	pool := wordsByLength[length]
	return append([]string(nil), pool[:count]...)
}

var wordsByLength = map[int][]string{
	4: {"RUIN", "PALM", "IRON", "GATE", "BOLT", "RAMP", "CORE", "DUST", "FUSE", "GRID", "LAMP", "MASK", "NODE", "PIPE", "RING", "RUST"},
	5: {"ALLOY", "ARMOR", "ATLAS", "BASIN", "BLAST", "BRICK", "CABLE", "CACHE", "CARGO", "CLIFF", "CLOCK", "CRANE", "CRATE", "CREEK", "DRAIN", "DRONE"},
	6: {"ANCHOR", "BASALT", "BEACON", "BUNKER", "CAVERN", "CIPHER", "CONVOY", "COURSE", "DEBRIS", "ENGINE", "FILTER", "FLIGHT", "FUNGUS", "GARDEN", "GIRDER", "HARBOR"},
	7: {"ANDROID", "ARCHIVE", "ARSENAL", "ARTICLE", "BATTERY", "BEDROCK", "BOMBARD", "BREAKER", "CAPSULE", "CHAMBER", "CIRCUIT", "COOLANT", "CORRODE", "CRUMBLE", "CRYSTAL", "DOSSIER"},
	8: {"CONCRETE", "DISTANCE", "ELECTRIC", "CHEMICAL", "GENERATE", "HOSPITAL", "INDUSTRY", "JUNCTION", "KEYSTONE", "LOCATION", "MOUNTAIN", "NAVIGATE", "OVERLOAD", "PIPELINE", "QUANTITY", "RADIATOR"},
}

func testHackState() *domain.HackState {
	return &domain.HackState{
		Level:        1,
		WordLength:   4,
		AttemptsMax:  4,
		AttemptsLeft: 4,
		SecretWord:   "CODE",
		WordsByID: map[string]domain.HackCandidate{
			"A1": {Text: "CODE"},
			"A2": {Text: "CAVE"},
			"A3": {Text: "DUST"},
			"A4": {Text: "IRON"},
		},
		UsedPatterns: map[string]struct{}{},
		Columns: []domain.HackColumn{
			{
				Addresses: []string{"0xC000"},
				Text:      "CODE!CAVE!DUST!IRON!",
				Words: []domain.HackWord{
					{ID: "A1", Start: 0, Length: 4},
					{ID: "A2", Start: 5, Length: 4},
					{ID: "A3", Start: 10, Length: 4},
					{ID: "A4", Start: 15, Length: 4},
				},
			},
			{Addresses: []string{"0xD000"}, Text: "!!!!!!!!!!!!", Words: []domain.HackWord{}},
		},
	}
}

func patternTestState() *domain.HackState {
	return &domain.HackState{
		Level:        1,
		WordLength:   4,
		AttemptsMax:  4,
		AttemptsLeft: 4,
		SecretWord:   "CODE",
		WordsByID:    map[string]domain.HackCandidate{"A1": {Text: "CODE"}, "A2": {Text: "DUST"}},
		UsedPatterns: map[string]struct{}{},
		Log:          []string{},
		Columns: []domain.HackColumn{
			{
				Addresses: []string{"0xC000"},
				Text:      "(!!)CODE!DUST",
				Words: []domain.HackWord{
					{ID: "A1", Start: 4, Length: 4},
					{ID: "A2", Start: 9, Length: 4},
				},
			},
			{Addresses: []string{"0xD000"}, Text: "!!!!!!!!!!!!", Words: []domain.HackWord{}},
		},
	}
}

func cloneHackState(t *testing.T, source *domain.HackState) *domain.HackState {
	t.Helper()
	clone := *source
	clone.Log = append([]string(nil), source.Log...)
	clone.WordsByID = make(map[string]domain.HackCandidate, len(source.WordsByID))
	for id, candidate := range source.WordsByID {
		clone.WordsByID[id] = candidate
	}
	clone.UsedPatterns = make(map[string]struct{}, len(source.UsedPatterns))
	for id := range source.UsedPatterns {
		clone.UsedPatterns[id] = struct{}{}
	}
	clone.Columns = make([]domain.HackColumn, len(source.Columns))
	for i, column := range source.Columns {
		clone.Columns[i] = column
		clone.Columns[i].Addresses = append([]string(nil), column.Addresses...)
		if column.Words != nil {
			clone.Columns[i].Words = append([]domain.HackWord{}, column.Words...)
		}
	}
	return &clone
}

func patternIDs(patterns []domain.HackPattern) []string {
	ids := make([]string, len(patterns))
	for index, pattern := range patterns {
		ids[index] = pattern.ID
	}
	return ids
}

func containsCandidate(hack *domain.HackState, text string) bool {
	for _, candidate := range hack.WordsByID {
		if candidate.Text == text {
			return true
		}
	}
	return false
}

func assertLogContains(t *testing.T, log []string, lines ...string) {
	t.Helper()
	for _, line := range lines {
		found := false
		for _, got := range log {
			if got == line {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("log %q does not contain %q", log, line)
		}
	}
}
