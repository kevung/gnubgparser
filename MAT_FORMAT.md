# Jellyfish MAT Format Support

The gnubgparser now supports parsing Jellyfish MAT (match) files in addition to GNU Backgammon SGF files.

## MAT Format Overview

Jellyfish MAT files are text-based match files with a simple, human-readable format:

```
; [EventDate "2025.11.08"]

 7 point match

 Game 1
 charlot1 : 0                   charlot2 : 0
  1)                             41: 13/9 24/23 
  2) 31: 6/5 8/5                 41: 6/5 9/5 
  3) 31: 24/21 6/5               65: 24/18 23/18 
  4) 41: 8/4 5/4                 21: 6/4* 18/17* 
  ...
 10) 61: 9/8 13/7                 Doubles => 2
 11)  Takes                      64: 13/7 7/3 
  ...
                                  Wins 2 points
```

## Format Structure

### File Header
- Optional comment lines starting with `;` or `#`
- Metadata can be embedded in comments: `[EventDate "YYYY.MM.DD"]`, `[Event "name"]`, etc.
- Match header: `N point match` (or `0 point match` for money games)

### Game Structure
- Game header: `Game N`
- Score line: `player1 : score1                   player2 : score2`
- Move lines: `N) [move1]                 [move2]`
  - Format: `dice: point/point point/point` (e.g., `31: 6/5 8/5`)
  - Special moves: `bar/23` (from bar), `6/off` (bearing off)
  - Hits marked with `*`: `6/4*`
- Cube actions:
  - `Doubles => N` (offer double to N)
  - `Takes` (accept double)
  - `Drops` (decline double)
- Game result: `Wins N points`

### Point Notation
- Points numbered 1-24 (standard backgammon notation)
- `bar` = on the bar
- `off` = borne off

## Usage

### Parsing MAT Files

```go
import "github.com/kevung/gnubgparser"

// Parse a MAT file
match, err := gnubgparser.ParseMATFile("match.mat")
if err != nil {
    log.Fatal(err)
}

// Access match data (same structure as SGF)
fmt.Printf("%s vs %s\n", match.Metadata.Player1, match.Metadata.Player2)
fmt.Printf("Match length: %d points\n", match.Metadata.MatchLength)

// Iterate through games
for _, game := range match.Games {
    fmt.Printf("Game %d: %d-%d\n", game.GameNumber, game.Score[0], game.Score[1])
    
    // Process moves
    for _, move := range game.Moves {
        switch move.Type {
        case gnubgparser.MoveTypeNormal:
            fmt.Printf("  %d%d: %s\n", move.Dice[0], move.Dice[1], move.MoveString)
        case gnubgparser.MoveTypeDouble:
            fmt.Printf("  Doubles => %d\n", move.CubeValue)
        case gnubgparser.MoveTypeTake:
            fmt.Println("  Takes")
        case gnubgparser.MoveTypeDrop:
            fmt.Println("  Drops")
        }
    }
}
```

### Command-Line Tool

```bash
# Parse MAT file to JSON
gnubgparser match.mat > match.json

# Display summary
gnubgparser -format=summary match.mat
```

## Supported Features

### Fully Supported
- ✅ Match length (point matches and money games)
- ✅ Player names (extracted from score lines)
- ✅ Game scores
- ✅ Dice rolls and moves
- ✅ Cube doubles, takes, drops
- ✅ Game winners and points
- ✅ Metadata comments (EventDate, Event, Site, etc.)
- ✅ Crawford game detection
- ✅ All standard move notation (including bar and bearoff)

### Not Available in MAT Format
- ❌ Move analysis (equity, probabilities)
- ❌ Cube decision analysis
- ❌ Luck and skill ratings
- ❌ Player ratings (unless in comments)
- ❌ Board positions (only moves are stored)

## Data Structures

The MAT parser produces the same `Match` structure as the SGF parser, making it easy to work with both formats:

```go
type Match struct {
    Metadata MatchMetadata
    Games    []Game
}

type Game struct {
    GameNumber   int
    Score        [2]int
    Winner       int
    Points       int
    Crawford     bool
    CrawfordGame bool
    Moves        []MoveRecord
    // ... other fields
}

type MoveRecord struct {
    Type       MoveType  // "move", "double", "take", "drop"
    Player     int       // 0 or 1
    Dice       [2]int    // For normal moves
    Move       [8]int    // Encoded move
    MoveString string    // Human-readable (e.g., "13/9 24/23")
    CubeValue  int       // For doubles
    // ... other fields
}
```

## Differences from SGF

1. **No Analysis**: MAT files only store moves, not evaluation data
2. **Simpler Format**: Text-based, easy to read and edit
3. **Column Layout**: Uses whitespace to separate player columns
4. **Legacy Format**: Widely used but less feature-rich than SGF

## Compatibility

The MAT parser has been tested with:
- Jellyfish-generated MAT files
- GridGammon exported MAT files
- GamesGrid MAT files

## Examples

See `examples/mat_example/` for a complete working example that:
- Parses a MAT file
- Displays match information
- Shows move-by-move data
- Exports to JSON

## Testing

Run MAT parser tests:
```bash
go test -v -run TestMAT
```

All MAT parsing functionality is covered by unit tests in `matparser_test.go`.
