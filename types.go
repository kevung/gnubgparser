// Package gnubgparser provides functionality for parsing GNU Backgammon SGF files.
//
// This package parses gnuBG match files in SGF (Smart Game Format) format,
// extracting match metadata, game moves, cube decisions, and analysis data.
// It provides both in-memory data structures and JSON export capabilities.
package gnubgparser

import (
	"encoding/json"
	"time"
)

// Match represents a complete backgammon match
type Match struct {
	// Match metadata
	Metadata MatchMetadata `json:"metadata"`
	// List of games in the match
	Games []Game `json:"games"`
}

// MatchMetadata contains information about the match
type MatchMetadata struct {
	// Player names
	Player1 string `json:"player1"`
	Player2 string `json:"player2"`
	// Player ratings
	Rating1 string `json:"rating1,omitempty"`
	Rating2 string `json:"rating2,omitempty"`
	// Match details
	MatchLength int    `json:"match_length"` // Points to play to (0 for money game)
	Event       string `json:"event,omitempty"`
	Round       string `json:"round,omitempty"`
	Place       string `json:"place,omitempty"`
	Date        string `json:"date,omitempty"` // ISO format YYYY-MM-DD
	Annotator   string `json:"annotator,omitempty"`
	Comment     string `json:"comment,omitempty"`
	// SGF metadata
	Application string `json:"application,omitempty"` // e.g., "GNU Backgammon:1.06.002"
}

// Game represents a single game within a match
type Game struct {
	GameNumber   int           `json:"game_number"`
	Score        [2]int        `json:"score"`         // Score before this game [player1, player2]
	Variation    string        `json:"variation"`     // "Standard", "Nackgammon", "Hypergammon1", etc.
	Crawford     bool          `json:"crawford"`      // Is Crawford rule in effect?
	CrawfordGame bool          `json:"crawford_game"` // Is this the Crawford game?
	Jacoby       bool          `json:"jacoby"`        // Jacoby rule (money games)
	CubeEnabled  bool          `json:"cube_enabled"`  // Is cube enabled?
	AutoDoubles  int           `json:"auto_doubles"`  // Number of automatic doubles
	Winner       int           `json:"winner"`        // Winner (0=player1, 1=player2, -1=not finished)
	Points       int           `json:"points"`        // Points won
	Resigned     bool          `json:"resigned"`      // Was the game resigned?
	Moves        []MoveRecord  `json:"moves"`
	GameComment  string        `json:"comment,omitempty"`
	Statistics   GameStatistic `json:"statistics,omitempty"`
}

// MoveRecord represents a single move, cube decision, or game event
type MoveRecord struct {
	Type         MoveType      `json:"type"`
	Player       int           `json:"player"` // 0 or 1
	Dice         [2]int        `json:"dice,omitempty"`
	Move         [8]int        `json:"move,omitempty"`          // Encoded move (gnuBG format)
	MoveString   string        `json:"move_string,omitempty"`   // Human-readable move
	CubeValue    int           `json:"cube_value,omitempty"`    // For SETCUBEVAL
	CubeOwner    int           `json:"cube_owner,omitempty"`    // For SETCUBEPOS (-1=center, 0=p1, 1=p2)
	Position     *Position     `json:"position,omitempty"`      // For SETBOARD
	Analysis     *MoveAnalysis `json:"analysis,omitempty"`      // Move analysis
	CubeAnalysis *CubeAnalysis `json:"cube_analysis,omitempty"` // Cube decision analysis
	Luck         *LuckRating   `json:"luck,omitempty"`
	Skill        *SkillRating  `json:"skill,omitempty"`
	Comment      string        `json:"comment,omitempty"`
}

// MoveType represents the type of move record
type MoveType string

const (
	MoveTypeNormal     MoveType = "move"       // Normal checker move
	MoveTypeDouble     MoveType = "double"     // Cube doubled
	MoveTypeTake       MoveType = "take"       // Double taken
	MoveTypeDrop       MoveType = "drop"       // Double dropped/passed
	MoveTypeResign     MoveType = "resign"     // Resignation
	MoveTypeSetBoard   MoveType = "setboard"   // Set board position
	MoveTypeSetDice    MoveType = "setdice"    // Set dice (for positions)
	MoveTypeSetCube    MoveType = "setcube"    // Set cube value
	MoveTypeSetCubePos MoveType = "setcubepos" // Set cube owner
)

// Position represents a backgammon board position
type Position struct {
	// Board[0] is player 0's checkers, Board[1] is player 1's checkers
	// Index 0-23 are points 1-24, index 24 is the bar
	Board       [2][25]int `json:"board"`
	CubeValue   int        `json:"cube_value"`
	CubeOwner   int        `json:"cube_owner"` // -1=center, 0=player1, 1=player2
	OnRoll      int        `json:"on_roll"`    // Player to roll (0 or 1)
	Dice        [2]int     `json:"dice"`       // Current dice (0 if not rolled)
	Score       [2]int     `json:"score"`
	MatchLength int        `json:"match_length"`
	Crawford    bool       `json:"crawford"`
}

// MoveAnalysis contains analysis for a checker move
type MoveAnalysis struct {
	// Top moves evaluated
	Moves []MoveOption `json:"moves"`
	// Selected move index (in Moves array)
	SelectedMove int `json:"selected_move"`
}

// MoveOption represents one possible move with evaluation
type MoveOption struct {
	Move                  [8]int  `json:"move"`        // Encoded move
	MoveString            string  `json:"move_string"` // Human-readable
	Equity                float64 `json:"equity"`      // Equity of position after this move
	Player1WinRate        float32 `json:"player1_win_rate"`
	Player1GammonRate     float32 `json:"player1_gammon_rate"`
	Player1BackgammonRate float32 `json:"player1_backgammon_rate"`
	Player2WinRate        float32 `json:"player2_win_rate"`
	Player2GammonRate     float32 `json:"player2_gammon_rate"`
	Player2BackgammonRate float32 `json:"player2_backgammon_rate"`
	AnalysisDepth         int     `json:"analysis_depth"` // Ply depth (0=book)
}

// CubeAnalysis contains analysis for cube decisions
type CubeAnalysis struct {
	Player1WinRate        float32 `json:"player1_win_rate"`
	Player1GammonRate     float32 `json:"player1_gammon_rate"`
	Player1BackgammonRate float32 `json:"player1_backgammon_rate"`
	Player2WinRate        float32 `json:"player2_win_rate"`
	Player2GammonRate     float32 `json:"player2_gammon_rate"`
	Player2BackgammonRate float32 `json:"player2_backgammon_rate"`
	// Cube equities
	CubelessEquity    float64 `json:"cubeless_equity"`
	CubefulNoDouble   float64 `json:"cubeful_no_double"`
	CubefulDoubleTake float64 `json:"cubeful_double_take"`
	CubefulDoublePass float64 `json:"cubeful_double_pass"`
	TooGoodPoint      float64 `json:"too_good_point,omitempty"`
	// Decision analysis
	BestAction           string  `json:"best_action"` // "double", "no_double", "take", "pass"
	WrongPassTakePercent float32 `json:"wrong_pass_take_percent,omitempty"`
	AnalysisDepth        int     `json:"analysis_depth"`
}

// LuckRating represents luck analysis for a roll
type LuckRating struct {
	Rating string  `json:"rating"` // "VeryBad", "Bad", "None", "Good", "VeryGood"
	Value  float64 `json:"value"`  // Luck value (equity change due to roll)
}

// SkillRating represents skill analysis for a decision
type SkillRating struct {
	Rating string  `json:"rating"` // "VeryBad", "Bad", "Doubtful", "None"
	Error  float64 `json:"error"`  // Error in equity
}

// GameStatistic contains statistics for a game
type GameStatistic struct {
	HasMoves bool             `json:"has_moves"`
	HasCube  bool             `json:"has_cube"`
	HasDice  bool             `json:"has_dice"`
	Moves    StatisticDetail  `json:"moves,omitempty"`
	Cube     StatisticDetail  `json:"cube,omitempty"`
	Luck     [2]LuckStatistic `json:"luck,omitempty"`
}

// StatisticDetail contains error statistics
type StatisticDetail struct {
	Unforced   [2]int     `json:"unforced"` // Unforced errors by player
	Forced     [2]int     `json:"forced"`   // Forced errors
	VeryBad    [2]int     `json:"very_bad"`
	Bad        [2]int     `json:"bad"`
	Doubtful   [2]int     `json:"doubtful"`
	ErrorTotal [2]float64 `json:"error_total"` // Total error in equity
	ErrorSkill [2]float64 `json:"error_skill"` // Skill-based error
	// Cube-specific statistics
	MissedDouble [2]int `json:"missed_double,omitempty"`
	WrongDouble  [2]int `json:"wrong_double,omitempty"`
	WrongTake    [2]int `json:"wrong_take,omitempty"`
	WrongPass    [2]int `json:"wrong_pass,omitempty"`
}

// LuckStatistic contains luck statistics for a player
type LuckStatistic struct {
	VeryBad  int     `json:"very_bad"`
	Bad      int     `json:"bad"`
	None     int     `json:"none"`
	Good     int     `json:"good"`
	VeryGood int     `json:"very_good"`
	Total    float64 `json:"total"`
	TotalSq  float64 `json:"total_sq"`
}

// ToJSON converts a Match to JSON bytes
func (m *Match) ToJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// ParseTime parses SGF date format (YYYY-MM-DD)
func ParseTime(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

// FormatMove converts a gnuBG encoded move to human-readable notation
// Move encoding: pairs of from/to positions, -1 terminates
func FormatMove(move [8]int, player int) string {
	if move[0] == -1 {
		return "no move"
	}

	var result string
	for i := 0; i < 8; i += 2 {
		if move[i] == -1 {
			break
		}
		if i > 0 {
			result += " "
		}
		from := move[i]
		to := move[i+1]

		// Convert to human-readable point numbers
		// gnuBG uses 0-23 for points, 24 for bar, 25 for off
		if player == 1 {
			// For player 1, invert the board
			from = 23 - from
			to = 23 - to
		}

		fromStr := pointToString(from)
		toStr := pointToString(to)
		result += fromStr + "/" + toStr
	}
	return result
}

func pointToString(point int) string {
	switch point {
	case 24:
		return "bar"
	case 25:
		return "off"
	default:
		return string(rune('a' + point))
	}
}
