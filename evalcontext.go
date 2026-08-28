package gnubgparser

import (
	"fmt"
	"strconv"
	"strings"
)

// SGFAnalysisFormatVersion is the analysis-record format version gnuBG writes
// today (`SGF_FORMAT_VER` in gnubg/eval.h). It appears as the "ver <n>" prefix of
// every A[] and DA[] payload and has been 3 since 2002.
//
// It is a *file format* version. It is not, and has never been, a search depth —
// mistaking the two makes every cube analysis report a fabricated 3-ply.
const SGFAnalysisFormatVersion = 3

// Evaluation kinds gnuBG can put at the head of an analysis record
// (gnubg/sgf.c, RestoreDoubleAnalysis / RestoreMoveAnalysis).
const (
	// EvalTypeEval is a static neural-net evaluation. Only this kind carries a
	// serialised evalcontext, and therefore only this kind carries a ply count.
	EvalTypeEval = "E"
	// EvalTypeRollout is a rollout record as written since format version 2.
	EvalTypeRollout = "X"
	// EvalTypeRolloutLegacy is the pre-version-2 rollout record.
	EvalTypeRolloutLegacy = "R"
)

// evalContext is the part of gnuBG's `evalcontext` this parser reads back.
//
// gnuBG serialises it as `<nPlies>[C] …` — the ply count, immediately followed by
// a literal 'C' when the evaluation was cubeful (gnubg/sgf.c, WriteEvalContext).
type evalContext struct {
	Plies   int
	Cubeful bool
}

// parsePlyToken decodes gnuBG's `<nPlies>[C]` token.
//
// The 'C' suffix is not a separator: "2C" means 2-ply cubeful, "2" means 2-ply
// cubeless. A token that is not of that shape is an error rather than a silent 0,
// which would read as a perfectly valid 0-ply evaluation.
func parsePlyToken(tok string) (evalContext, error) {
	var ec evalContext

	digits := tok
	if strings.HasSuffix(digits, "C") {
		ec.Cubeful = true
		digits = digits[:len(digits)-1]
	}

	plies, err := strconv.Atoi(digits)
	if err != nil {
		return evalContext{}, fmt.Errorf("gnubgparser: %q is not a gnuBG ply token (<nPlies>[C]): %w", tok, err)
	}
	if plies < 0 {
		return evalContext{}, fmt.Errorf("gnubgparser: negative ply count %d in token %q", plies, tok)
	}

	ec.Plies = plies
	return ec, nil
}

// parseFormatVersion reads the "ver <n>" prefix that follows the evaluation kind
// in every analysis payload. `parts` must start at the "ver" token.
func parseFormatVersion(parts []string) (int, error) {
	if len(parts) < 2 || parts[0] != "ver" {
		return 0, fmt.Errorf("gnubgparser: analysis record has no \"ver <n>\" prefix")
	}
	ver, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("gnubgparser: unreadable analysis format version %q: %w", parts[1], err)
	}
	return ver, nil
}
