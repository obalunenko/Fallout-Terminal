// Package hack implements the server-authoritative terminal hacking game.
package hack

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
)

const (
	boardRows       = 16
	boardRowWidth   = 12
	columnChars     = boardRows * boardRowWidth
	wordGap         = 4
	placementTries  = 300
	administrator   = "SUCCESS"
	maximumAttempts = 4
)

var fillerPool = []byte("!@#$%^&*()_+-=[]{}\\|;:'\",.<>/?~")

// Random is the small random-number boundary used by board generation.
type Random interface {
	Intn(limit int) int
}

type systemRandom struct{}

func (systemRandom) Intn(limit int) int {
	if limit <= 1 {
		return 0
	}
	var bytes [8]byte
	if _, err := cryptorand.Read(bytes[:]); err != nil {
		return 0
	}
	return int(binary.LittleEndian.Uint64(bytes[:]) % uint64(limit))
}

// GenerateBoard creates a fresh private hacking aggregate. Random and words
// are injectable; nil values use the system random source and built-in bank.
func GenerateBoard(level int, random Random, words WordSource) *domain.HackState {
	random = randomOrDefault(random)
	wordLength := levelWordLength(level)
	wordCount := 11 + clamp(level, 1, 5)
	if words == nil {
		words = WordBank{Random: random}
	}
	candidates := normalizeCandidates(words.PickWords(wordLength, wordCount), wordLength, wordCount)
	if len(candidates) < wordCount {
		fallback := WordBank{Random: random}.PickWords(wordLength, wordCount)
		candidates = supplementCandidates(candidates, fallback, wordCount)
	}
	if len(candidates) == 0 {
		return nil
	}

	state := &domain.HackState{
		Level:        level,
		WordLength:   wordLength,
		AttemptsMax:  maximumAttempts,
		AttemptsLeft: maximumAttempts,
		SecretWord:   candidates[safeIntn(random, len(candidates))],
		WordsByID:    make(map[string]domain.HackCandidate, len(candidates)+1),
		Log:          []string{},
	}

	columnA := newColumnBuilder("A", random)
	columnB := newColumnBuilder("B", random)
	if id, ok := columnB.place(administrator, true, 0); ok {
		state.WordsByID[id] = domain.HackCandidate{Text: administrator, IsAdmin: true}
	}
	for index, text := range candidates {
		builder := columnA
		if index%2 != 0 {
			builder = columnB
		}
		id, ok := builder.place(text, false, -1)
		if !ok {
			continue
		}
		state.WordsByID[id] = domain.HackCandidate{Text: text}
	}

	state.Columns = []domain.HackColumn{
		columnA.finish(),
		columnB.finish(),
	}
	return state
}

// ApplyGuess applies a candidate ID or a "column:character" filler target.
// Unknown, stale, and terminal-state actions are ignored.
func ApplyGuess(state *domain.HackState, targetID string) {
	if state == nil || state.Solved || state.Failed {
		return
	}

	if candidate, ok := state.WordsByID[targetID]; ok {
		if candidate.IsAdmin {
			ApplyAdmin(state, nil)
			return
		}
		pushLog(state, candidate.Text)
		matches := countMatches(candidate.Text, state.SecretWord)
		if matches == state.WordLength {
			state.Solved = true
			pushSuccessLog(state)
			return
		}
		spendAttempt(state, matches)
		return
	}

	columnIndex, characterIndex, ok := parseFillerTarget(targetID)
	if !ok || columnIndex >= len(state.Columns) {
		return
	}
	column := &state.Columns[columnIndex]
	if characterIndex >= len(column.Text) || containsWord(column.Words, characterIndex) {
		return
	}
	pushLog(state, string(column.Text[characterIndex]))
	spendAttempt(state, 0)
}

// ApplyAdmin removes every ordinary candidate except the secret and one
// random decoy. The compatibility log is emitted on every eligible use, while
// board mutation occurs only once.
func ApplyAdmin(state *domain.HackState, random Random) {
	if state == nil || state.Solved || state.Failed {
		return
	}
	pushLog(state, "Режим администратора активирован.")
	if state.AdminModeUsed {
		return
	}
	state.AdminModeUsed = true
	random = randomOrDefault(random)

	candidateIDs := make([]string, 0, len(state.WordsByID))
	secretID := ""
	for _, column := range state.Columns {
		for _, word := range column.Words {
			candidate, exists := state.WordsByID[word.ID]
			if !exists || candidate.IsAdmin {
				continue
			}
			candidateIDs = append(candidateIDs, word.ID)
			if secretID == "" && candidate.Text == state.SecretWord {
				secretID = word.ID
			}
		}
	}
	if secretID == "" && len(candidateIDs) > 0 {
		secretID = candidateIDs[0]
	}

	decoys := make([]string, 0, len(candidateIDs))
	for _, id := range candidateIDs {
		if id != secretID {
			decoys = append(decoys, id)
		}
	}
	decoyID := ""
	if len(decoys) > 0 {
		decoyID = decoys[safeIntn(random, len(decoys))]
	}
	for _, id := range candidateIDs {
		if id == secretID || id == decoyID {
			continue
		}
		dotCandidate(state, id)
	}
}

// ForceSuccess solves a currently active puzzle without spending an attempt.
func ForceSuccess(state *domain.HackState) {
	if state == nil || state.Solved || state.Failed {
		return
	}
	pushLog(state, state.SecretWord)
	state.Solved = true
	pushSuccessLog(state)
}

// PublicState creates a detached client-safe projection of private state.
func PublicState(state *domain.HackState) *domain.PublicHackState {
	if state == nil {
		return nil
	}
	public := &domain.PublicHackState{
		Level:        state.Level,
		WordLength:   state.WordLength,
		AttemptsMax:  state.AttemptsMax,
		AttemptsLeft: state.AttemptsLeft,
		Solved:       state.Solved,
		Failed:       state.Failed,
		Log:          append([]string(nil), state.Log...),
		Columns:      make([]domain.HackColumn, len(state.Columns)),
	}
	for index, column := range state.Columns {
		public.Columns[index] = column
		public.Columns[index].Addresses = append([]string(nil), column.Addresses...)
		public.Columns[index].Words = append([]domain.HackWord(nil), column.Words...)
	}
	return public
}

type columnBuilder struct {
	prefix string
	random Random
	chars  []byte
	used   []bool
	words  []domain.HackWord
	nextID int
}

func newColumnBuilder(prefix string, random Random) *columnBuilder {
	return &columnBuilder{
		prefix: prefix,
		random: random,
		chars:  make([]byte, columnChars),
		used:   make([]bool, columnChars),
		nextID: 1,
	}
}

func (builder *columnBuilder) place(text string, admin bool, requestedStart int) (string, bool) {
	start := requestedStart
	if start < 0 {
		start = builder.randomStart(len(text))
	}
	if !builder.canPlace(start, len(text)) {
		return "", false
	}
	id := fmt.Sprintf("%s%d", builder.prefix, builder.nextID)
	builder.nextID++
	copy(builder.chars[start:start+len(text)], text)
	for index := start; index < start+len(text); index++ {
		builder.used[index] = true
	}
	builder.words = append(builder.words, domain.HackWord{
		ID: id, Start: start, Length: len(text), IsAdmin: admin,
	})
	return id, true
}

func (builder *columnBuilder) randomStart(length int) int {
	limit := columnChars - length + 1
	for range placementTries {
		candidate := safeIntn(builder.random, limit)
		if builder.canPlace(candidate, length) {
			return candidate
		}
	}
	for candidate := 0; candidate < limit; candidate++ {
		if builder.canPlace(candidate, length) {
			return candidate
		}
	}
	return -1
}

func (builder *columnBuilder) canPlace(start, length int) bool {
	if start < 0 || length <= 0 || start+length > len(builder.chars) {
		return false
	}
	from := max(0, start-wordGap)
	to := min(len(builder.chars), start+length+wordGap)
	for index := from; index < to; index++ {
		if builder.used[index] {
			return false
		}
	}
	return true
}

func (builder *columnBuilder) finish() domain.HackColumn {
	for index := range builder.chars {
		if !builder.used[index] {
			builder.chars[index] = fillerPool[safeIntn(builder.random, len(fillerPool))]
		}
	}
	return domain.HackColumn{
		Addresses: generateAddresses(boardRows, builder.random),
		Text:      string(builder.chars),
		Words:     append([]domain.HackWord(nil), builder.words...),
	}
}

func generateAddresses(count int, random Random) []string {
	address := safeIntn(random, 0x4000) + 0xC000
	steps := [...]int{0x0C, 0x10, 0x14, 0x18}
	step := steps[safeIntn(random, len(steps))]
	addresses := make([]string, count)
	for index := range addresses {
		addresses[index] = fmt.Sprintf("0x%04X", address&0xFFFF)
		address += step
	}
	return addresses
}

func normalizeCandidates(input []string, length, count int) []string {
	result := make([]string, 0, min(count, len(input)))
	seen := make(map[string]struct{}, len(input))
	for _, raw := range input {
		word := strings.ToUpper(raw)
		if len(word) != length {
			continue
		}
		if _, duplicate := seen[word]; duplicate {
			continue
		}
		seen[word] = struct{}{}
		result = append(result, word)
		if len(result) == count {
			break
		}
	}
	return result
}

func supplementCandidates(candidates, fallback []string, count int) []string {
	result := append([]string(nil), candidates...)
	seen := make(map[string]struct{}, len(result))
	for _, word := range result {
		seen[word] = struct{}{}
	}
	for _, word := range fallback {
		if len(result) == count {
			break
		}
		if _, duplicate := seen[word]; duplicate {
			continue
		}
		seen[word] = struct{}{}
		result = append(result, word)
	}
	return result
}

func dotCandidate(state *domain.HackState, id string) {
	for columnIndex := range state.Columns {
		column := &state.Columns[columnIndex]
		for wordIndex, word := range column.Words {
			if word.ID != id {
				continue
			}
			column.Text = column.Text[:word.Start] + strings.Repeat(".", word.Length) + column.Text[word.Start+word.Length:]
			column.Words = append(column.Words[:wordIndex], column.Words[wordIndex+1:]...)
			delete(state.WordsByID, id)
			return
		}
	}
}

func parseFillerTarget(target string) (int, int, bool) {
	parts := strings.Split(target, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, 0, false
	}
	column, err := strconv.Atoi(parts[0])
	if err != nil || column < 0 {
		return 0, 0, false
	}
	character, err := strconv.Atoi(parts[1])
	if err != nil || character < 0 {
		return 0, 0, false
	}
	return column, character, true
}

func containsWord(words []domain.HackWord, index int) bool {
	for _, word := range words {
		if index >= word.Start && index < word.Start+word.Length {
			return true
		}
	}
	return false
}

func countMatches(candidate, secret string) int {
	limit := min(len(candidate), len(secret))
	matches := 0
	for index := 0; index < limit; index++ {
		if candidate[index] == secret[index] {
			matches++
		}
	}
	return matches
}

func spendAttempt(state *domain.HackState, matches int) {
	state.AttemptsLeft = max(0, state.AttemptsLeft-1)
	pushLog(state, "Отказ в доступе", fmt.Sprintf("%d/%d правильно.", matches, state.WordLength))
	if state.AttemptsLeft == 0 {
		state.Failed = true
	}
}

func pushSuccessLog(state *domain.HackState) {
	pushLog(state, "Точно!", "Пожалуйста,", "подождите", "входа в систему.")
}

func pushLog(state *domain.HackState, lines ...string) {
	for _, line := range lines {
		state.Log = append(state.Log, "> "+line)
	}
}

func levelWordLength(level int) int {
	if level >= 1 && level <= 5 {
		return level + 3
	}
	return 4
}

func clamp(value, low, high int) int {
	return min(max(value, low), high)
}

func randomOrDefault(random Random) Random {
	if random == nil {
		return systemRandom{}
	}
	return random
}

func safeIntn(random Random, limit int) int {
	if limit <= 1 {
		return 0
	}
	value := random.Intn(limit)
	value %= limit
	if value < 0 {
		value += limit
	}
	return value
}
