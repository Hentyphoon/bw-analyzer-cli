package analyze

import (
	"slices"

	"github.com/Hentyphoon/bw-analyzer-cli/internal/replay"
)

// StepKind groups build order entries.
type StepKind string

const (
	Structure StepKind = "structure" // build commands and building morphs
	Unit      StepKind = "unit"      // train commands and Zerg unit morphs
	Research  StepKind = "research"  // tech and upgrade commands
)

// The allowlists below name everything a player of each race can order,
// spelled exactly as screp reports it (checked by a test in
// internal/parser). Names not listed are dropped from build orders: hero and
// campaign units, map-only structures, Interceptors and Scarabs (refills,
// not decisions), and "Unknown 0x.." values from corrupt commands.
//
// Abilities every unit starts with (Scanner Sweep, Defensive Matrix,
// Infestation, Dark Swarm, Parasite, Feedback, Healing) and merges (Archon,
// Dark Archon) are never ordered as research or training, so they are absent.
//
// These maps are lookup tables, filled at init and never written after, so
// sharing them across goroutines is safe.
var allowed = map[replay.Race]map[string]StepKind{
	replay.Terran: kinds(
		[]string{
			"Command Center", "ComSat", "Nuclear Silo", "Supply Depot", "Refinery",
			"Barracks", "Academy", "Factory", "Starport", "Control Tower",
			"Science Facility", "Covert Ops", "Physics Lab", "Machine Shop",
			"Engineering Bay", "Armory", "Missile Turret", "Bunker",
		},
		[]string{
			"SCV", "Marine", "Firebat", "Medic", "Ghost", "Vulture",
			"Siege Tank (Tank Mode)", "Goliath", "Wraith", "Dropship",
			"Science Vessel", "Battlecruiser", "Valkyrie", "Nuclear Missile",
		},
		[]string{
			"Stim Packs", "Lockdown", "EMP Shockwave", "Spider Mines", "Tank Siege Mode",
			"Irradiate", "Yamato Gun", "Cloaking Field", "Personnel Cloaking",
			"Restoration", "Optical Flare",
			"Terran Infantry Armor", "Terran Vehicle Plating", "Terran Ship Plating",
			"Terran Infantry Weapons", "Terran Vehicle Weapons", "Terran Ship Weapons",
			"U-238 Shells (Marine Range)", "Ion Thrusters (Vulture Speed)",
			"Titan Reactor (Science Vessel Energy)", "Ocular Implants (Ghost Sight)",
			"Moebius Reactor (Ghost Energy)", "Apollo Reactor (Wraith Energy)",
			"Colossus Reactor (Battle Cruiser Energy)", "Caduceus Reactor (Medic Energy)",
			"Charon Boosters (Goliath Range)",
		},
	),
	replay.Zerg: kinds(
		[]string{
			"Hatchery", "Lair", "Hive", "Extractor", "Spawning Pool", "Evolution Chamber",
			"Hydralisk Den", "Spire", "Greater Spire", "Queens Nest", "Ultralisk Cavern",
			"Defiler Mound", "Nydus Canal", "Creep Colony", "Sunken Colony", "Spore Colony",
		},
		[]string{
			"Drone", "Overlord", "Zergling", "Hydralisk", "Lurker", "Mutalisk",
			"Guardian", "Devourer", "Scourge", "Queen", "Ultralisk", "Defiler",
			"Infested Terran",
		},
		[]string{
			"Burrowing", "Spawn Broodlings", "Plague", "Consume", "Ensnare", "Lurker Aspect",
			"Zerg Carapace", "Zerg Flyer Carapace", "Zerg Melee Attacks",
			"Zerg Missile Attacks", "Zerg Flyer Attacks",
			"Ventral Sacs (Overlord Transport)", "Antennae (Overlord Sight)",
			"Pneumatized Carapace (Overlord Speed)", "Metabolic Boost (Zergling Speed)",
			"Adrenal Glands (Zergling Attack)", "Muscular Augments (Hydralisk Speed)",
			"Grooved Spines (Hydralisk Range)", "Gamete Meiosis (Queen Energy)",
			"Defiler Energy", "Chitinous Plating (Ultralisk Armor)",
			"Anabolic Synthesis (Ultralisk Speed)",
		},
	),
	replay.Protoss: kinds(
		[]string{
			"Nexus", "Pylon", "Assimilator", "Gateway", "Forge", "Photon Cannon",
			"Cybernetics Core", "Shield Battery", "Robotics Facility", "Observatory",
			"Robotics Support Bay", "Citadel of Adun", "Templar Archives", "Stargate",
			"Fleet Beacon", "Arbiter Tribunal",
		},
		[]string{
			"Probe", "Zealot", "Dragoon", "High Templar", "Dark Templar", "Shuttle",
			"Reaver", "Observer", "Scout", "Corsair", "Carrier", "Arbiter",
		},
		[]string{
			"Psionic Storm", "Hallucination", "Recall", "Stasis Field",
			"Disruption Web", "Mind Control", "Maelstrom",
			"Protoss Ground Armor", "Protoss Air Armor", "Protoss Ground Weapons",
			"Protoss Air Weapons", "Protoss Plasma Shields",
			"Singularity Charge (Dragoon Range)", "Leg Enhancement (Zealot Speed)",
			"Scarab Damage", "Reaver Capacity", "Gravitic Drive (Shuttle Speed)",
			"Sensor Array (Observer Sight)", "Gravitic Booster (Observer Speed)",
			"Khaydarin Amulet (Templar Energy)", "Apial Sensors (Scout Sight)",
			"Gravitic Thrusters (Scout Speed)", "Carrier Capacity",
			"Khaydarin Core (Arbiter Energy)", "Argus Jewel (Corsair Energy)",
			"Argus Talisman (Dark Archon Energy)",
		},
	),
}

// workerNames are the worker units, which the worker estimate counts.
var workerNames = map[string]bool{"SCV": true, "Probe": true, "Drone": true}

func kinds(structures, units, research []string) map[string]StepKind {
	m := make(map[string]StepKind, len(structures)+len(units)+len(research))
	for _, n := range structures {
		m[n] = Structure
	}
	for _, n := range units {
		m[n] = Unit
	}
	for _, n := range research {
		m[n] = Research
	}
	return m
}

// kindOf reports how a name is classified for a race, and whether it is on
// that race's allowlist at all.
func kindOf(race replay.Race, name string) (StepKind, bool) {
	k, ok := allowed[race][name]
	return k, ok
}

// AllowedNames returns the sorted allowlist for a race. It exists so tests
// outside this package can check the names against the parsing library.
func AllowedNames(race replay.Race) []string {
	names := make([]string, 0, len(allowed[race]))
	for n := range allowed[race] {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}
