// Common English words for the hacking minigame, bucketed by actual length
// (not by manual grouping) so a typo never lands a word in the wrong bucket.
// Difficulty 1..4 maps to word length 4..7 (see hack.js).
const RAW_WORDS = [
  'RUIN', 'PALM', 'IRON', 'GATE', 'BOLT', 'RAMP', 'CORE', 'DUST', 'FUSE', 'GRID',
  'LAMP', 'MASK', 'NODE', 'PIPE', 'RING', 'RUST', 'SHIP', 'SILO', 'TANK', 'VENT',
  'WIRE', 'ZONE', 'CODE', 'DATA', 'DISK', 'DOOR', 'EXIT', 'FIRE', 'FUEL', 'GEAR',
  'LOCK', 'MINE', 'PLUG', 'PUMP', 'ROAD', 'ROOF', 'SEAL', 'TENT', 'WALL', 'ACID',
  'ARMY', 'ATOM', 'BEAM', 'CAVE', 'COAL', 'COIL', 'DOSE', 'DUNE', 'DIRT', 'JUNK',

  'ALLOY', 'ARMOR', 'ATLAS', 'BASIN', 'BLAST', 'BRICK', 'CABLE', 'CACHE', 'CARGO',
  'CLIFF', 'CLOCK', 'CRANE', 'CRATE', 'CREEK', 'DRAIN', 'DRONE', 'FLARE', 'FLOOR',
  'FAULT', 'FENCE', 'FORGE', 'FUMES', 'GRATE', 'GUARD', 'HATCH', 'HINGE', 'MOTOR',
  'PANEL', 'RIDGE', 'RIFLE', 'ROBOT', 'ROVER', 'SHAFT', 'SIREN', 'SPIKE', 'STEEL',
  'STOCK', 'STORM', 'SWORD', 'TOWER', 'TRACE', 'TRACK', 'TRAIL', 'TRUNK', 'VALVE', 'VAULT',

  'ANCHOR', 'BASALT', 'BEACON', 'BUNKER', 'CAVERN', 'CIPHER', 'CONVOY', 'COURSE',
  'DEBRIS', 'ENGINE', 'FILTER', 'FLIGHT', 'FUNGUS', 'GARDEN', 'GIRDER', 'HARBOR',
  'HOLLOW', 'LEDGER', 'MARROW', 'MIRAGE', 'NOZZLE', 'OUTAGE', 'PISTON', 'PLAGUE',
  'POISON', 'PYLONS', 'QUARRY', 'RANCID', 'ROCKET', 'RUBBLE', 'RUSTIC', 'SENSOR',
  'SLUDGE', 'SLUICE', 'SOCKET', 'SWITCH', 'SYSTEM', 'TANKER', 'TUNNEL', 'TURRET',
  'VESSEL', 'WANDER', 'WELDER',

  'ANDROID', 'ARCHIVE', 'ARSENAL', 'ARTICLE', 'BATTERY', 'BEDROCK', 'BOMBARD',
  'BREAKER', 'CAPSULE', 'CHAMBER', 'CIRCUIT', 'COOLANT', 'CORRODE', 'CRUMBLE',
  'CRYSTAL', 'DOSSIER', 'EXHAUST', 'FALLOUT', 'FOUNDRY', 'FREIGHT', 'GEARBOX',
  'IMPLANT', 'KESTREL', 'LANDING', 'LEAKAGE', 'MACHINE', 'MANGLED', 'MINERAL',
  'MONITOR', 'NETWORK', 'NUCLEUS', 'OUTPOST', 'PATHWAY', 'PLATING', 'PROTONS',
  'QUARTER', 'RAILWAY', 'REACTOR', 'RELAYED', 'RESIDUE', 'RUPTURE', 'SCANNER',
  'SEALANT', 'SHATTER', 'SHIPPED', 'SILICON', 'SKYLINE', 'SPARROW', 'STATION',
  'STORAGE', 'THERMAL', 'TORNADO', 'TRIGGER', 'TURBINE', 'VACUUMS', 'WARHEAD',

  'CONCRETE', 'DISTANCE', 'ELECTRIC', 'CHEMICAL', 'GENERATE', 'HOSPITAL',
  'INDUSTRY', 'JUNCTION', 'KEYSTONE', 'LOCATION', 'MOUNTAIN', 'NAVIGATE',
  'OVERLOAD', 'PIPELINE', 'QUANTITY', 'RADIATOR', 'SIGNPOST', 'TERMINAL',
  'UNIFORMS', 'VOLTAGES', 'YIELDING', 'ZEPPELIN',
];

const byLength = { 4: [], 5: [], 6: [], 7: [], 8: [] };
for (const raw of RAW_WORDS) {
  const word = raw.toUpperCase();
  if (byLength[word.length] && !byLength[word.length].includes(word)) {
    byLength[word.length].push(word);
  }
}

function pickWords(length, count) {
  const pool = (byLength[length] || []).slice();
  const picked = [];
  while (picked.length < count && pool.length) {
    const i = Math.floor(Math.random() * pool.length);
    picked.push(pool.splice(i, 1)[0]);
  }
  return picked;
}

module.exports = { pickWords, byLength };
