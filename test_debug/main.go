package main
import ("encoding/json"; "fmt"; "github.com/kevung/gnubgparser")
func main() {
main.go match, _ := gnubgparser.ParseSGFFile("../test/charlot1-charlot2_7p_2025-11-08-2305.sgf")
main.go if len(match.Games) > 0 && len(match.Games[0].Moves) > 0 {
main.go main.go move := match.Games[0].Moves[0]
main.go main.go fmt.Printf("Move Type: %s, Player: %d, Dice: %v\n", move.Type, move.Player, move.Dice)
main.go main.go if move.Analysis != nil && len(move.Analysis.Moves) > 0 {
main.go main.go main.go opt := move.Analysis.Moves[0]
main.go main.go main.go data, _ := json.MarshalIndent(opt, "", "  ")
main.go main.go main.go fmt.Printf("First option:\n%s\n", string(data))
main.go main.go }
main.go }
}
