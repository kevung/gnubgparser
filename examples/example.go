package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kevung/gnubgparser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run example.go <sgf-file>")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Parse the SGF file
	match, err := gnubgparser.ParseSGFFile(filename)
	if err != nil {
		log.Fatalf("Error parsing file: %v", err)
	}

	// Display match information
	fmt.Println("=== Match Information ===")
	fmt.Printf("Players: %s vs %s\n", match.Metadata.Player1, match.Metadata.Player2)
	fmt.Printf("Match Length: %d points\n", match.Metadata.MatchLength)
	fmt.Printf("Date: %s\n", match.Metadata.Date)
	fmt.Printf("Total Games: %d\n\n", len(match.Games))

	// Display information about each game
	for i, game := range match.Games {
		fmt.Printf("--- Game %d ---\n", i+1)
		fmt.Printf("Score: %d-%d\n", game.Score[0], game.Score[1])

		// Count move types
		checkerMoves := 0
		doubles := 0
		takes := 0
		drops := 0

		for _, move := range game.Moves {
			switch move.Type {
			case gnubgparser.MoveTypeNormal:
				checkerMoves++
			case gnubgparser.MoveTypeDouble:
				doubles++
			case gnubgparser.MoveTypeTake:
				takes++
			case gnubgparser.MoveTypeDrop:
				drops++
			}
		}

		fmt.Printf("Total Moves: %d\n", len(game.Moves))
		fmt.Printf("Checker Moves: %d\n", checkerMoves)
		if doubles > 0 {
			fmt.Printf("Cube Actions: %d doubles, %d takes, %d drops\n", doubles, takes, drops)
		}

		// Display winner
		winnerName := match.Metadata.Player1
		if game.Winner == 1 {
			winnerName = match.Metadata.Player2
		}
		fmt.Printf("Winner: %s (%d points)", winnerName, game.Points)
		if game.Resigned {
			fmt.Printf(" - Resigned")
		}
		fmt.Println()

		// Show first few moves as example
		if len(game.Moves) > 0 {
			fmt.Println("\nFirst 3 moves:")
			count := 0
			for _, move := range game.Moves {
				if move.Type == gnubgparser.MoveTypeNormal && count < 3 {
					playerName := match.Metadata.Player1
					if move.Player == 1 {
						playerName = match.Metadata.Player2
					}
					fmt.Printf("  %s rolls %d-%d: %s\n",
						playerName, move.Dice[0], move.Dice[1], move.MoveString)

					// Show analysis if available
					if move.Analysis != nil && len(move.Analysis.Moves) > 0 {
						fmt.Printf("    Equity: %.3f\n", move.Analysis.Moves[0].Equity)
						fmt.Printf("    P1 Win/Gammon/BG: %.3f / %.3f / %.3f\n",
							move.Analysis.Moves[0].Player1WinRate,
							move.Analysis.Moves[0].Player1GammonRate,
							move.Analysis.Moves[0].Player1BackgammonRate)
						fmt.Printf("    P2 Win/Gammon/BG: %.3f / %.3f / %.3f\n",
							move.Analysis.Moves[0].Player2WinRate,
							move.Analysis.Moves[0].Player2GammonRate,
							move.Analysis.Moves[0].Player2BackgammonRate)
					}
					count++
				}
			}
		}
		fmt.Println()
	}

	// Export to JSON (optional)
	fmt.Println("=== JSON Export ===")
	jsonData, err := match.ToJSON()
	if err != nil {
		log.Fatalf("Error converting to JSON: %v", err)
	}
	fmt.Printf("JSON size: %d bytes\n", len(jsonData))
	fmt.Println("(Use -format=json flag to see full JSON output)")
}
