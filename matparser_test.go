package gnubgparser

import (
	"strings"
	"testing"
)

func TestParseMATFile(t *testing.T) {
	// Test parsing the actual MAT file
	match, err := ParseMATFile("test/charlot1-charlot2_7p_2025-11-08-2305.mat")
	if err != nil {
		t.Fatalf("Failed to parse MAT file: %v", err)
	}

	// Verify match metadata
	if match.Metadata.MatchLength != 7 {
		t.Errorf("Expected match length 7, got %d", match.Metadata.MatchLength)
	}

	if match.Metadata.Date != "2025-11-08" {
		t.Errorf("Expected date 2025-11-08, got %s", match.Metadata.Date)
	}

	// Verify we have games
	if len(match.Games) == 0 {
		t.Fatal("No games parsed")
	}

	t.Logf("Parsed %d games", len(match.Games))

	// Verify first game
	game1 := match.Games[0]
	if game1.GameNumber != 1 {
		t.Errorf("Expected game number 1, got %d", game1.GameNumber)
	}

	if game1.Score[0] != 0 || game1.Score[1] != 0 {
		t.Errorf("Expected score [0, 0], got %v", game1.Score)
	}

	// Check that moves were parsed
	if len(game1.Moves) == 0 {
		t.Error("No moves parsed for game 1")
	}

	t.Logf("Game 1 has %d moves", len(game1.Moves))

	// Find a double and check it was parsed correctly
	foundDouble := false
	for _, move := range game1.Moves {
		if move.Type == MoveTypeDouble {
			foundDouble = true
			if move.CubeValue != 2 {
				t.Errorf("Expected cube value 2 for first double, got %d", move.CubeValue)
			}
			break
		}
	}

	if !foundDouble {
		t.Error("No double found in game 1, but match shows doubles")
	}

	// Check game winner
	if game1.Winner == -1 {
		t.Error("Game 1 should have a winner")
	}

	if game1.Points != 2 {
		t.Errorf("Expected game 1 to be worth 2 points, got %d", game1.Points)
	}
}

func TestParseMATBasic(t *testing.T) {
	matContent := `; [EventDate "2025.11.08"]

 7 point match

 Game 1
 Player1 : 0                   Player2 : 0
  1)                             41: 13/9 24/23 
  2) 31: 6/5 8/5                 41: 6/5 9/5 
  3) 31: 24/21 6/5               65: 24/18 23/18 
  4)  Doubles => 2                Takes
  5) 64: 13/7 7/3                55: 22/17 8/3 8/3 6/1
                                  Wins 2 points

 Game 2
 Player1 : 0                   Player2 : 2
  1)                             65: 24/18 18/13 
  2) 32: 24/21 13/11             64: 24/20 20/14
  3)  Doubles => 2                Drops
      Wins 2 points
`

	match, err := ParseMAT(strings.NewReader(matContent))
	if err != nil {
		t.Fatalf("Failed to parse MAT: %v", err)
	}

	// Check match length
	if match.Metadata.MatchLength != 7 {
		t.Errorf("Expected match length 7, got %d", match.Metadata.MatchLength)
	}

	// Check date
	if match.Metadata.Date != "2025-11-08" {
		t.Errorf("Expected date 2025-11-08, got %s", match.Metadata.Date)
	}

	// Check number of games
	if len(match.Games) != 2 {
		t.Fatalf("Expected 2 games, got %d", len(match.Games))
	}

	// Check game 1
	game1 := match.Games[0]
	if game1.GameNumber != 1 {
		t.Errorf("Expected game number 1, got %d", game1.GameNumber)
	}
	if game1.Score[0] != 0 || game1.Score[1] != 0 {
		t.Errorf("Expected score [0, 0], got %v", game1.Score)
	}
	if game1.Winner == -1 {
		t.Error("Game 1 should have a winner")
	}
	if game1.Points != 2 {
		t.Errorf("Expected 2 points, got %d", game1.Points)
	}

	// Check that game 1 has a double and a take
	hasDouble := false
	hasTake := false
	for _, move := range game1.Moves {
		if move.Type == MoveTypeDouble {
			hasDouble = true
			if move.CubeValue != 2 {
				t.Errorf("Expected cube value 2, got %d", move.CubeValue)
			}
		}
		if move.Type == MoveTypeTake {
			hasTake = true
		}
	}
	if !hasDouble {
		t.Error("Game 1 should have a double")
	}
	if !hasTake {
		t.Error("Game 1 should have a take")
	}

	// Check game 2
	game2 := match.Games[1]
	if game2.GameNumber != 2 {
		t.Errorf("Expected game number 2, got %d", game2.GameNumber)
	}
	if game2.Score[0] != 0 || game2.Score[1] != 2 {
		t.Errorf("Expected score [0, 2], got %v", game2.Score)
	}

	// Check that game 2 has a double and a drop
	hasDouble = false
	hasDrop := false
	for _, move := range game2.Moves {
		if move.Type == MoveTypeDouble {
			hasDouble = true
		}
		if move.Type == MoveTypeDrop {
			hasDrop = true
		}
	}
	if !hasDouble {
		t.Error("Game 2 should have a double")
	}
	if !hasDrop {
		t.Error("Game 2 should have a drop")
	}
}

func TestParseMatMove(t *testing.T) {
	tests := []struct {
		input    string
		expected [8]int
	}{
		{
			input:    "6/5 8/5",
			expected: [8]int{5, 4, 7, 4, -1, -1, -1, -1}, // 6->5 is 5->4, 8->5 is 7->4
		},
		{
			input:    "24/18 23/18",
			expected: [8]int{23, 17, 22, 17, -1, -1, -1, -1},
		},
		{
			input:    "bar/23",
			expected: [8]int{24, 22, -1, -1, -1, -1, -1, -1},
		},
		{
			input:    "6/off",
			expected: [8]int{5, -1, -1, -1, -1, -1, -1, -1},
		},
		{
			input:    "",
			expected: [8]int{-1, -1, -1, -1, -1, -1, -1, -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseMatMove(tt.input)
			if result != tt.expected {
				t.Errorf("parseMatMove(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseMatPoint(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"1", 0},
		{"24", 23},
		{"bar", 24},
		{"Bar", 24},
		{"off", -1},
		{"Off", -1},
		{"13*", 12}, // Hit marker should be stripped
		{"invalid", -2},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseMatPoint(tt.input)
			if result != tt.expected {
				t.Errorf("parseMatPoint(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSplitMoveLine(t *testing.T) {
	tests := []struct {
		input    string
		expected [2]string
	}{
		{
			input:    " 31: 6/5 8/5                 41: 6/5 9/5",
			expected: [2]string{"31: 6/5 8/5", "41: 6/5 9/5"},
		},
		{
			input:    "                             41: 13/9 24/23",
			expected: [2]string{"", "41: 13/9 24/23"},
		},
		{
			input:    " Doubles => 2                Takes",
			expected: [2]string{"Doubles => 2", "Takes"},
		},
		{
			input:    " 31: 6/5 8/5",
			expected: [2]string{"31: 6/5 8/5", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitMoveLine(tt.input)
			if result[0] != tt.expected[0] || result[1] != tt.expected[1] {
				t.Errorf("splitMoveLine(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
