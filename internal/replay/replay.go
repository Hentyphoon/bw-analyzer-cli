// Package replay defines the domain model for a parsed 1v1 replay. It imports
// no other internal package and nothing from screp or pgx.
package replay

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Replay is one parsed replay. Every player-issued command appears in
// Actions; the subset that produces something also appears in Production.
type Replay struct {
	Hash      string    `json:"hash"`      // sha256 hex of the file bytes
	FileName  string    `json:"file_name"` // base name, no directory
	MapName   string    `json:"map_name"`
	Version   string    `json:"version"`   // range reported by the library, e.g. "1.21+"
	GameType  string    `json:"game_type"` // e.g. "Melee", "Top vs Bottom"
	Frames    Frame     `json:"frames"`
	StartTime time.Time `json:"start_time"`

	// Players in team order, observers included.
	Players []Player `json:"players"`

	Actions    []Action        `json:"-"`
	Production []ProductionCmd `json:"-"`

	// ParseErrors counts commands the library could not decode. Each one may
	// hide further commands in the same frame, so it is surfaced, not ignored.
	ParseErrors int `json:"parse_errors"`
}

// Race is a playable race, stored as its letter.
type Race string

const (
	Terran  Race = "T"
	Protoss Race = "P"
	Zerg    Race = "Z"
	// RaceUnknown is anything else the replay header reports.
	RaceUnknown Race = "?"
)

// Result is a player's game outcome as inferred by the library.
type Result string

const (
	Win           Result = "win"
	Loss          Result = "loss"
	ResultUnknown Result = "unknown"
)

// Point is a map position in pixels (one tile is 32 pixels).
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Player is one participant in a replay.
type Player struct {
	ID       byte   `json:"id"` // in-game player ID, matches Action.PlayerID
	Name     string `json:"name"`
	Race     Race   `json:"race"`
	Team     int    `json:"team"`
	Human    bool   `json:"human"`
	Observer bool   `json:"observer"`
	Result   Result `json:"result"`
	// Start is nil when the map data has no start location for the player.
	Start *Point `json:"start,omitempty"`
}

// Action is one command issued by a player. Every command the library
// attributes to a player counts, matching how it computes APM.
type Action struct {
	Frame     Frame
	PlayerID  byte
	Effective bool
}

// ProductionKind says what a production command asks for.
type ProductionKind string

const (
	KindBuild         ProductionKind = "build"          // a worker places a structure
	KindTrain         ProductionKind = "train"          // a structure trains a unit
	KindUnitMorph     ProductionKind = "unit_morph"     // a larva or unit morphs (Zerg)
	KindBuildingMorph ProductionKind = "building_morph" // Lair, Hive, Sunken Colony, ...
	KindTech          ProductionKind = "tech"
	KindUpgrade       ProductionKind = "upgrade"
)

// ProductionCmd is a command that orders a structure, unit, tech, or upgrade.
// It records what was ordered, not what was completed: cancelled or failed
// orders still appear.
type ProductionCmd struct {
	Frame     Frame
	PlayerID  byte
	Kind      ProductionKind
	Name      string // unit, tech, or upgrade name as the library reports it
	Effective bool
	// Pos is set for KindBuild only.
	Pos *Point
}

// ContentHash returns the sha256 hex digest used to identify a replay file.
func ContentHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
