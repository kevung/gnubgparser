package gnubgparser

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// convertNodesToMatch converts parsed SGF nodes into a Match structure
func convertNodesToMatch(nodes []*SGFNode) (*Match, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no games found")
	}

	match := &Match{
		Games: make([]Game, 0),
	}

	// Process each game tree
	for _, gameNode := range nodes {
		game, err := convertGame(gameNode, match)
		if err != nil {
			return nil, err
		}
		match.Games = append(match.Games, *game)
	}

	return match, nil
}

// convertGame converts an SGF game tree to a Game structure
func convertGame(root *SGFNode, match *Match) (*Game, error) {
	game := &Game{
		Moves:       make([]MoveRecord, 0),
		CubeEnabled: true,
	}

	// Extract match/game metadata from root node
	if err := extractMetadata(root, match, game); err != nil {
		return nil, err
	}

	// Process the game tree (sequence of nodes)
	current := root
	for current != nil {
		if err := processNode(current, game); err != nil {
			return nil, err
		}

		// Move to next node in sequence
		if len(current.Children) > 0 {
			current = current.Children[0]
		} else {
			current = nil
		}
	}

	return game, nil
}

// extractMetadata extracts metadata from the root node
func extractMetadata(node *SGFNode, match *Match, game *Game) error {
	// SGF format info
	if ap := getProperty(node, "AP"); ap != "" {
		match.Metadata.Application = ap
	}

	// Player names
	if pw := getProperty(node, "PW"); pw != "" {
		match.Metadata.Player1 = pw
	}
	if pb := getProperty(node, "PB"); pb != "" {
		match.Metadata.Player2 = pb
	}

	// Player ratings
	if wr := getProperty(node, "WR"); wr != "" {
		match.Metadata.Rating1 = wr
	}
	if br := getProperty(node, "BR"); br != "" {
		match.Metadata.Rating2 = br
	}

	// Event information
	if ev := getProperty(node, "EV"); ev != "" {
		match.Metadata.Event = ev
	}
	if ro := getProperty(node, "RO"); ro != "" {
		match.Metadata.Round = ro
	}
	if pc := getProperty(node, "PC"); pc != "" {
		match.Metadata.Place = pc
	}
	if dt := getProperty(node, "DT"); dt != "" {
		match.Metadata.Date = dt
	}
	if an := getProperty(node, "AN"); an != "" {
		match.Metadata.Annotator = an
	}
	if gc := getProperty(node, "GC"); gc != "" {
		match.Metadata.Comment = gc
	}

	// Match info (MI property)
	// MI is stored as multiple values: MI[length:7][game:0][ws:0][bs:0]
	// Need to join all values since getProperty only returns the first one
	if values, ok := node.Properties["MI"]; ok && len(values) > 0 {
		mi := strings.Join(values, "][")
		parseMatchInfo(mi, match, game)
	}

	// Rules
	if ru := getProperty(node, "RU"); ru != "" {
		parseRules(ru, game)
	}

	// Cube value
	if cv := getProperty(node, "CV"); cv != "" {
		game.AutoDoubles = getPropertyInt(node, "CV")
	}

	// Result
	if re := getProperty(node, "RE"); re != "" {
		parseResult(re, game)
	}

	return nil
}

// parseMatchInfo parses the MI (match info) property
// Format: MI[length:7][game:1][ws:0][bs:0]
func parseMatchInfo(mi string, match *Match, game *Game) {
	parts := strings.Split(mi, "][")
	for _, part := range parts {
		part = strings.Trim(part, "[]")
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}

		key := kv[0]
		value := kv[1]

		switch key {
		case "length":
			if v, err := strconv.Atoi(value); err == nil {
				match.Metadata.MatchLength = v
			}
		case "game":
			if v, err := strconv.Atoi(value); err == nil {
				game.GameNumber = v
			}
		case "ws":
			if v, err := strconv.Atoi(value); err == nil {
				game.Score[0] = v
			}
		case "bs":
			if v, err := strconv.Atoi(value); err == nil {
				game.Score[1] = v
			}
		}
	}
}

// parseRules parses the RU (rules) property
// Format: RU[Crawford:CrawfordGame:Jacoby:Nackgammon]
func parseRules(ru string, game *Game) {
	rules := strings.Split(ru, ":")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		switch rule {
		case "Crawford":
			game.Crawford = true
		case "CrawfordGame":
			game.CrawfordGame = true
		case "Jacoby":
			game.Jacoby = true
		case "NoCube":
			game.CubeEnabled = false
		case "Nackgammon":
			game.Variation = "Nackgammon"
		case "Hypergammon1":
			game.Variation = "Hypergammon1"
		case "Hypergammon2":
			game.Variation = "Hypergammon2"
		case "Hypergammon3":
			game.Variation = "Hypergammon3"
		}
	}

	if game.Variation == "" {
		game.Variation = "Standard"
	}
}

// parseResult parses the RE (result) property
// Format: RE[W+2] or RE[B+1R] (R means resigned)
func parseResult(re string, game *Game) {
	if len(re) < 3 {
		return
	}

	// Winner
	if re[0] == 'W' {
		game.Winner = 0
	} else if re[0] == 'B' {
		game.Winner = 1
	}

	// Points
	pointsStr := strings.TrimLeft(re[1:], "+")
	pointsStr = strings.TrimSuffix(pointsStr, "R")
	if points, err := strconv.Atoi(pointsStr); err == nil {
		game.Points = points
	}

	// Resigned?
	if strings.HasSuffix(re, "R") {
		game.Resigned = true
	}
}

// processNode processes a single SGF node
func processNode(node *SGFNode, game *Game) error {
	// Check for comment
	comment := getProperty(node, "C")

	// Check for move (B or W property)
	if bMove := getProperty(node, "B"); bMove != "" {
		return processMove(node, game, 1, bMove, comment)
	}
	if wMove := getProperty(node, "W"); wMove != "" {
		return processMove(node, game, 0, wMove, comment)
	}

	// Check for set board (AE, AW, AB properties)
	if hasProperty(node, "AE") || hasProperty(node, "AW") || hasProperty(node, "AB") {
		return processSetBoard(node, game, comment)
	}

	// Check for set cube value
	if hasProperty(node, "CV") {
		mr := MoveRecord{
			Type:      MoveTypeSetCube,
			CubeValue: getPropertyInt(node, "CV"),
			Comment:   comment,
		}
		game.Moves = append(game.Moves, mr)
	}

	// Check for set cube position
	if cp := getProperty(node, "CP"); cp != "" {
		mr := MoveRecord{
			Type:    MoveTypeSetCubePos,
			Comment: comment,
		}
		switch cp {
		case "c":
			mr.CubeOwner = -1
		case "w":
			mr.CubeOwner = 0
		case "b":
			mr.CubeOwner = 1
		}
		game.Moves = append(game.Moves, mr)
	}

	// Check for set dice (DI property)
	if di := getProperty(node, "DI"); di != "" && len(di) >= 2 {
		mr := MoveRecord{
			Type:    MoveTypeSetDice,
			Comment: comment,
		}
		mr.Dice[0], _ = strconv.Atoi(string(di[0]))
		mr.Dice[1], _ = strconv.Atoi(string(di[1]))

		// Parse the luck BEFORE appending: append copies the record, so a
		// value written afterwards would land on a local copy nobody reads.
		if hasProperty(node, "LU") {
			parseLuck(node, &mr)
		}

		game.Moves = append(game.Moves, mr)
	}

	// Check for player on roll (PL property)
	// This is informational, don't create a move record

	return nil
}

// processMove processes a move (B or W property)
// Format: B[52lpab] - dice 52, move encoded as lpab
func processMove(node *SGFNode, game *Game, player int, moveStr string, comment string) error {
	mr := MoveRecord{
		Player:  player,
		Comment: comment,
	}

	// Parse move string
	if moveStr == "double" {
		mr.Type = MoveTypeDouble
	} else if moveStr == "take" {
		mr.Type = MoveTypeTake
	} else if moveStr == "drop" || moveStr == "pass" {
		mr.Type = MoveTypeDrop
	} else {
		// Normal move: dice + encoded move
		mr.Type = MoveTypeNormal

		if len(moveStr) >= 2 {
			mr.Dice[0], _ = strconv.Atoi(string(moveStr[0]))
			mr.Dice[1], _ = strconv.Atoi(string(moveStr[1]))

			// Parse encoded move
			if len(moveStr) > 2 {
				parseEncodedMove(moveStr[2:], &mr)
			} else {
				// No encoded move after dice: player cannot move
				// Initialize Move to all -1 (same convention as XG/MAT)
				for i := range mr.Move {
					mr.Move[i] = -1
				}
				mr.MoveString = "Cannot Move"
			}
		}
	}

	// Parse analysis (A property)
	if hasProperty(node, "A") {
		parseMoveAnalysis(node, &mr)
	}

	// Parse double analysis (DA property)
	if hasProperty(node, "DA") {
		parseCubeAnalysis(node, &mr)
	}

	// Parse luck (LU property)
	if hasProperty(node, "LU") {
		parseLuck(node, &mr)
	}

	// Parse skill (SK property)
	if hasProperty(node, "SK") {
		parseSkill(node, &mr)
	}

	// For "Cannot Move" positions: if we have cube analysis (DA) but no checker
	// analysis (A), synthesize a single-entry checker analysis from the DA data.
	// This matches the XG behavior where "Cannot Move" positions display
	// evaluation data (win rates, equity) in the checker analysis.
	if mr.MoveString == "Cannot Move" && mr.Analysis == nil && mr.CubeAnalysis != nil {
		mr.Analysis = &MoveAnalysis{
			Moves: []MoveOption{
				{
					Move:                  [8]int{-1, -1, -1, -1, -1, -1, -1, -1},
					MoveString:            "Cannot Move",
					Equity:                mr.CubeAnalysis.CubelessEquity,
					Player1WinRate:        mr.CubeAnalysis.Player1WinRate,
					Player1GammonRate:     mr.CubeAnalysis.Player1GammonRate,
					Player1BackgammonRate: mr.CubeAnalysis.Player1BackgammonRate,
					Player2WinRate:        mr.CubeAnalysis.Player2WinRate,
					Player2GammonRate:     mr.CubeAnalysis.Player2GammonRate,
					Player2BackgammonRate: mr.CubeAnalysis.Player2BackgammonRate,
					AnalysisDepth:         mr.CubeAnalysis.AnalysisDepth,
					AnalysisDepthKnown:    mr.CubeAnalysis.AnalysisDepthKnown,
					Cubeful:               mr.CubeAnalysis.Cubeful,
					EvalType:              mr.CubeAnalysis.EvalType,
				},
			},
			SelectedMove:  0,
			FormatVersion: mr.CubeAnalysis.FormatVersion,
		}
	}

	game.Moves = append(game.Moves, mr)
	return nil
}

// parseEncodedMove parses gnuBG's encoded move format
// Format: sequences of 2 letters representing from/to points
// a-x represent points 1-24, y is bar (25), z is off (26)
func parseEncodedMove(encoded string, mr *MoveRecord) {
	moveIdx := 0
	for i := 0; i+1 < len(encoded) && moveIdx < 8; i += 2 {
		from := decodePoint(encoded[i])
		to := decodePoint(encoded[i+1])

		mr.Move[moveIdx] = from
		mr.Move[moveIdx+1] = to
		moveIdx += 2
	}

	// Terminate with -1
	if moveIdx < 8 {
		mr.Move[moveIdx] = -1
	}

	// Generate human-readable string
	mr.MoveString = FormatMove(mr.Move, mr.Player)
}

// decodePoint converts SGF point encoding to internal representation
func decodePoint(ch byte) int {
	if ch >= 'a' && ch <= 'x' {
		return int(ch - 'a')
	}
	if ch == 'y' {
		return 24 // bar
	}
	if ch == 'z' {
		return 25 // off
	}
	return -1
}

// processSetBoard processes board setup (AE, AW, AB properties)
func processSetBoard(node *SGFNode, game *Game, comment string) error {
	pos := &Position{
		Board: [2][25]int{},
	}

	// AE clears points (usually [a:y] to clear all)
	// AW sets white checkers
	if aw := node.Properties["AW"]; len(aw) > 0 {
		for _, point := range aw {
			if len(point) == 1 {
				pt := decodePoint(point[0])
				if pt >= 0 && pt < 25 {
					pos.Board[0][pt]++
				}
			}
		}
	}

	// AB sets black checkers
	if ab := node.Properties["AB"]; len(ab) > 0 {
		for _, point := range ab {
			if len(point) == 1 {
				pt := decodePoint(point[0])
				if pt >= 0 && pt < 25 {
					pos.Board[1][pt]++
				}
			}
		}
	}

	// Player on roll
	if pl := getProperty(node, "PL"); pl != "" {
		if pl == "W" || pl == "w" {
			pos.OnRoll = 0
		} else {
			pos.OnRoll = 1
		}
	}

	mr := MoveRecord{
		Type:     MoveTypeSetBoard,
		Position: pos,
		Comment:  comment,
	}

	game.Moves = append(game.Moves, mr)
	return nil
}

// parseMoveAnalysis parses move analysis (A property).
//
// gnuBG writes it in WriteMoveAnalysis (gnubg/sgf.c) as one leading value followed
// by one value per evaluated move:
//
//	A[<iMove>][<move> E ver <VER> <5 probs> <equity> <nPlies>[C] <0> <fDeterministic> <rNoise> <fUsePrune>]…
//
// The leading <iMove> is `pmr->n.iMove`: the rank, in the evaluated move list, of
// the move actually played. It is NOT a ply — it reaches 11 in real files — and it
// is reported as MoveAnalysis.SelectedMove.
//
// Each option's ply sits at the end of that option, in its evalcontext, at index
// [10]; the <0> at [11] is a reduced-evaluation flag gnuBG still writes and
// discards. Options evaluated at different depths in the same decision are normal:
// move filters evaluate the candidate list at 0-ply and the survivors deeper.
//
// Field indices per option (0-based):
//
//	[0]  = encoded move
//	[1]  = evaluation kind: "E" static eval, "X"/"R" rollout
//	[2]  = literal "ver"
//	[3]  = SGF analysis-record format version — NOT a ply
//	[4]  = OUTPUT_WIN               (player's total win probability)
//	[5]  = OUTPUT_WINGAMMON         (player wins gammon)
//	[6]  = OUTPUT_WINBACKGAMMON     (player wins backgammon)
//	[7]  = OUTPUT_LOSEGAMMON        (opponent wins gammon)
//	[8]  = OUTPUT_LOSEBACKGAMMON    (opponent wins backgammon)
//	[9]  = rScore (equity)
//	[10] = evalcontext: ply count, with a trailing 'C' when cubeful
func parseMoveAnalysis(node *SGFNode, mr *MoveRecord) {
	analysisStrs := node.Properties["A"]
	if len(analysisStrs) == 0 {
		return
	}

	mr.Analysis = &MoveAnalysis{
		Moves: make([]MoveOption, 0),
	}

	// The first value is the index of the move actually played, not a ply.
	if selected, err := strconv.Atoi(strings.TrimSpace(analysisStrs[0])); err == nil {
		mr.Analysis.SelectedMove = selected
		analysisStrs = analysisStrs[1:]
	}

	// Parse each move option
	for _, aStr := range analysisStrs {
		parts := strings.Fields(aStr)
		if len(parts) < 10 {
			continue
		}

		opt := MoveOption{EvalType: parts[1]}

		// Move encoding at parts[0]
		if len(parts[0]) >= 2 {
			parseEncodedMoveOption(parts[0], &opt)
		}

		if ver, err := parseFormatVersion(parts[2:]); err == nil {
			mr.Analysis.FormatVersion = ver
		}

		// A rollout option has a different payload from index [1] on; keep the move
		// and say the rest is unknown rather than reading "Trials" as a probability.
		if opt.EvalType != EvalTypeEval {
			mr.Analysis.Moves = append(mr.Analysis.Moves, opt)
			continue
		}

		opt.Player1WinRate, _ = parseFloat32(parts[4])        // OUTPUT_WIN
		opt.Player1GammonRate, _ = parseFloat32(parts[5])     // OUTPUT_WINGAMMON
		opt.Player1BackgammonRate, _ = parseFloat32(parts[6]) // OUTPUT_WINBACKGAMMON
		opt.Player2GammonRate, _ = parseFloat32(parts[7])     // OUTPUT_LOSEGAMMON
		opt.Player2BackgammonRate, _ = parseFloat32(parts[8]) // OUTPUT_LOSEBACKGAMMON
		opt.Equity, _ = strconv.ParseFloat(parts[9], 64)      // rScore (equity)

		// Player2 win rate is calculated as 1.0 - Player1 win rate
		opt.Player2WinRate = 1.0 - opt.Player1WinRate

		// The ply of this option, from its own evalcontext.
		if len(parts) > 10 {
			if ec, err := parsePlyToken(parts[10]); err == nil {
				opt.AnalysisDepth = ec.Plies
				opt.Cubeful = ec.Cubeful
				opt.AnalysisDepthKnown = true
			}
		}

		mr.Analysis.Moves = append(mr.Analysis.Moves, opt)
	}
}

// parseEncodedMoveOption parses move encoding for analysis
func parseEncodedMoveOption(encoded string, opt *MoveOption) {
	moveIdx := 0
	for i := 0; i+1 < len(encoded) && moveIdx < 8; i += 2 {
		from := decodePoint(encoded[i])
		to := decodePoint(encoded[i+1])

		opt.Move[moveIdx] = from
		opt.Move[moveIdx+1] = to
		moveIdx += 2
	}

	if moveIdx < 8 {
		opt.Move[moveIdx] = -1
	}

	opt.MoveString = FormatMove(opt.Move, 0) // Player doesn't matter for display
}

// parseCubeAnalysis parses cube decision analysis (DA property).
//
// gnuBG writes the property in WriteDoubleAnalysis (gnubg/sgf.c). For a static
// evaluation (EVAL_EVAL) the payload is:
//
//	DA[E ver <VER> <nPlies>[C] <fDeterministic> <rNoise> <fUsePrune> <aarOutput[2][7]>]
//
// Field indices (0-based, after splitting on whitespace, for VER >= 3):
//
//	[0]  = evaluation kind: "E" static eval, "X"/"R" rollout
//	[1]  = literal "ver"
//	[2]  = SGF analysis-record format version — NOT a ply (see SGFAnalysisFormatVersion)
//	[3]  = evalcontext: ply count, with a trailing 'C' when cubeful ("2C" = 2-ply cubeful)
//	[4]  = evalcontext: fDeterministic
//	[5]  = evalcontext: rNoise
//	[6]  = evalcontext: fUsePrune
//	[7]  = P(player wins)                  ] no-double branch,
//	[8]  = P(player wins gammon)           ] gnuBG's aarOutput[0][0..6]
//	[9]  = P(player wins backgammon)       ]
//	[10] = P(opponent wins gammon)         ]
//	[11] = P(opponent wins backgammon)     ]
//	[12] = cubeless equity (EMG)           ]
//	[13] = cubeful no-double equity        ]
//	[14-20] = the same seven fields for the double/take branch (aarOutput[1])
//
// Format version 2 inserts one extra token (a reduced-evaluation flag, dropped in
// version 3) between the ply and fDeterministic, which shifts the floats by one.
//
// A rollout cube analysis has an entirely different payload
// ("X ver <VER> Eq Trials <n> NoDouble Output …"), carries no evalcontext, and is
// therefore reported with AnalysisDepthKnown = false and no probabilities rather
// than being force-fitted into the layout above.
//
// Note: In match play, cubeful equities are Match Winning Chances (MWC, 0.0-1.0).
// Double/Pass equity is not stored in the DA property; it equals 1.0 for money games,
// or must be computed from the match equity table for match play.
//
// Probability fields follow GNUbg's eval.h output order:
//
//	OUTPUT_WIN=0, OUTPUT_WINGAMMON=1, OUTPUT_WINBACKGAMMON=2,
//	OUTPUT_LOSEGAMMON=3, OUTPUT_LOSEBACKGAMMON=4
func parseCubeAnalysis(node *SGFNode, mr *MoveRecord) {
	daStrs := node.Properties["DA"]
	if len(daStrs) == 0 {
		return
	}

	parts := strings.Fields(daStrs[0])
	if len(parts) < 3 {
		return
	}

	ca := &CubeAnalysis{EvalType: parts[0]}

	// "ver <n>" is the record format version, never a search depth.
	ver, err := parseFormatVersion(parts[1:])
	if err != nil {
		// Without a readable version the field layout is unknown; do not guess.
		mr.CubeAnalysis = ca
		return
	}
	ca.FormatVersion = ver

	// Only a static evaluation carries an evalcontext, and only versions 2 and 3
	// have a layout this parser knows. Anything else keeps EvalType/FormatVersion
	// and an explicitly unknown depth.
	if ca.EvalType != EvalTypeEval || (ver != 2 && ver != SGFAnalysisFormatVersion) {
		mr.CubeAnalysis = ca
		return
	}

	// First float of the no-double branch. Version 2 carries one extra
	// (reduced-evaluation) token inside the evalcontext.
	base := 7
	if ver == 2 {
		base = 8
	}
	if len(parts) < base+6 {
		mr.CubeAnalysis = ca
		return
	}

	if ec, err := parsePlyToken(parts[3]); err == nil {
		ca.AnalysisDepth = ec.Plies
		ca.Cubeful = ec.Cubeful
		ca.AnalysisDepthKnown = true
	}

	pWin, _ := parseFloat32(parts[base])
	pWinGammon, _ := parseFloat32(parts[base+1])
	pWinBG, _ := parseFloat32(parts[base+2])
	pLoseGammon, _ := parseFloat32(parts[base+3])
	pLoseBG, _ := parseFloat32(parts[base+4])

	ca.Player1WinRate = pWin
	ca.Player1GammonRate = pWinGammon
	ca.Player1BackgammonRate = pWinBG
	ca.Player2WinRate = 1.0 - pWin // Opponent win rate = 1 - player win rate
	ca.Player2GammonRate = pLoseGammon
	ca.Player2BackgammonRate = pLoseBG

	ca.CubelessEquity, _ = strconv.ParseFloat(parts[base+5], 64)

	// Cubeful No-Double equity closes the no-double branch.
	if len(parts) > base+6 {
		ca.CubefulNoDouble, _ = strconv.ParseFloat(parts[base+6], 64)
	}

	// Cubeful Double/Take equity closes the double/take branch.
	if len(parts) > base+13 {
		ca.CubefulDoubleTake, _ = strconv.ParseFloat(parts[base+13], 64)
	}

	// Double/Pass equity: +1.0 for money games (normalized per cube value).
	// For match play, this should ideally be computed from the match equity table,
	// but +1.0 is the standard GNUbg convention for the DA property.
	ca.CubefulDoublePass = 1.0

	// BestAction is NOT computed here because in match play the cubeful equities
	// (ND and DT) are stored as MWC (Match Winning Chances, 0.0-1.0) while DP is
	// set to 1.0 (EMG convention). Comparing MWC with EMG gives wrong results.
	// The consumer (e.g., blunderDB) should convert MWC→EMG first, then compute BestAction.

	mr.CubeAnalysis = ca
}

// parseLuck parses the luck of a roll (LU property).
//
// Format: LU[value] — a single float, e.g. LU[-0.00537]. That is what gnuBG
// writes (WriteLuck, gnubg/sgf.c), and the luck classification it shows in the
// UI ("lucky", "very unlucky") is not part of this property: gnuBG records it
// separately as GB/GW. Requiring a rating word in front of the value here made
// every real file parse as "no luck at all".
//
// A two-field form is still accepted, so a file written by some other producer
// as "rating value" keeps working.
func parseLuck(node *SGFNode, mr *MoveRecord) {
	luStr := getProperty(node, "LU")
	if luStr == "" {
		return
	}

	parts := strings.Fields(luStr)
	if len(parts) == 0 {
		return
	}

	// The value is the last field; anything before it is a rating word.
	value, err := strconv.ParseFloat(parts[len(parts)-1], 64)
	if err != nil {
		return
	}

	// gnuBG writes LU[-inf] for a roll whose luck it never computed (ERR_VAL),
	// and Go parses that happily as -Inf. Reporting it as a number would hand
	// callers an infinity to average, so it counts as no luck at all.
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return
	}

	luck := &LuckRating{Value: value}
	if len(parts) > 1 {
		luck.Rating = parts[len(parts)-2]
	}
	mr.Luck = luck
}

// parseSkill parses skill rating (SK property)
// Format: SK[rating error]
func parseSkill(node *SGFNode, mr *MoveRecord) {
	skStr := getProperty(node, "SK")
	if skStr == "" {
		return
	}

	parts := strings.Fields(skStr)
	if len(parts) < 2 {
		return
	}

	mr.Skill = &SkillRating{
		Rating: parts[0],
	}
	mr.Skill.Error, _ = strconv.ParseFloat(parts[1], 64)
}

// parseFloat32 parses a float32 value
func parseFloat32(s string) (float32, error) {
	v, err := strconv.ParseFloat(s, 32)
	return float32(v), err
}
