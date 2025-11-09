package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kevung/gnubgparser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file.mat>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nExample MAT parser - parses Jellyfish .mat files\n")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Parse the MAT file
	match, err := gnubgparser.ParseMATFile(filename)
	if err != nil {
		log.Fatalf("Error parsing MAT file: %v\n", err)
	}

	// Print match information
	fmt.Println("=== Jellyfish MAT File ===")
	fmt.Printf("Match Length: %d point%s\n", match.Metadata.MatchLength, plural(match.Metadata.MatchLength))

	if match.Metadata.Date != "" {
		fmt.Printf("Date: %s\n", match.Metadata.Date)
	}
	if match.Metadata.Event != "" {
		fmt.Printf("Event: %s\n", match.Metadata.Event)
	}
	if match.Metadata.Place != "" {
		fmt.Printf("Place: %s\n", match.Metadata.Place)
	}

	fmt.Printf("\nTotal Games: %d\n", len(match.Games))

	// Print game summaries
	for _, game := range match.Games {
		fmt.Printf("\n--- Game %d ---\n", game.GameNumber)
		fmt.Printf("Score before: %d-%d\n", game.Score[0], game.Score[1])

		// Determine player names from first game's moves
		if game.GameNumber == 1 && len(match.Games) > 0 {
			// Extract player names if available (would need to be in metadata)
			fmt.Printf("Players: Player1 vs Player2\n")
		}

		// Count move types
		normalMoves := 0
		doubles := 0
		takes := 0
		drops := 0

		for _, move := range game.Moves {
			switch move.Type {
			case gnubgparser.MoveTypeNormal:
				normalMoves++
			case gnubgparser.MoveTypeDouble:
				doubles++
			case gnubgparser.MoveTypeTake:
				takes++
			case gnubgparser.MoveTypeDrop:
				drops++
			}
		}

		fmt.Printf("Total moves: %d\n", len(game.Moves))
		fmt.Printf("  Checker moves: %d\n", normalMoves)
		if doubles > 0 {
			fmt.Printf("  Cube doubles: %d\n", doubles)
			if takes > 0 {
				fmt.Printf("  Cube takes: %d\n", takes)
			}
			if drops > 0 {
				fmt.Printf("  Cube drops: %d\n", drops)
			}
		}

		// Show first few moves
		if len(game.Moves) > 0 {
			fmt.Println("\nFirst few moves:")
			for i := 0; i < len(game.Moves) && i < 5; i++ {
				move := game.Moves[i]
				playerName := fmt.Sprintf("Player %d", move.Player+1)

				switch move.Type {
				case gnubgparser.MoveTypeNormal:
					fmt.Printf("  %s: %d%d %s\n", playerName, move.Dice[0], move.Dice[1], move.MoveString)
				case gnubgparser.MoveTypeDouble:
					fmt.Printf("  %s: Doubles => %d\n", playerName, move.CubeValue)
				case gnubgparser.MoveTypeTake:
					fmt.Printf("  %s: Takes\n", playerName)
				case gnubgparser.MoveTypeDrop:
					fmt.Printf("  %s: Drops\n", playerName)
				}
			}
		}

		// Show winner
		if game.Winner >= 0 {
			winnerName := fmt.Sprintf("Player %d", game.Winner+1)
			fmt.Printf("\nWinner: %s (%d point%s)\n", winnerName, game.Points, plural(game.Points))
		}

		if game.CrawfordGame {
			fmt.Println("(Crawford game)")
		}
	}

	// Export to JSON
	fmt.Println("\n=== JSON Output ===")
	jsonData, err := match.ToJSON()
	if err != nil {
		log.Fatalf("Error converting to JSON: %v\n", err)
	}
	fmt.Println(string(jsonData))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
