package gnubgparser

import (
	"testing"
)

func TestAnalysisValuesParsing(t *testing.T) {
	// Parse a test file and check that analysis values are reasonable
	match, err := ParseSGFFile("test/charlot1-charlot2_7p_2025-11-08-2305.sgf")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	if len(match.Games) == 0 || len(match.Games[0].Moves) == 0 {
		t.Fatal("No games or moves found")
	}

	// Find first move with analysis
	var foundMove *MoveRecord
	for i := range match.Games[0].Moves {
		if match.Games[0].Moves[i].Analysis != nil && len(match.Games[0].Moves[i].Analysis.Moves) > 0 {
			foundMove = &match.Games[0].Moves[i]
			break
		}
	}

	if foundMove == nil {
		t.Fatal("No move with analysis found")
	}

	opt := foundMove.Analysis.Moves[0]

	// Test that probabilities are in valid range (0 to 1)
	if opt.Player1WinRate < 0 || opt.Player1WinRate > 1 {
		t.Errorf("Player1WinRate out of range: %f (should be 0-1)", opt.Player1WinRate)
	}
	if opt.Player1GammonRate < 0 || opt.Player1GammonRate > 1 {
		t.Errorf("Player1GammonRate out of range: %f (should be 0-1)", opt.Player1GammonRate)
	}
	if opt.Player1BackgammonRate < 0 || opt.Player1BackgammonRate > 1 {
		t.Errorf("Player1BackgammonRate out of range: %f (should be 0-1)", opt.Player1BackgammonRate)
	}
	if opt.Player2WinRate < 0 || opt.Player2WinRate > 1 {
		t.Errorf("Player2WinRate out of range: %f (should be 0-1)", opt.Player2WinRate)
	}
	if opt.Player2GammonRate < 0 || opt.Player2GammonRate > 1 {
		t.Errorf("Player2GammonRate out of range: %f (should be 0-1)", opt.Player2GammonRate)
	}

	// Test that equity is reasonable (-inf to +inf, but typically -3 to +3)
	if opt.Equity < -10 || opt.Equity > 10 {
		t.Logf("Warning: Equity seems extreme: %f", opt.Equity)
	}

	// Test that analysis depth is reasonable
	if opt.AnalysisDepth < 0 || opt.AnalysisDepth > 10 {
		t.Errorf("AnalysisDepth seems unreasonable: %d", opt.AnalysisDepth)
	}

	// Log the values for inspection
	t.Logf("First move analysis:")
	t.Logf("  Player1WinRate: %f", opt.Player1WinRate)
	t.Logf("  Player1GammonRate: %f", opt.Player1GammonRate)
	t.Logf("  Player1BackgammonRate: %f", opt.Player1BackgammonRate)
	t.Logf("  Player2WinRate: %f", opt.Player2WinRate)
	t.Logf("  Player2GammonRate: %f", opt.Player2GammonRate)
	t.Logf("  Player2BackgammonRate: %f", opt.Player2BackgammonRate)
	t.Logf("  Equity: %f", opt.Equity)
	t.Logf("  AnalysisDepth: %d", opt.AnalysisDepth)
}

func TestCubeAnalysisValuesParsing(t *testing.T) {
	// Parse a test file and check that cube analysis values are reasonable
	match, err := ParseSGFFile("test/charlot1-charlot2_7p_2025-11-08-2305.sgf")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	if len(match.Games) == 0 || len(match.Games[0].Moves) == 0 {
		t.Fatal("No games or moves found")
	}

	// Find first move with cube analysis
	var foundMove *MoveRecord
	for i := range match.Games[0].Moves {
		if match.Games[0].Moves[i].CubeAnalysis != nil {
			foundMove = &match.Games[0].Moves[i]
			break
		}
	}

	if foundMove == nil {
		t.Skip("No move with cube analysis found")
	}

	ca := foundMove.CubeAnalysis

	// Test that probabilities are in valid range (0 to 1)
	if ca.Player1WinRate < 0 || ca.Player1WinRate > 1 {
		t.Errorf("Player1WinRate out of range: %f (should be 0-1)", ca.Player1WinRate)
	}
	if ca.Player1GammonRate < 0 || ca.Player1GammonRate > 1 {
		t.Errorf("Player1GammonRate out of range: %f (should be 0-1)", ca.Player1GammonRate)
	}
	if ca.Player2WinRate < 0 || ca.Player2WinRate > 1 {
		t.Errorf("Player2WinRate out of range: %f (should be 0-1)", ca.Player2WinRate)
	}
	if ca.Player2GammonRate < 0 || ca.Player2GammonRate > 1 {
		t.Errorf("Player2GammonRate out of range: %f (should be 0-1)", ca.Player2GammonRate)
	}

	// Log the values for inspection
	t.Logf("First cube analysis:")
	t.Logf("  Player1WinRate: %f", ca.Player1WinRate)
	t.Logf("  Player1GammonRate: %f", ca.Player1GammonRate)
	t.Logf("  Player1BackgammonRate: %f", ca.Player1BackgammonRate)
	t.Logf("  Player2WinRate: %f", ca.Player2WinRate)
	t.Logf("  Player2GammonRate: %f", ca.Player2GammonRate)
	t.Logf("  Player2BackgammonRate: %f", ca.Player2BackgammonRate)
	t.Logf("  CubelessEquity: %f", ca.CubelessEquity)
}
