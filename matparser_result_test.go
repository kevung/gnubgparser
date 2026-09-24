package gnubgparser

import (
	"strings"
	"testing"
)

// The fixtures below were written by gnubg itself (1.08.003, `export match mat`);
// the .gnubg script next to each one regenerates it with `gnubg -t -q < script`.

// TestParseMATWinsLineColumn: gnubg indents a "Wins" line standing on its own in
// the WINNER's column. Here player 2 rolls 51 and resigns: the last cell of the
// last numbered line is player 2's, so "the last player who acted" names the
// loser, and only the column names the winner.
func TestParseMATWinsLineColumn(t *testing.T) {
	match, err := ParseMATFile("test/gnubg_roll_then_resign_1p.mat")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(match.Games) != 1 {
		t.Fatalf("games = %d, want 1", len(match.Games))
	}
	g := match.Games[0]
	if g.Winner != 0 {
		t.Errorf("Winner = %d, want 0 (\"      Wins\" sits in player 1's column)", g.Winner)
	}
	if g.Points != 1 {
		t.Errorf("Points = %d, want 1", g.Points)
	}
}

// TestParseMATDropScoresTheCube: a refused double wins the cube's value, whether
// gnubg writes the "Wins" on the same line (the dropper is in the left column) or
// on the next one (the dropper is in the right column).
func TestParseMATDropScoresTheCube(t *testing.T) {
	match, err := ParseMATFile("test/gnubg_selfplay_drops_7p.mat")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cases := []struct {
		game, winner, points int
	}{
		{0, 0, 1}, // "4) Doubles => 2   Drops" then "      Wins 1 point"
		{1, 1, 1}, // "8) Drops           Wins 1 point"
		{3, 1, 2}, // "22) Drops          Wins 2 points" after a take at 2
	}
	for _, c := range cases {
		g := match.Games[c.game]
		if g.Winner != c.winner || g.Points != c.points {
			t.Errorf("game %d: winner %d, points %d; want winner %d, points %d",
				g.GameNumber, g.Winner, g.Points, c.winner, c.points)
		}
	}
}

// TestParseMATResultsChainIntoScores holds every game of every gnubg-written file
// to the score line of the game after it: the points read must be the points the
// match counted, credited to the player it credited them to.
func TestParseMATResultsChainIntoScores(t *testing.T) {
	for _, file := range []string{
		"test/gnubg_selfplay_drops_7p.mat",
		"test/gnubg_roll_then_resign_1p.mat",
		"test/charlot1-charlot2_7p_2025-11-08-2305.mat",
	} {
		match, err := ParseMATFile(file)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		for i, g := range match.Games {
			if g.Winner != 0 && g.Winner != 1 {
				t.Errorf("%s game %d: no winner", file, g.GameNumber)
				continue
			}
			if g.Points <= 0 {
				t.Errorf("%s game %d: %d points", file, g.GameNumber, g.Points)
			}
			want := g.Score
			want[g.Winner] += g.Points
			var got [2]int
			if i+1 < len(match.Games) {
				got = match.Games[i+1].Score
			} else {
				// The last game ends the match: its winner reaches the length.
				got = want
				if want[g.Winner] < match.Metadata.MatchLength {
					t.Errorf("%s last game: winner ends on %d, match is to %d",
						file, want[g.Winner], match.Metadata.MatchLength)
				}
			}
			if got != want {
				t.Errorf("%s game %d: %v + %d to player %d = %v, next game starts at %v",
					file, g.GameNumber, g.Score, g.Points, g.Winner, want, got)
			}
		}
	}
}

// TestParseMATUnrecordedMove: a "???" cell (XG writes it when a player rolled and
// resigned) is a play the record does not carry — not a dance, which is a cell
// holding the dice alone. Both decode to an all -1 Move; Unrecorded tells them apart.
func TestParseMATUnrecordedMove(t *testing.T) {
	matContent := ` 3 point match

 Game 1
 Player1 : 0                   Player2 : 0
  1) 31: 8/5 6/5                 64:
  2) 21: ???                     
                                  Wins 1 point
`
	match, err := ParseMAT(strings.NewReader(matContent))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	moves := match.Games[0].Moves
	if len(moves) != 3 {
		t.Fatalf("moves = %d, want 3", len(moves))
	}
	if moves[0].Unrecorded {
		t.Error("a played move is not unrecorded")
	}
	if moves[1].Unrecorded {
		t.Error("a dance (dice alone) is not unrecorded")
	}
	if !moves[2].Unrecorded {
		t.Error("a ??? cell is unrecorded")
	}
	if moves[2].MoveString != "???" {
		t.Errorf("MoveString = %q, want the raw \"???\" kept", moves[2].MoveString)
	}
	if moves[2].Move != [8]int{-1, -1, -1, -1, -1, -1, -1, -1} {
		t.Errorf("Move = %v, want all -1", moves[2].Move)
	}
	if g := match.Games[0]; g.Winner != 1 || g.Points != 1 {
		t.Errorf("winner %d, points %d; want 1, 1", g.Winner, g.Points)
	}
}
