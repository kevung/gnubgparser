package gnubgparser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// MATParser handles parsing of Jellyfish .mat files
type MATParser struct {
	scanner    *bufio.Scanner
	lineNum    int
	peekedLine string
	hasPeeked  bool
}

// NewMATParser creates a new MAT parser from a reader
func NewMATParser(r io.Reader) *MATParser {
	return &MATParser{
		scanner: bufio.NewScanner(r),
	}
}

// nextLine returns the next line, using the peeked line if available.
func (p *MATParser) nextLine() (string, bool) {
	if p.hasPeeked {
		p.hasPeeked = false
		p.lineNum++
		return p.peekedLine, true
	}
	if p.scanner.Scan() {
		p.lineNum++
		return p.scanner.Text(), true
	}
	return "", false
}

// unreadLine pushes a line back so the next call to nextLine returns it.
func (p *MATParser) unreadLine(line string) {
	p.peekedLine = line
	p.hasPeeked = true
	p.lineNum-- // will be re-incremented on next read
}

// ParseMATFile parses a .mat file and returns a Match
func ParseMATFile(filename string) (*Match, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return ParseMAT(file)
}

// ParseMAT parses MAT data from a reader and returns a Match
func ParseMAT(r io.Reader) (*Match, error) {
	parser := NewMATParser(r)
	return parser.parse()
}

// Regular expressions for MAT format parsing
var (
	// Match header: " 7 point match"
	matchHeaderRe = regexp.MustCompile(`^\s*(\d+)\s+point\s+match\s*$`)

	// Game header: " Game 1" or "Game 1"
	gameHeaderRe = regexp.MustCompile(`^\s*Game\s+(\d+)\s*$`)

	// Score line: " charlot1 : 0                   charlot2 : 2"
	scoreLineRe = regexp.MustCompile(`^\s*(\S+.*?)\s*:\s*(\d+)\s+(\S+.*?)\s*:\s*(\d+)\s*$`)

	// Move line: "  1) 31: 6/5 8/5" or "  1)                             41: 13/9 24/23"
	// Note: Do NOT use \s* after ) — the leading whitespace is needed by splitMoveLine
	// to detect whether the left column is empty (3+ spaces = column separator).
	moveLineRe = regexp.MustCompile(`^\s*(\d+)\)(.*)$`)

	// Comment line starting with ; or #
	commentLineRe = regexp.MustCompile(`^\s*[;#]\s*(.*)$`)

	// EventDate comment: "; [EventDate "2025.11.08"]"
	eventDateRe = regexp.MustCompile(`\[EventDate\s+"(\d{4})\.(\d{2})\.(\d{2})"\]`)

	// Other metadata comments
	eventRe       = regexp.MustCompile(`\[Event\s+"([^"]+)"\]`)
	roundRe       = regexp.MustCompile(`\[Round\s+"([^"]+)"\]`)
	siteRe        = regexp.MustCompile(`\[Site\s+"([^"]+)"\]`)
	transcriberRe = regexp.MustCompile(`\[Transcriber\s+"([^"]+)"\]`)

	// Wins line: "                                  Wins 2 points" or "Wins 2 points and the match"
	winsLineRe = regexp.MustCompile(`^\s*Wins\s+(\d+)\s+points?(?:\s+and\s+the\s+match)?\s*$`)

	// Dice and move: "31: 6/5 8/5" or "41: 13/9 24/23"
	diceAndMoveRe = regexp.MustCompile(`^(\d)(\d):\s*(.*)$`)

	// Cube actions
	doublesRe = regexp.MustCompile(`^Doubles\s*=>\s*(\d+)\s*$`)
	takesRe   = regexp.MustCompile(`^Takes\s*$`)
	dropsRe   = regexp.MustCompile(`^Drops\s*$`)
)

// parse parses the entire MAT file
func (p *MATParser) parse() (*Match, error) {
	match := &Match{
		Metadata: MatchMetadata{},
		Games:    []Game{},
	}

	// Parse comments and match header
	matchLength := 0
	for {
		line, ok := p.nextLine()
		if !ok {
			break
		}

		// Check for comments with metadata
		if matches := commentLineRe.FindStringSubmatch(line); matches != nil {
			p.parseMetadataComment(match, matches[1])
			continue
		}

		// Check for match header
		if matches := matchHeaderRe.FindStringSubmatch(line); matches != nil {
			length, _ := strconv.Atoi(matches[1])
			matchLength = length
			match.Metadata.MatchLength = length
			break
		}
	}

	if matchLength == 0 {
		return nil, fmt.Errorf("invalid MAT file: no match header found")
	}

	// Parse games
	for {
		game, err := p.parseGame(matchLength, match)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error parsing game at line %d: %w", p.lineNum, err)
		}
		if game != nil {
			match.Games = append(match.Games, *game)
		}
	}

	if len(match.Games) == 0 {
		return nil, fmt.Errorf("no games found in MAT file")
	}

	return match, nil
}

// parseMetadataComment extracts metadata from comment lines
func (p *MATParser) parseMetadataComment(match *Match, comment string) {
	if matches := eventDateRe.FindStringSubmatch(comment); matches != nil {
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])
		match.Metadata.Date = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
	} else if matches := eventRe.FindStringSubmatch(comment); matches != nil {
		match.Metadata.Event = matches[1]
	} else if matches := roundRe.FindStringSubmatch(comment); matches != nil {
		match.Metadata.Round = matches[1]
	} else if matches := siteRe.FindStringSubmatch(comment); matches != nil {
		match.Metadata.Place = matches[1]
	} else if matches := transcriberRe.FindStringSubmatch(comment); matches != nil {
		match.Metadata.Annotator = matches[1]
	}
}

// parseGame parses a single game
func (p *MATParser) parseGame(matchLength int, match *Match) (*Game, error) {
	// Find game header
	var gameNumber int
	for {
		line, ok := p.nextLine()
		if !ok {
			break
		}

		if matches := gameHeaderRe.FindStringSubmatch(line); matches != nil {
			num, _ := strconv.Atoi(matches[1])
			gameNumber = num
			break
		}
	}

	if gameNumber == 0 {
		return nil, io.EOF
	}

	// Parse score line
	scoreLine, ok := p.nextLine()
	if !ok {
		return nil, io.EOF
	}

	matches := scoreLineRe.FindStringSubmatch(scoreLine)
	if matches == nil {
		return nil, fmt.Errorf("invalid score line: %s", scoreLine)
	}

	player1 := strings.TrimSpace(matches[1])
	score1, _ := strconv.Atoi(matches[2])
	player2 := strings.TrimSpace(matches[3])
	score2, _ := strconv.Atoi(matches[4])

	// Update match metadata with player names (from first game)
	if gameNumber == 1 {
		// Remove trailing commas and ratings if present
		player1Clean := strings.Split(player1, ",")[0]
		player2Clean := strings.Split(player2, ",")[0]
		match.Metadata.Player1 = strings.TrimSpace(player1Clean)
		match.Metadata.Player2 = strings.TrimSpace(player2Clean)
	}

	game := &Game{
		GameNumber:  gameNumber,
		Score:       [2]int{score1, score2},
		Variation:   "Standard",
		Crawford:    matchLength > 0,
		Jacoby:      matchLength == 0,
		CubeEnabled: true,
		Winner:      -1,
		Moves:       []MoveRecord{},
	}

	// Determine if this is Crawford game
	if matchLength > 0 {
		if score1 == matchLength-1 && score2 < matchLength-1 {
			game.CrawfordGame = true
		} else if score2 == matchLength-1 && score1 < matchLength-1 {
			game.CrawfordGame = true
		}
	}

	// Parse moves
	currentPlayer := 1 // Start with player 2 (1-indexed in MAT format)
	cubeValue := 1
	_ = cubeValue // Will be used for cube tracking in future
	gameEnded := false

	for {
		line, ok := p.nextLine()
		if !ok {
			break
		}

		// Check for wins line (end of game)
		// "Wins" can appear standalone or on the right side of a move line
		if matches := winsLineRe.FindStringSubmatch(line); matches != nil {
			points, _ := strconv.Atoi(matches[1])
			game.Points = points
			game.Winner = currentPlayer
			break
		}

		// Check for empty line (might indicate end of game)
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for next game starting - unread the line so the next parseGame finds it
		if gameHeaderRe.MatchString(line) {
			p.unreadLine(line)
			break
		}

		// Parse move line
		if matches := moveLineRe.FindStringSubmatch(line); matches != nil {
			// moveNum, _ := strconv.Atoi(matches[1])
			moveContent := matches[2]

			// Split into left and right parts (player 1 and player 2)
			parts := splitMoveLine(moveContent)

			for i, part := range parts {
				if part == "" {
					continue
				}

				player := i // 0 = player1, 1 = player2

				// Check for "Wins N point(s)" on a column (e.g., right side of a move line)
				if wm := winsLineRe.FindStringSubmatch(part); wm != nil {
					points, _ := strconv.Atoi(wm[1])
					game.Points = points
					game.Winner = player
					gameEnded = true
					break
				}

				// Check for cube actions first (they don't have dice)
				if matches := doublesRe.FindStringSubmatch(part); matches != nil {
					newCube, _ := strconv.Atoi(matches[1])
					move := MoveRecord{
						Type:      MoveTypeDouble,
						Player:    player,
						CubeValue: newCube,
					}
					game.Moves = append(game.Moves, move)
					currentPlayer = player
					continue
				}

				if takesRe.MatchString(part) {
					move := MoveRecord{
						Type:   MoveTypeTake,
						Player: player,
					}
					game.Moves = append(game.Moves, move)
					cubeValue *= 2
					currentPlayer = player
					continue
				}

				if dropsRe.MatchString(part) {
					move := MoveRecord{
						Type:   MoveTypeDrop,
						Player: player,
					}
					game.Moves = append(game.Moves, move)
					// Game ends on a drop
					game.Winner = 1 - player
					currentPlayer = 1 - player
					gameEnded = true
					break
				}

				// Check for dice and move
				if matches := diceAndMoveRe.FindStringSubmatch(part); matches != nil {
					die1, _ := strconv.Atoi(matches[1])
					die2, _ := strconv.Atoi(matches[2])
					moveStr := strings.TrimSpace(matches[3])

					move := MoveRecord{
						Type:       MoveTypeNormal,
						Player:     player,
						Dice:       [2]int{die1, die2},
						MoveString: moveStr,
						Move:       parseMatMove(moveStr), // Always parse; empty string returns all -1 (no move)
					}

					game.Moves = append(game.Moves, move)
					currentPlayer = player
				}
			}
		}

		if gameEnded {
			break
		}
	}

	return game, nil
}

// splitMoveLine splits a move line into left (player1) and right (player2) parts
// MAT format uses multiple spaces (typically 3+) to separate the two player columns.
// When the left column has a long move (e.g., 4-submove doubles like "66: 22/16 22/16 16/10 16/10"),
// only 1-2 spaces may separate the columns. In that case, we fall back to detecting
// a second dice pattern (NN:) which marks the start of the right column.
func splitMoveLine(line string) [2]string {
	var result [2]string

	// Look for a sequence of 3 or more spaces that separates the two moves
	re := regexp.MustCompile(`\s{3,}`)
	loc := re.FindStringIndex(line)

	if loc != nil {
		left := strings.TrimSpace(line[:loc[0]])
		right := strings.TrimSpace(line[loc[1]:])

		if right != "" {
			// Clean split with non-empty right side
			result[0] = left
			result[1] = right
			return result
		}
		// Right side is empty (gap was trailing spaces) — fall through to secondary detection
	}

	// Fallback: look for a second dice pattern (NN:) in the content.
	// The first NN: is at the start of the line (left player's dice).
	// Any subsequent NN: preceded by whitespace marks the right player's column.
	trimmed := strings.TrimSpace(line)
	secondDiceRe := regexp.MustCompile(`\s(\d\d:\s*)`)
	allMatches := secondDiceRe.FindAllStringIndex(trimmed, -1)

	if len(allMatches) >= 1 {
		// Split at the space before the second dice pattern
		splitPos := allMatches[0][0]
		result[0] = strings.TrimSpace(trimmed[:splitPos])
		result[1] = strings.TrimSpace(trimmed[splitPos:])
		return result
	}

	// Also check for cube actions after a move: detect "Doubles", "Takes", "Drops"
	// following a dice+move pattern
	if len(allMatches) >= 1 {
		cubeRe := regexp.MustCompile(`\s+(Doubles|Takes|Drops)`)
		cubeLoc := cubeRe.FindStringIndex(trimmed)
		if cubeLoc != nil && cubeLoc[0] > allMatches[0][0] {
			result[0] = strings.TrimSpace(trimmed[:cubeLoc[0]])
			result[1] = strings.TrimSpace(trimmed[cubeLoc[0]:])
			return result
		}
	}

	// No large gap and no second dice pattern — entire line is one column
	result[0] = strings.TrimSpace(line)
	return result
}

// parseMatMove converts MAT move notation to internal format
// MAT format: "6/5 8/5" or "13/9 24/23" or "bar/23" or "18/16(2) 6/4(2)"
// The (N) suffix means the movement is repeated N times
func parseMatMove(moveStr string) [8]int {
	move := [8]int{-1, -1, -1, -1, -1, -1, -1, -1}

	if moveStr == "" || strings.Contains(strings.ToLower(moveStr), "can't move") ||
		strings.Contains(strings.ToLower(moveStr), "cannot move") {
		return move
	}

	// Split by spaces to get individual moves
	parts := strings.Fields(moveStr)
	idx := 0

	for _, part := range parts {
		if idx >= 8 {
			break
		}

		// Check for multiplier (N) suffix, e.g., "18/16(2)"
		multiplier := 1
		cleanPart := part
		if parenIdx := strings.LastIndex(part, "("); parenIdx >= 0 {
			if closeIdx := strings.Index(part[parenIdx:], ")"); closeIdx >= 0 {
				nStr := part[parenIdx+1 : parenIdx+closeIdx]
				if n, err := strconv.Atoi(nStr); err == nil && n > 0 {
					multiplier = n
				}
				// Remove the (N) suffix from the part for parsing
				cleanPart = part[:parenIdx]
			}
		}

		// Parse "from/to" format
		moveparts := strings.Split(cleanPart, "/")
		if len(moveparts) != 2 {
			continue
		}

		from := parseMatPoint(moveparts[0])
		to := parseMatPoint(moveparts[1])

		if from >= 0 && to >= -1 {
			for r := 0; r < multiplier && idx < 8; r++ {
				move[idx] = from
				move[idx+1] = to
				idx += 2
			}
		}
	}

	return move
}

// parseMatPoint converts a MAT point notation to internal format
// MAT uses: 1-24 for points, "bar" or 25 for bar, "off" or 0 for off
// May have suffixes like "*" (hit) or "(N)" (multiplier) which are stripped
func parseMatPoint(s string) int {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "*") // Remove hit marker

	// Remove "(N)" multiplier suffix (e.g., "16(2)" → "16")
	if idx := strings.Index(s, "("); idx >= 0 {
		s = s[:idx]
	}

	// Check for special points (text form, used in TXT format)
	lower := strings.ToLower(s)
	if lower == "bar" {
		return 24 // bar
	}
	if lower == "off" {
		return -1 // off
	}

	// Parse numeric point
	point, err := strconv.Atoi(s)
	if err != nil {
		return -2 // invalid
	}

	// MAT format uses 0 for bearoff and 25 for bar (numeric equivalents)
	if point == 0 {
		return -1 // off (bearoff)
	}
	if point == 25 {
		return 24 // bar
	}

	// Convert from MAT format (1-24) to internal format (0-23)
	if point >= 1 && point <= 24 {
		return point - 1
	}

	return -2 // invalid
}
