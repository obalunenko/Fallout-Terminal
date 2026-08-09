package hack

import "strings"

// WordSource supplies candidate words for a hacking board.
type WordSource interface {
	PickWords(length, count int) []string
}

// WordBank supplies words from the built-in, length-indexed vocabulary.
// Its random source is injectable so board generation remains deterministic in
// tests. A nil random source uses the package's system-backed default.
type WordBank struct {
	Random Random
}

// PickWords returns up to count distinct uppercase words of the requested
// length without modifying the built-in vocabulary.
func (w WordBank) PickWords(length, count int) []string {
	if count <= 0 {
		return []string{}
	}

	pool := append([]string(nil), builtinWordsByLength[length]...)
	picked := make([]string, 0, min(count, len(pool)))
	random := randomOrDefault(w.Random)
	for len(picked) < count && len(pool) > 0 {
		index := safeIntn(random, len(pool))
		picked = append(picked, pool[index])
		pool = append(pool[:index], pool[index+1:]...)
	}
	return picked
}

var builtinWordsByLength = buildWordBuckets([]string{
	"RUIN", "PALM", "IRON", "GATE", "BOLT", "RAMP", "CORE", "DUST", "FUSE", "GRID",
	"LAMP", "MASK", "NODE", "PIPE", "RING", "RUST", "SHIP", "SILO", "TANK", "VENT",
	"WIRE", "ZONE", "CODE", "DATA", "DISK", "DOOR", "EXIT", "FIRE", "FUEL", "GEAR",
	"LOCK", "MINE", "PLUG", "PUMP", "ROAD", "ROOF", "SEAL", "TENT", "WALL", "ACID",
	"ARMY", "ATOM", "BEAM", "CAVE", "COAL", "COIL", "DOSE", "DUNE", "DIRT", "JUNK",

	"ALLOY", "ARMOR", "ATLAS", "BASIN", "BLAST", "BRICK", "CABLE", "CACHE", "CARGO",
	"CLIFF", "CLOCK", "CRANE", "CRATE", "CREEK", "DRAIN", "DRONE", "FLARE", "FLOOR",
	"FAULT", "FENCE", "FORGE", "FUMES", "GRATE", "GUARD", "HATCH", "HINGE", "MOTOR",
	"PANEL", "RIDGE", "RIFLE", "ROBOT", "ROVER", "SHAFT", "SIREN", "SPIKE", "STEEL",
	"STOCK", "STORM", "SWORD", "TOWER", "TRACE", "TRACK", "TRAIL", "TRUNK", "VALVE", "VAULT",

	"ANCHOR", "BASALT", "BEACON", "BUNKER", "CAVERN", "CIPHER", "CONVOY", "COURSE",
	"DEBRIS", "ENGINE", "FILTER", "FLIGHT", "FUNGUS", "GARDEN", "GIRDER", "HARBOR",
	"HOLLOW", "LEDGER", "MARROW", "MIRAGE", "NOZZLE", "OUTAGE", "PISTON", "PLAGUE",
	"POISON", "PYLONS", "QUARRY", "RANCID", "ROCKET", "RUBBLE", "RUSTIC", "SENSOR",
	"SLUDGE", "SLUICE", "SOCKET", "SWITCH", "SYSTEM", "TANKER", "TUNNEL", "TURRET",
	"VESSEL", "WANDER", "WELDER",

	"ANDROID", "ARCHIVE", "ARSENAL", "ARTICLE", "BATTERY", "BEDROCK", "BOMBARD",
	"BREAKER", "CAPSULE", "CHAMBER", "CIRCUIT", "COOLANT", "CORRODE", "CRUMBLE",
	"CRYSTAL", "DOSSIER", "EXHAUST", "FALLOUT", "FOUNDRY", "FREIGHT", "GEARBOX",
	"IMPLANT", "KESTREL", "LANDING", "LEAKAGE", "MACHINE", "MANGLED", "MINERAL",
	"MONITOR", "NETWORK", "NUCLEUS", "OUTPOST", "PATHWAY", "PLATING", "PROTONS",
	"QUARTER", "RAILWAY", "REACTOR", "RELAYED", "RESIDUE", "RUPTURE", "SCANNER",
	"SEALANT", "SHATTER", "SHIPPED", "SILICON", "SKYLINE", "SPARROW", "STATION",
	"STORAGE", "THERMAL", "TORNADO", "TRIGGER", "TURBINE", "VACUUMS", "WARHEAD",

	"CONCRETE", "DISTANCE", "ELECTRIC", "CHEMICAL", "GENERATE", "HOSPITAL",
	"INDUSTRY", "JUNCTION", "KEYSTONE", "LOCATION", "MOUNTAIN", "NAVIGATE",
	"OVERLOAD", "PIPELINE", "QUANTITY", "RADIATOR", "SIGNPOST", "TERMINAL",
	"UNIFORMS", "VOLTAGES", "YIELDING", "ZEPPELIN",
})

func buildWordBuckets(words []string) map[int][]string {
	buckets := map[int][]string{4: {}, 5: {}, 6: {}, 7: {}, 8: {}}
	seen := make(map[string]struct{}, len(words))
	for _, raw := range words {
		word := strings.ToUpper(raw)
		if _, supported := buckets[len(word)]; !supported {
			continue
		}
		if _, duplicate := seen[word]; duplicate {
			continue
		}
		seen[word] = struct{}{}
		buckets[len(word)] = append(buckets[len(word)], word)
	}
	return buckets
}
