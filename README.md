# gnubgparser

A Go package for parsing GNU Backgammon (gnuBG) and Jellyfish match files in SGF and MAT formats.

## Features

- Parse gnuBG SGF match files
- Parse Jellyfish MAT match files
- Extract match metadata (players, event, date, score)
- Parse game moves, cube decisions, and analysis
- JSON export for easy integration with backends
- Command-line tool for file parsing
- In-memory data structures for programmatic use

## Installation

```bash
go get github.com/kevung/gnubgparser
```

## Usage

### As a Library

```go
package main

import (
    "fmt"
    "log"
    "github.com/kevung/gnubgparser"
)

func main() {
    // Parse SGF file
    match, err := gnubgparser.ParseSGFFile("match.sgf")
    if err != nil {
        log.Fatal(err)
    }

    // OR parse MAT file
    match, err = gnubgparser.ParseMATFile("match.mat")
    if err != nil {
        log.Fatal(err)
    }

    // Access match data
    fmt.Printf("%s vs %s\n", match.Metadata.Player1, match.Metadata.Player2)
    fmt.Printf("Match length: %d\n", match.Metadata.MatchLength)
    fmt.Printf("Games: %d\n", len(match.Games))
    
    // Export to JSON
    jsonData, _ := match.ToJSON()
    fmt.Println(string(jsonData))
}
```

### Command-Line Tool

```bash
# Build
go build -o gnubgparser ./cmd/gnubgparser

# Parse SGF file to JSON
./gnubgparser -format=json match.sgf > match.json

# Parse MAT file to JSON
./gnubgparser -format=json match.mat > match.json

# Parse and display summary
./gnubgparser -format=summary match.sgf
./gnubgparser -format=summary match.mat
```

Example summary output:
```
=== Match Summary ===
Players: charlot1 vs charlot2
Match Length: 7 points
Date: 2025-11-08
Application: GNU Backgammon:1.08.003

Games: 4

--- Game 1 ---
Score: 0-0
Moves: 47
Winner: charlot2 (2 points) - Resigned
Crawford rule: enabled
Checker moves: 45
Doubles: 1 (Takes: 1, Drops: 0)
```

## Supported Formats

### SGF Format (GNU Backgammon)

This parser supports the gnuBG variant of SGF format (GM[6]) including:

- **Match Info (MI)**: Match length, game number, scores
- **Players (PW/PB)**: Player names and ratings (WR/BR)
- **Game Info**: Event (EV), round (RO), place (PC), date (DT)
- **Moves (B/W)**: Dice rolls and checker moves
- **Cube Actions**: Double, take, drop decisions
- **Analysis (A)**: Move analysis with equity and probabilities
- **Double Analysis (DA)**: Cube decision analysis
- **Statistics**: Luck, skill ratings, error analysis
- **Board Positions (AE/AW/AB)**: Position setup

### MAT Format (Jellyfish)

This parser supports Jellyfish MAT (match) files including:

- **Match Header**: Match length (point matches or money games)
- **Game Headers**: Game number and scores
- **Player Names**: Extracted from game headers
- **Moves**: Dice rolls and move notation (e.g., "13/9 24/23")
- **Cube Actions**: Doubles, takes, drops
- **Game Results**: Winner and points won
- **Metadata Comments**: EventDate, Event, Site, etc. (in comment lines)
- **Point Notation**: Standard notation (1-24, bar, off)

## Data Structures

The parser creates a structured representation of the match:

- `Match`: Top-level structure containing metadata and games
- `Game`: Individual game with moves and result
- `MoveRecord`: Checker moves, cube decisions, analysis
- `Analysis`: Equity calculations and probability distributions
- `Position`: Board position with checkers and cube state

## Inspiration

This project is inspired by [xgparser](https://github.com/kevung/xgparser) which parses eXtremeGammon match files. gnubgparser provides similar functionality for GNU Backgammon's SGF format.

## License

LGPL-2.1 (matching gnuBG's license)
