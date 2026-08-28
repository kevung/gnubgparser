package gnubgparser

import "testing"

// The property strings below are verbatim extracts from gnuBG 1.08.003 exports
// (test/charlot1-charlot2_7p_2025-11-08-2305.sgf and a 2025-11-08-2308 match).
//
// Layout, per gnubg/sgf.c:
//
//	WriteDoubleAnalysis()  DA[E ver <VER> <nPlies>[C] <fDeterministic> <rNoise> <fUsePrune> <2x7 floats>]
//	WriteMoveAnalysis()    A[<iMove>][<move> E ver <VER> <5 probs> <equity> <nPlies>[C] <0> <fDeterministic> <rNoise> <fUsePrune>]…
//
// so the token right after "ver <VER>" in DA[] is the ply, and "ver <VER>" itself
// is the SGF analysis-record format version — a constant 3, never a search depth.

func TestParseCubeAnalysisDepthIsNotFormatVersion(t *testing.T) {
	tests := []struct {
		name          string
		da            string
		wantDepth     int
		wantKnown     bool
		wantCubeful   bool
		wantFormatVer int
	}{
		{
			name:          "2-ply cubeful cube analysis",
			da:            "E ver 3 2C 1 0.000000 1 0.503635 0.135264 0.005951 0.140890 0.006297 0.001137 0.500296 0.503635 0.135264 0.005951 0.140890 0.006297 0.001137 0.478995",
			wantDepth:     2,
			wantKnown:     true,
			wantCubeful:   true,
			wantFormatVer: 3,
		},
		{
			name:          "0-ply cubeful cube analysis",
			da:            "E ver 3 0C 1 0.000000 1 0.485985 0.139367 0.006799 0.148849 0.006264 -0.037065 0.496719 0.485985 0.139367 0.006799 0.148849 0.006264 -0.037065 0.473077",
			wantDepth:     0,
			wantKnown:     true,
			wantCubeful:   true,
			wantFormatVer: 3,
		},
		{
			name:          "3-ply cubeless cube analysis",
			da:            "E ver 3 3 1 0.000000 1 0.493496 0.140697 0.005441 0.152764 0.006935 -0.027036 0.497992 0.493496 0.140697 0.005441 0.152764 0.006935 -0.027036 0.474958",
			wantDepth:     3,
			wantKnown:     true,
			wantCubeful:   false,
			wantFormatVer: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node := &SGFNode{Properties: map[string][]string{"DA": {tc.da}}}
			mr := &MoveRecord{}
			parseCubeAnalysis(node, mr)

			if mr.CubeAnalysis == nil {
				t.Fatal("no cube analysis parsed")
			}
			ca := mr.CubeAnalysis
			if ca.AnalysisDepth != tc.wantDepth {
				t.Errorf("AnalysisDepth = %d, want %d (must not be the SGF format version)", ca.AnalysisDepth, tc.wantDepth)
			}
			if ca.AnalysisDepthKnown != tc.wantKnown {
				t.Errorf("AnalysisDepthKnown = %v, want %v", ca.AnalysisDepthKnown, tc.wantKnown)
			}
			if ca.Cubeful != tc.wantCubeful {
				t.Errorf("Cubeful = %v, want %v", ca.Cubeful, tc.wantCubeful)
			}
			if ca.FormatVersion != tc.wantFormatVer {
				t.Errorf("FormatVersion = %d, want %d", ca.FormatVersion, tc.wantFormatVer)
			}
		})
	}
}

// A rollout cube analysis carries no evalcontext at all — gnubg writes
// "X ver 3 Eq Trials …". The depth must read as unknown, and the fields that the
// EVAL_EVAL layout would have supplied must not be invented.
func TestParseCubeAnalysisRolloutHasUnknownDepth(t *testing.T) {
	da := "X ver 3 Eq Trials 1296 NoDouble Output 0.5 0.13 0.005 0.14 0.006 0.001 0.5 StdDev 0.001 0.001 0.001 0.001 0.001 0.001 0.001 DoubleTake Output 0.5 0.13 0.005 0.14 0.006 0.001 0.48 StdDev 0.001 0.001 0.001 0.001 0.001 0.001 0.001"
	node := &SGFNode{Properties: map[string][]string{"DA": {da}}}
	mr := &MoveRecord{}
	parseCubeAnalysis(node, mr)

	if mr.CubeAnalysis == nil {
		t.Fatal("no cube analysis parsed")
	}
	if mr.CubeAnalysis.AnalysisDepthKnown {
		t.Error("AnalysisDepthKnown = true for a rollout; the DA rollout form carries no ply")
	}
	if mr.CubeAnalysis.AnalysisDepth != 0 {
		t.Errorf("AnalysisDepth = %d, want 0 alongside AnalysisDepthKnown=false", mr.CubeAnalysis.AnalysisDepth)
	}
	if mr.CubeAnalysis.EvalType != EvalTypeRollout {
		t.Errorf("EvalType = %q, want %q", mr.CubeAnalysis.EvalType, EvalTypeRollout)
	}
	if mr.CubeAnalysis.Player1WinRate != 0 {
		t.Errorf("Player1WinRate = %v; the EVAL_EVAL field layout must not be applied to a rollout record", mr.CubeAnalysis.Player1WinRate)
	}
}

// The leading value of A[…] is the index of the move actually played
// (gnubg WriteMoveAnalysis: A[%u] with pmr->n.iMove), not a ply. The ply of each
// option sits at the end of that option, in its evalcontext.
func TestParseMoveAnalysisDepthComesFromEvalContext(t *testing.T) {
	node := &SGFNode{Properties: map[string][]string{"A": {
		"1",
		"mhxw E ver 3 0.500287 0.135144 0.005422 0.130120 0.004473 0.007629 2C 0 1 0.000000 1",
		"hchg E ver 3 0.420863 0.112744 0.007688 0.197772 0.019148 -0.361306 0C 0 1 0.000000 1",
	}}}
	mr := &MoveRecord{}
	parseMoveAnalysis(node, mr)

	if mr.Analysis == nil || len(mr.Analysis.Moves) != 2 {
		t.Fatalf("want 2 move options, got %+v", mr.Analysis)
	}
	if mr.Analysis.SelectedMove != 1 {
		t.Errorf("SelectedMove = %d, want 1 (the A[n] leading value)", mr.Analysis.SelectedMove)
	}
	if mr.Analysis.FormatVersion != 3 {
		t.Errorf("FormatVersion = %d, want 3", mr.Analysis.FormatVersion)
	}
	wantDepths := []int{2, 0}
	for i, want := range wantDepths {
		opt := mr.Analysis.Moves[i]
		if opt.AnalysisDepth != want {
			t.Errorf("Moves[%d].AnalysisDepth = %d, want %d", i, opt.AnalysisDepth, want)
		}
		if !opt.AnalysisDepthKnown {
			t.Errorf("Moves[%d].AnalysisDepthKnown = false, want true", i)
		}
		if !opt.Cubeful {
			t.Errorf("Moves[%d].Cubeful = false, want true", i)
		}
	}
	// Equity must still be read from the same index as before the fix.
	if got := mr.Analysis.Moves[0].Equity; got < 0.0076 || got > 0.0077 {
		t.Errorf("Moves[0].Equity = %v, want ~0.007629", got)
	}
}

// A malformed depth token must not silently become a valid-looking 0-ply.
func TestParseCubeAnalysisRejectsNonNumericDepth(t *testing.T) {
	da := "E ver 3 xx 1 0.000000 1 0.5 0.13 0.005 0.14 0.006 0.001 0.5 0.5 0.13 0.005 0.14 0.006 0.001 0.48"
	node := &SGFNode{Properties: map[string][]string{"DA": {da}}}
	mr := &MoveRecord{}
	parseCubeAnalysis(node, mr)

	if mr.CubeAnalysis == nil {
		t.Fatal("no cube analysis parsed")
	}
	if mr.CubeAnalysis.AnalysisDepthKnown {
		t.Error("AnalysisDepthKnown = true for an unparsable depth token")
	}
}

// Real gnuBG files: no cube analysis may claim the format version as its depth.
func TestFixtureCubeDepthsAreNotTheFormatVersion(t *testing.T) {
	for _, path := range []string{
		"test/charlot1-charlot2_7p_2025-11-08-2305.sgf",
		"test/charlot1-charlot2_7p_2025-11-08-2308.sgf",
		"test/charlot1-charlot2_7p_2025-11-08-2305_analysed.sgf",
	} {
		match, err := ParseSGFFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		seen := map[int]int{}
		total := 0
		for _, g := range match.Games {
			for _, m := range g.Moves {
				if m.CubeAnalysis == nil {
					continue
				}
				total++
				seen[m.CubeAnalysis.AnalysisDepth]++
				if m.CubeAnalysis.FormatVersion != SGFAnalysisFormatVersion {
					t.Errorf("%s: FormatVersion = %d, want %d", path, m.CubeAnalysis.FormatVersion, SGFAnalysisFormatVersion)
				}
			}
		}
		if total == 0 {
			t.Fatalf("%s: no cube analysis found", path)
		}
		if seen[3] == total {
			t.Errorf("%s: all %d cube analyses report depth 3 — that is the format version, not a ply", path, total)
		}
	}
}

// A rollout move option has a different payload from the evaluation kind on
// (gnubg/sgf.c, WriteRolloutAnalysis with fIsMove=1); its numeric fields must not
// be read at the EVAL_EVAL indices, and it has no ply.
func TestParseMoveAnalysisRolloutOptionHasUnknownDepth(t *testing.T) {
	node := &SGFNode{Properties: map[string][]string{"A": {
		"0",
		"mhxw X ver 3 Score 0.007629 0.0001 Trials 1296 Output 0.5 0.13 0.005 0.14 0.006 0.001 0.5 StdDev 0.001 0.001 0.001 0.001 0.001 0.001 0.001",
	}}}
	mr := &MoveRecord{}
	parseMoveAnalysis(node, mr)

	if mr.Analysis == nil || len(mr.Analysis.Moves) != 1 {
		t.Fatalf("want 1 move option, got %+v", mr.Analysis)
	}
	opt := mr.Analysis.Moves[0]
	if opt.EvalType != EvalTypeRollout {
		t.Errorf("EvalType = %q, want %q", opt.EvalType, EvalTypeRollout)
	}
	if opt.AnalysisDepthKnown {
		t.Error("AnalysisDepthKnown = true for a rollout option")
	}
	if opt.Player1WinRate != 0 || opt.Equity != 0 {
		t.Errorf("rollout option decoded at EVAL_EVAL indices: win=%v equity=%v", opt.Player1WinRate, opt.Equity)
	}
	if opt.MoveString == "" {
		t.Error("the move itself must still be reported for a rollout option")
	}
}

func TestParsePlyToken(t *testing.T) {
	tests := []struct {
		tok         string
		wantPlies   int
		wantCubeful bool
		wantErr     bool
	}{
		{tok: "0", wantPlies: 0},
		{tok: "2", wantPlies: 2},
		{tok: "0C", wantPlies: 0, wantCubeful: true},
		{tok: "2C", wantPlies: 2, wantCubeful: true},
		{tok: "ver", wantErr: true},
		{tok: "C", wantErr: true},
		{tok: "", wantErr: true},
		{tok: "-1", wantErr: true},
		{tok: "3.0", wantErr: true},
	}
	for _, tc := range tests {
		got, err := parsePlyToken(tc.tok)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parsePlyToken(%q) = %+v, want error (a silent 0 reads as a real 0-ply)", tc.tok, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parsePlyToken(%q): %v", tc.tok, err)
			continue
		}
		if got.Plies != tc.wantPlies || got.Cubeful != tc.wantCubeful {
			t.Errorf("parsePlyToken(%q) = %+v, want plies=%d cubeful=%v", tc.tok, got, tc.wantPlies, tc.wantCubeful)
		}
	}
}

// Move-option depths come from each option's own evalcontext, so within one
// decision gnuBG's move filters leave a mix of 0-ply candidates and deeper
// survivors. A single value everywhere means the ply was not read from there.
func TestFixtureMoveDepthsVaryWithinTheFile(t *testing.T) {
	match, err := ParseSGFFile("test/charlot1-charlot2_7p_2025-11-08-2305_analysed.sgf")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int]int{}
	selected := map[int]int{}
	for _, g := range match.Games {
		for _, m := range g.Moves {
			if m.Analysis == nil {
				continue
			}
			selected[m.Analysis.SelectedMove]++
			for _, o := range m.Analysis.Moves {
				if !o.AnalysisDepthKnown {
					continue
				}
				seen[o.AnalysisDepth]++
			}
		}
	}
	if len(seen) < 2 {
		t.Errorf("move-option depths = %v, want at least two distinct plies", seen)
	}
	// gnuBG's A[n] leading value is the rank of the played move; this file has
	// decisions where the player did not play gnuBG's first choice.
	if len(selected) < 2 {
		t.Errorf("SelectedMove values = %v, want more than one (A[n] is the played move's rank)", selected)
	}
}
