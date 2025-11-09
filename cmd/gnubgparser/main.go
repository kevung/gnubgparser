// gnubgparser command-line tool
//
// Parse gnuBG SGF match files and output JSON or summary information.
//
// Usage:
//   gnubgparser <file.sgf>              - Parse and output JSON
//   gnubgparser -format=summary <file.sgf> - Show match summary

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kevung/gnubgparser"
)

var (
	formatFlag = flag.String("format", "json", "Output format: json, summary")
)

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <file.sgf|file.mat>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nSupported formats:\n")
		fmt.Fprintf(os.Stderr, "  .sgf - GNU Backgammon SGF format\n")
		fmt.Fprintf(os.Stderr, "  .mat - Jellyfish MAT format\n")
		os.Exit(1)
	}

	filename := flag.Arg(0)

	// Determine file type and parse accordingly
	var match *gnubgparser.Match
	var err error

	if len(filename) > 4 && filename[len(filename)-4:] == ".mat" {
		// Parse MAT file
		match, err = gnubgparser.ParseMATFile(filename)
		if err != nil {
			log.Fatalf("Error parsing MAT file: %v\n", err)
		}
	} else {
		// Parse SGF file (default)
		match, err = gnubgparser.ParseSGFFile(filename)
		if err != nil {
			log.Fatalf("Error parsing SGF file: %v\n", err)
		}
	}

	// Output based on format
	switch *formatFlag {
	case "json":
		jsonData, err := match.ToJSON()
		if err != nil {
			log.Fatalf("Error converting to JSON: %v\n", err)
		}
		fmt.Println(string(jsonData))

	case "summary":
		printSummary(match)

	default:
		log.Fatalf("Unknown format: %s\n", *formatFlag)
	}
}

func printSummary(match *gnubgparser.Match) {
	fmt.Println("=== Match Summary ===")
	fmt.Printf("Players: %s vs %s\n", match.Metadata.Player1, match.Metadata.Player2)

	if match.Metadata.Rating1 != "" || match.Metadata.Rating2 != "" {
		fmt.Printf("Ratings: %s vs %s\n", match.Metadata.Rating1, match.Metadata.Rating2)
	}

	if match.Metadata.MatchLength > 0 {
		fmt.Printf("Match Length: %d points\n", match.Metadata.MatchLength)
	} else {
		fmt.Println("Match Type: Money game")
	}

	if match.Metadata.Event != "" {
		fmt.Printf("Event: %s\n", match.Metadata.Event)
	}
	if match.Metadata.Round != "" {
		fmt.Printf("Round: %s\n", match.Metadata.Round)
	}
	if match.Metadata.Place != "" {
		fmt.Printf("Location: %s\n", match.Metadata.Place)
	}
	if match.Metadata.Date != "" {
		fmt.Printf("Date: %s\n", match.Metadata.Date)
	}
	if match.Metadata.Application != "" {
		fmt.Printf("Application: %s\n", match.Metadata.Application)
	}

	fmt.Printf("\nGames: %d\n", len(match.Games))

	for i, game := range match.Games {
		fmt.Printf("\n--- Game %d ---\n", i+1)
		fmt.Printf("Score: %d-%d\n", game.Score[0], game.Score[1])
		fmt.Printf("Moves: %d\n", len(game.Moves))

		if game.Winner >= 0 {
			winner := match.Metadata.Player1
			if game.Winner == 1 {
				winner = match.Metadata.Player2
			}
			fmt.Printf("Winner: %s", winner)
			if game.Points > 0 {
				fmt.Printf(" (%d points)", game.Points)
			}
			if game.Resigned {
				fmt.Print(" - Resigned")
			}
			fmt.Println()
		}

		if game.Crawford {
			fmt.Println("Crawford rule: enabled")
		}
		if game.CrawfordGame {
			fmt.Println("This is the Crawford game")
		}
		if game.Jacoby {
			fmt.Println("Jacoby rule: enabled")
		}
		if !game.CubeEnabled {
			fmt.Println("Cube: disabled")
		}
		if game.Variation != "" && game.Variation != "Standard" {
			fmt.Printf("Variation: %s\n", game.Variation)
		}

		// Count move types
		moves := 0
		doubles := 0
		takes := 0
		drops := 0

		for _, mr := range game.Moves {
			switch mr.Type {
			case gnubgparser.MoveTypeNormal:
				moves++
			case gnubgparser.MoveTypeDouble:
				doubles++
			case gnubgparser.MoveTypeTake:
				takes++
			case gnubgparser.MoveTypeDrop:
				drops++
			}
		}

		if moves > 0 {
			fmt.Printf("Checker moves: %d\n", moves)
		}
		if doubles > 0 {
			fmt.Printf("Doubles: %d (Takes: %d, Drops: %d)\n", doubles, takes, drops)
		}
	}
}
