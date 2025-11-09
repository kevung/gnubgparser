package gnubgparser

import (
	"os"
	"testing"
)

func TestParseSGFFile(t *testing.T) {
	testFiles := []string{
		"test/charlot1-charlot2_7p_2025-11-08-2305.sgf",
		"test/charlot1-charlot2_7p_2025-11-08-2308.sgf",
	}

	for _, filename := range testFiles {
		t.Run(filename, func(t *testing.T) {
			// Check if file exists
			if _, err := os.Stat(filename); os.IsNotExist(err) {
				t.Skipf("Test file not found: %s", filename)
				return
			}

			match, err := ParseSGFFile(filename)
			if err != nil {
				t.Fatalf("Failed to parse file %s: %v", filename, err)
			}

			// Basic validation
			if match == nil {
				t.Fatal("Match is nil")
			}

			// Check metadata
			if match.Metadata.Player1 == "" {
				t.Error("Player1 name is empty")
			}
			if match.Metadata.Player2 == "" {
				t.Error("Player2 name is empty")
			}

			// Check games
			if len(match.Games) == 0 {
				t.Error("No games found in match")
			}

			// Validate each game
			for i, game := range match.Games {
				// Game numbers might start at 0 or 1, just check structure
				t.Logf("Game %d: Number=%d, Moves=%d, Winner=%d",
					i, game.GameNumber, len(game.Moves), game.Winner)

				// Check for moves
				if len(game.Moves) == 0 {
					t.Errorf("Game %d has no moves", i)
				}
			}

			// Try to convert to JSON
			jsonData, err := match.ToJSON()
			if err != nil {
				t.Errorf("Failed to convert to JSON: %v", err)
			}
			if len(jsonData) == 0 {
				t.Error("JSON output is empty")
			}
		})
	}
}

func TestParseMatchInfo(t *testing.T) {
	tests := []struct {
		name     string
		mi       string
		wantLen  int
		wantGame int
	}{
		{
			name:     "7 point match",
			mi:       "[length:7][game:1][ws:0][bs:0]",
			wantLen:  7,
			wantGame: 1,
		},
		{
			name:     "Money game",
			mi:       "[length:0][game:1][ws:0][bs:0]",
			wantLen:  0,
			wantGame: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := &Match{}
			game := &Game{}

			parseMatchInfo(tt.mi, match, game)

			if match.Metadata.MatchLength != tt.wantLen {
				t.Errorf("MatchLength = %d, want %d", match.Metadata.MatchLength, tt.wantLen)
			}
			if game.GameNumber != tt.wantGame {
				t.Errorf("GameNumber = %d, want %d", game.GameNumber, tt.wantGame)
			}
		})
	}
}

func TestParseRules(t *testing.T) {
	tests := []struct {
		name            string
		ru              string
		wantCrawford    bool
		wantJacoby      bool
		wantCubeEnabled bool
		wantVariation   string
	}{
		{
			name:            "Crawford game",
			ru:              "Crawford:CrawfordGame",
			wantCrawford:    true,
			wantCubeEnabled: true,
			wantVariation:   "Standard",
		},
		{
			name:            "Jacoby",
			ru:              "Jacoby",
			wantJacoby:      true,
			wantCubeEnabled: true,
			wantVariation:   "Standard",
		},
		{
			name:            "No cube",
			ru:              "NoCube",
			wantCubeEnabled: false,
			wantVariation:   "Standard",
		},
		{
			name:            "Nackgammon",
			ru:              "Nackgammon",
			wantCubeEnabled: true,
			wantVariation:   "Nackgammon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := &Game{CubeEnabled: true}
			parseRules(tt.ru, game)

			if game.Crawford != tt.wantCrawford {
				t.Errorf("Crawford = %v, want %v", game.Crawford, tt.wantCrawford)
			}
			if game.Jacoby != tt.wantJacoby {
				t.Errorf("Jacoby = %v, want %v", game.Jacoby, tt.wantJacoby)
			}
			if game.CubeEnabled != tt.wantCubeEnabled {
				t.Errorf("CubeEnabled = %v, want %v", game.CubeEnabled, tt.wantCubeEnabled)
			}
			if game.Variation != tt.wantVariation {
				t.Errorf("Variation = %s, want %s", game.Variation, tt.wantVariation)
			}
		})
	}
}

func TestParseResult(t *testing.T) {
	tests := []struct {
		name         string
		re           string
		wantWinner   int
		wantPoints   int
		wantResigned bool
	}{
		{
			name:         "White wins 2 points",
			re:           "W+2",
			wantWinner:   0,
			wantPoints:   2,
			wantResigned: false,
		},
		{
			name:         "Black resigns 1 point",
			re:           "B+1R",
			wantWinner:   1,
			wantPoints:   1,
			wantResigned: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := &Game{Winner: -1}
			parseResult(tt.re, game)

			if game.Winner != tt.wantWinner {
				t.Errorf("Winner = %d, want %d", game.Winner, tt.wantWinner)
			}
			if game.Points != tt.wantPoints {
				t.Errorf("Points = %d, want %d", game.Points, tt.wantPoints)
			}
			if game.Resigned != tt.wantResigned {
				t.Errorf("Resigned = %v, want %v", game.Resigned, tt.wantResigned)
			}
		})
	}
}

func TestDecodePoint(t *testing.T) {
	tests := []struct {
		ch   byte
		want int
	}{
		{'a', 0},
		{'b', 1},
		{'x', 23},
		{'y', 24}, // bar
		{'z', 25}, // off
	}

	for _, tt := range tests {
		t.Run(string(tt.ch), func(t *testing.T) {
			got := decodePoint(tt.ch)
			if got != tt.want {
				t.Errorf("decodePoint(%c) = %d, want %d", tt.ch, got, tt.want)
			}
		})
	}
}

func TestFormatMove(t *testing.T) {
	tests := []struct {
		name   string
		move   [8]int
		player int
		want   string
	}{
		{
			name:   "No move",
			move:   [8]int{-1, -1, -1, -1, -1, -1, -1, -1},
			player: 0,
			want:   "no move",
		},
		{
			name:   "Simple move",
			move:   [8]int{5, 3, -1, -1, -1, -1, -1, -1},
			player: 0,
			want:   "f/d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMove(tt.move, tt.player)
			if got != tt.want {
				t.Errorf("FormatMove() = %q, want %q", got, tt.want)
			}
		})
	}
}
