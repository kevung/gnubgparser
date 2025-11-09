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
	scanner *bufio.Scanner
	lineNum int
}

// NewMATParser creates a new MAT parser from a reader
func NewMATParser(r io.Reader) *MATParser {
	return &MATParser{
		scanner: bufio.NewScanner(r),
	}
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
	moveLineRe = regexp.MustCompile(`^\s*(\d+)\)\s*(.*)$`)

	// Comment line starting with ; or #
	commentLineRe = regexp.MustCompile(`^\s*[;#]\s*(.*)$`)

	// EventDate comment: "; [EventDate "2025.11.08"]"
	eventDateRe = regexp.MustCompile(`\[EventDate\s+"(\d{4})\.(\d{2})\.(\d{2})"\]`)

	// Other metadata comments
	eventRe       = regexp.MustCompile(`\[Event\s+"([^"]+)"\]`)
	roundRe       = regexp.MustCompile(`\[Round\s+"([^"]+)"\]`)
	siteRe        = regexp.MustCompile(`\[Site\s+"([^"]+)"\]`)
	transcriberRe = regexp.MustCompile(`\[Transcriber\s+"([^"]+)"\]`)

	// Wins line: "                                  Wins 2 points"
	winsLineRe = regexp.MustCompile(`^\s*Wins\s+(\d+)\s+points?\s*$`)

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
	for p.scanner.Scan() {
		p.lineNum++
		line := p.scanner.Text()

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

	if matchLength == 0 && !p.scanner.Scan() {
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
	for p.scanner.Scan() {
		p.lineNum++
		line := p.scanner.Text()

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
	if !p.scanner.Scan() {
		return nil, io.EOF
	}
	p.lineNum++
	scoreLine := p.scanner.Text()

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

	for p.scanner.Scan() {
		p.lineNum++
		line := p.scanner.Text()

		// Check for wins line (end of game)
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

		// Check for next game starting
		if gameHeaderRe.MatchString(line) {
			// Put the line back for the next game parse
			// (We can't really unread, so we'll handle this in the main loop)
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
					}

					// Parse the move notation if present
					if moveStr != "" {
						moveArray := parseMatMove(moveStr)
						move.Move = moveArray
					}

					game.Moves = append(game.Moves, move)
					currentPlayer = player
				}
			}
		}
	}

	return game, nil
}

// splitMoveLine splits a move line into left (player1) and right (player2) parts
// MAT format uses multiple spaces (typically 3+) to separate the two player columns
func splitMoveLine(line string) [2]string {
	var result [2]string

	// Look for a sequence of 3 or more spaces that separates the two moves
	// This is more robust than using a fixed column number
	re := regexp.MustCompile(`\s{3,}`)
	loc := re.FindStringIndex(line)

	if loc != nil {
		// Found a large gap - split here
		result[0] = strings.TrimSpace(line[:loc[0]])
		result[1] = strings.TrimSpace(line[loc[1]:])
	} else {
		// No large gap found, entire line is one column
		result[0] = strings.TrimSpace(line)
	}

	return result
}

// parseMatMove converts MAT move notation to internal format
// MAT format: "6/5 8/5" or "13/9 24/23" or "bar/23"
func parseMatMove(moveStr string) [8]int {
	move := [8]int{-1, -1, -1, -1, -1, -1, -1, -1}

	if moveStr == "" || strings.Contains(strings.ToLower(moveStr), "can't move") {
		return move
	}

	// Split by spaces to get individual moves
	parts := strings.Fields(moveStr)
	idx := 0

	for _, part := range parts {
		if idx >= 8 {
			break
		}

		// Parse "from/to" format
		moveparts := strings.Split(part, "/")
		if len(moveparts) != 2 {
			continue
		}

		from := parseMatPoint(moveparts[0])
		to := parseMatPoint(moveparts[1])

		if from >= 0 && to >= -1 {
			move[idx] = from
			move[idx+1] = to
			idx += 2
		}
	}

	return move
}

// parseMatPoint converts a MAT point notation to internal format
// MAT uses: 1-24 for points, "bar" for bar, "off" for off
func parseMatPoint(s string) int {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "*") // Remove hit marker

	// Check for special points
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

	// Convert from MAT format (1-24) to internal format (0-23)
	if point >= 1 && point <= 24 {
		return point - 1
	}

	return -2 // invalid
}
