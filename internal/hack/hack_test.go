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
			if hack.Solved || hack.Failed || hack.AdminModeUsed || len(hack.Log) != 0 {
				t.Fatalf("new puzzle has progressed state: %#v", hack)
			}
			if len(hack.WordsByID) != test.wordCount+1 {
				t.Fatalf("private lookup has %d entries, want %d candidates plus administrator", len(hack.WordsByID), test.wordCount)
			}

			candidateCount := 0
			adminCount := 0
			for _, candidate := range hack.WordsByID {
				if candidate.IsAdmin {
					adminCount++
					if candidate.Text != "SUCCESS" {
						t.Errorf("administrator entry text = %q, want SUCCESS", candidate.Text)
					}
					continue
				}
				candidateCount++
				if len(candidate.Text) != test.wordLength {
					t.Errorf("candidate %q has length %d, want %d", candidate.Text, len(candidate.Text), test.wordLength)
				}
			}
			if candidateCount != test.wordCount || candidateCount < 12 || candidateCount > 16 {
				t.Errorf("candidate count = %d, want %d in range 12..16", candidateCount, test.wordCount)
			}
			if adminCount != 1 {
				t.Errorf("administrator entry count = %d, want 1", adminCount)
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

func TestApplyAdminKeepsSecretAndOneDecoyAtMostOnce(t *testing.T) {
	hack := testHackState()

	ApplyAdmin(hack, &constantRandom{value: 0})

	if !hack.AdminModeUsed {
		t.Fatal("administrator mode was not marked used")
	}
	if hack.AttemptsLeft != 4 || hack.Solved || hack.Failed {
		t.Fatalf("administrator changed puzzle outcome or attempts: %#v", hack)
	}
	if !containsCandidate(hack, hack.SecretWord) {
		t.Fatalf("administrator removed secret word %q", hack.SecretWord)
	}
	if got := nonAdminCandidateCount(hack); got != 2 {
		t.Fatalf("administrator left %d candidates, want secret plus one decoy", got)
	}
	if !strings.Contains(hack.Columns[0].Text, "....") {
		t.Fatalf("removed words were not replaced with dots: %q", hack.Columns[0].Text)
	}
	assertLogContains(t, hack.Log, "> Режим администратора активирован.")

	afterFirstUse := cloneHackState(t, hack)
	ApplyAdmin(hack, &constantRandom{value: 1})
	if !reflect.DeepEqual(hack.WordsByID, afterFirstUse.WordsByID) || !reflect.DeepEqual(hack.Columns, afterFirstUse.Columns) {
		t.Fatalf("second administrator action removed more candidates\ngot:  %#v\nwant: %#v", hack, afterFirstUse)
	}
	if hack.AttemptsLeft != afterFirstUse.AttemptsLeft || len(hack.Log) != len(afterFirstUse.Log)+1 {
		t.Fatalf("second administrator action changed anything except its compatibility log: %#v", hack)
	}
}

func TestAdministratorWordDelegatesToAdminAction(t *testing.T) {
	hack := testHackState()

	ApplyGuess(hack, "B1")

	if !hack.AdminModeUsed {
		t.Fatal("guessing administrator word did not activate administrator mode")
	}
	if hack.AttemptsLeft != 4 {
		t.Fatalf("administrator word spent an attempt: %d", hack.AttemptsLeft)
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
			"B1": {Text: "SUCCESS", IsAdmin: true},
		},
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
			{
				Addresses: []string{"0xD000"},
				Text:      "SUCCESS!!!!!",
				Words: []domain.HackWord{
					{ID: "B1", Start: 0, Length: 7, IsAdmin: true},
				},
			},
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
	clone.Columns = make([]domain.HackColumn, len(source.Columns))
	for i, column := range source.Columns {
		clone.Columns[i] = column
		clone.Columns[i].Addresses = append([]string(nil), column.Addresses...)
		clone.Columns[i].Words = append([]domain.HackWord(nil), column.Words...)
	}
	return &clone
}

func containsCandidate(hack *domain.HackState, text string) bool {
	for _, candidate := range hack.WordsByID {
		if !candidate.IsAdmin && candidate.Text == text {
			return true
		}
	}
	return false
}

func nonAdminCandidateCount(hack *domain.HackState) int {
	count := 0
	for _, candidate := range hack.WordsByID {
		if !candidate.IsAdmin {
			count++
		}
	}
	return count
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
