package main

import (
"encoding/json"
"fmt"
"github.com/kevung/gnubgparser"
)

func main() {
match, err := gnubgparser.ParseSGFFile("../test/charlot1-charlot2_7p_2025-11-08-2305.sgf")
if err != nil {
fmt.Printf("Error: %v\n", err)
return
}

if len(match.Games) > 0 && len(match.Games[0].Moves) > 0 {
// Print first 3 moves with analysis
for i := 0; i < 3 && i < len(match.Games[0].Moves); i++ {
move := match.Games[0].Moves[i]
fmt.Printf("\n=== Move %d ===\n", i)
fmt.Printf("Type: %s, Player: %d\n", move.Type, move.Player)
fmt.Printf("Dice: %v\n", move.Dice)

if move.Analysis != nil {
fmt.Printf("Analysis has %d moves\n", len(move.Analysis.Moves))
if len(move.Analysis.Moves) > 0 {
opt := move.Analysis.Moves[0]
data, _ := json.MarshalIndent(opt, "", "  ")
fmt.Printf("First option:\n%s\n", string(data))
}
}

if move.CubeAnalysis != nil {
data, _ := json.MarshalIndent(move.CubeAnalysis, "", "  ")
fmt.Printf("Cube analysis:\n%s\n", string(data))
}
}
}
}
