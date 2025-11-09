package gnubgparser

import (
	"fmt"
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
	if mi := getProperty(node, "MI"); mi != "" {
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
		game.Moves = append(game.Moves, mr)

		// Check for luck rating
		if hasProperty(node, "LU") {
			parseLuck(node, &mr)
		}
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

// parseMoveAnalysis parses move analysis (A property)
// Format: A[ply][move rating ver version player1_win player1_gammon player1_bg player2_win player2_gammon equity ...]
// Example: A[0][lpab E ver 3 0.496365 0.140890 0.006297 0.135264 0.005951 -0.004723 ...]
// The probabilities come BEFORE equity in the format
func parseMoveAnalysis(node *SGFNode, mr *MoveRecord) {
	analysisStrs := node.Properties["A"]
	if len(analysisStrs) == 0 {
		return
	}

	mr.Analysis = &MoveAnalysis{
		Moves: make([]MoveOption, 0),
	}

	// Parse ply depth from first element (if it's a single number)
	plyDepth := 0
	if len(analysisStrs) > 0 {
		plyStr := strings.TrimSpace(analysisStrs[0])
		if depth, err := strconv.Atoi(plyStr); err == nil {
			// First element is the ply depth, remove it from the list
			plyDepth = depth
			analysisStrs = analysisStrs[1:]
		}
	}

	// Parse each move option
	for _, aStr := range analysisStrs {
		parts := strings.Fields(aStr)
		if len(parts) < 10 {
			continue
		}

		opt := MoveOption{}

		// Move encoding at parts[0]
		if len(parts[0]) >= 2 {
			parseEncodedMoveOption(parts[0], &opt)
		}

		// Format: parts[0]=move, parts[1]=rating(E/G/VB), parts[2]=ver, parts[3]=version
		// Then from GNU Backgammon source (sgf.c):
		// arEvalMove[0] arEvalMove[1] arEvalMove[2] arEvalMove[3] arEvalMove[4] rScore ...
		// Where arEvalMove indices are:
		//   0 = OUTPUT_WIN (player's total win probability)
		//   1 = OUTPUT_WINGAMMON (player wins gammon)
		//   2 = OUTPUT_WINBACKGAMMON (player wins backgammon)
		//   3 = OUTPUT_LOSEGAMMON (opponent wins gammon)
		//   4 = OUTPUT_LOSEBACKGAMMON (opponent wins backgammon)
		// And rScore = equity

		opt.Player1WinRate, _ = parseFloat32(parts[4])        // OUTPUT_WIN
		opt.Player1GammonRate, _ = parseFloat32(parts[5])     // OUTPUT_WINGAMMON
		opt.Player1BackgammonRate, _ = parseFloat32(parts[6]) // OUTPUT_WINBACKGAMMON
		opt.Player2GammonRate, _ = parseFloat32(parts[7])     // OUTPUT_LOSEGAMMON
		opt.Player2BackgammonRate, _ = parseFloat32(parts[8]) // OUTPUT_LOSEBACKGAMMON
		opt.Equity, _ = strconv.ParseFloat(parts[9], 64)      // rScore (equity)

		// Player2 win rate is calculated as 1.0 - Player1 win rate
		opt.Player2WinRate = 1.0 - opt.Player1WinRate

		// Set the ply depth from the first element
		opt.AnalysisDepth = plyDepth

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

// parseCubeAnalysis parses cube decision analysis (DA property)
// Format: DA[rating ver version cubelevel cubedecision skill matchlength player1_win player2_win player2_gam player1_gam player1_bg equity ...]
// Example: DA[E ver 3 2C 1 0.000000 1 0.503635 0.135264 0.005951 0.140890 0.006297 0.001137 ...]
func parseCubeAnalysis(node *SGFNode, mr *MoveRecord) {
	daStrs := node.Properties["DA"]
	if len(daStrs) == 0 {
		return
	}

	parts := strings.Fields(daStrs[0])
	if len(parts) < 13 {
		return
	}

	ca := &CubeAnalysis{}

	// parts[0] = rating (E, G, etc)
	// parts[1] = "ver"
	// parts[2] = version number
	// parts[3] = cube level (like "2C")
	// parts[4] = ? (often 1)
	// parts[5] = skill value (often 0.000000)
	// parts[6] = match length (often 1)

	// Probabilities start at index 7
	ca.Player1WinRate, _ = parseFloat32(parts[7])
	ca.Player2WinRate, _ = parseFloat32(parts[8])
	ca.Player2GammonRate, _ = parseFloat32(parts[9])
	ca.Player1GammonRate, _ = parseFloat32(parts[10])
	ca.Player1BackgammonRate, _ = parseFloat32(parts[11])

	// Equity at index 12
	ca.CubelessEquity, _ = strconv.ParseFloat(parts[12], 64)

	// Additional equities might follow
	if len(parts) >= 16 {
		ca.CubefulNoDouble, _ = strconv.ParseFloat(parts[13], 64)
		// The remaining values appear to repeat the probabilities and show cubeful equities
		// but the exact format needs more investigation
	}

	// Set analysis depth from parts[2] if it's numeric
	ca.AnalysisDepth, _ = strconv.Atoi(parts[2])

	// Determine best action
	if mr.Type == MoveTypeDouble {
		if ca.CubefulDoubleTake > ca.CubefulNoDouble {
			ca.BestAction = "double"
		} else {
			ca.BestAction = "no_double"
		}
	} else if mr.Type == MoveTypeTake || mr.Type == MoveTypeDrop {
		if ca.CubefulDoubleTake > ca.CubefulDoublePass {
			ca.BestAction = "take"
		} else {
			ca.BestAction = "pass"
		}
	}

	mr.CubeAnalysis = ca
}

// parseLuck parses luck rating (LU property)
// Format: LU[rating value]
func parseLuck(node *SGFNode, mr *MoveRecord) {
	luStr := getProperty(node, "LU")
	if luStr == "" {
		return
	}

	parts := strings.Fields(luStr)
	if len(parts) < 2 {
		return
	}

	mr.Luck = &LuckRating{
		Rating: parts[0],
	}
	mr.Luck.Value, _ = strconv.ParseFloat(parts[1], 64)
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
