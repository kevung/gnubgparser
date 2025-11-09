package gnubgparser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// SGFNode represents a node in the SGF game tree
type SGFNode struct {
	Properties map[string][]string
	Children   []*SGFNode
}

// SGFParser handles parsing of SGF files
type SGFParser struct {
	reader  *bufio.Reader
	char    rune
	hasChar bool
}

// NewSGFParser creates a new SGF parser from a reader
func NewSGFParser(r io.Reader) *SGFParser {
	return &SGFParser{
		reader: bufio.NewReader(r),
	}
}

// ParseSGFFile parses an SGF file and returns a Match
func ParseSGFFile(filename string) (*Match, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return ParseSGF(file)
}

// ParseSGF parses SGF data from a reader and returns a Match
func ParseSGF(r io.Reader) (*Match, error) {
	parser := NewSGFParser(r)

	// Parse the SGF tree
	nodes, err := parser.parseGameTree()
	if err != nil {
		return nil, fmt.Errorf("failed to parse SGF: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("empty SGF file")
	}

	// Convert SGF nodes to Match structure
	match, err := convertNodesToMatch(nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert SGF to match: %w", err)
	}

	return match, nil
}

// parseGameTree parses an SGF game tree
func (p *SGFParser) parseGameTree() ([]*SGFNode, error) {
	var games []*SGFNode

	for {
		p.skipWhitespace()
		ch, err := p.peekChar()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if ch == '(' {
			// Parse a game
			game, err := p.parseGame()
			if err != nil {
				return nil, err
			}
			games = append(games, game)
		} else {
			break
		}
	}

	return games, nil
}

// parseGame parses a single game (game tree in parentheses)
func (p *SGFParser) parseGame() (*SGFNode, error) {
	// Expect '('
	ch, err := p.readChar()
	if err != nil || ch != '(' {
		return nil, fmt.Errorf("expected '(' at start of game")
	}

	// Parse sequence of nodes
	root := &SGFNode{Properties: make(map[string][]string)}
	current := root

	for {
		p.skipWhitespace()
		ch, err := p.peekChar()
		if err != nil {
			return nil, err
		}

		if ch == ')' {
			// End of game tree
			p.readChar()
			break
		} else if ch == ';' {
			// Parse node
			node, err := p.parseNode()
			if err != nil {
				return nil, err
			}

			// First node is root
			if len(root.Properties) == 0 {
				root = node
				current = root
			} else {
				// Add as child
				current.Children = append(current.Children, node)
				current = node
			}
		} else if ch == '(' {
			// Variation (not commonly used in gnuBG)
			p.readChar() // consume '('
			// Skip variations for now
			depth := 1
			for depth > 0 {
				ch, err := p.readChar()
				if err != nil {
					return nil, err
				}
				if ch == '(' {
					depth++
				} else if ch == ')' {
					depth--
				}
			}
		} else {
			return nil, fmt.Errorf("unexpected character in game tree: %c", ch)
		}
	}

	return root, nil
}

// parseNode parses a single SGF node
func (p *SGFParser) parseNode() (*SGFNode, error) {
	// Expect ';'
	ch, err := p.readChar()
	if err != nil || ch != ';' {
		return nil, fmt.Errorf("expected ';' at start of node")
	}

	node := &SGFNode{Properties: make(map[string][]string)}

	for {
		p.skipWhitespace()
		ch, err := p.peekChar()
		if err != nil {
			return nil, err
		}

		// Check if this is a property identifier (uppercase letter)
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
			prop, values, err := p.parseProperty()
			if err != nil {
				return nil, fmt.Errorf("error parsing property: %w", err)
			}
			node.Properties[prop] = values
		} else {
			// End of node properties
			break
		}
	}

	return node, nil
}

// parseProperty parses a property and its values
func (p *SGFParser) parseProperty() (string, []string, error) {
	// Read property name (1-2 uppercase letters)
	name := ""
	for {
		ch, err := p.readChar()
		if err != nil {
			return "", nil, err
		}
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
			name += string(ch)
		} else {
			// Put back the character
			p.unreadChar()
			break
		}
		if len(name) >= 2 {
			break
		}
	}

	// Read property values [...]
	var values []string
	for {
		p.skipWhitespace()
		ch, err := p.peekChar()
		if err != nil {
			return "", nil, err
		}

		if ch == '[' {
			value, err := p.parsePropertyValue()
			if err != nil {
				return "", nil, err
			}
			values = append(values, value)
		} else {
			break
		}
	}

	return name, values, nil
}

// parsePropertyValue parses a property value in brackets [...]
func (p *SGFParser) parsePropertyValue() (string, error) {
	// Expect '['
	ch, err := p.readChar()
	if err != nil || ch != '[' {
		return "", fmt.Errorf("expected '[' at start of property value")
	}

	var value strings.Builder
	escaped := false

	for {
		ch, err := p.readChar()
		if err != nil {
			return "", err
		}

		if escaped {
			// Handle escape sequences
			value.WriteRune(ch)
			escaped = false
		} else if ch == '\\' {
			escaped = true
		} else if ch == ']' {
			break
		} else {
			value.WriteRune(ch)
		}
	}

	return value.String(), nil
}

// readChar reads the next character
func (p *SGFParser) readChar() (rune, error) {
	if p.hasChar {
		p.hasChar = false
		return p.char, nil
	}

	ch, _, err := p.reader.ReadRune()
	if err != nil {
		return 0, err
	}
	p.char = ch // Save the character for potential unread
	return ch, nil
}

// peekChar peeks at the next character without consuming it
func (p *SGFParser) peekChar() (rune, error) {
	if p.hasChar {
		return p.char, nil
	}

	ch, err := p.readChar()
	if err != nil {
		return 0, err
	}
	p.char = ch
	p.hasChar = true
	return ch, nil
}

// unreadChar unreads the last character
func (p *SGFParser) unreadChar() {
	p.hasChar = true
}

// skipWhitespace skips whitespace characters
func (p *SGFParser) skipWhitespace() {
	for {
		ch, err := p.peekChar()
		if err != nil {
			return
		}
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			p.readChar()
		} else {
			break
		}
	}
}

// Helper functions to extract property values

func getProperty(node *SGFNode, name string) string {
	if values, ok := node.Properties[name]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}

func getPropertyInt(node *SGFNode, name string) int {
	str := getProperty(node, name)
	if str == "" {
		return 0
	}
	val, _ := strconv.Atoi(str)
	return val
}

func getPropertyFloat(node *SGFNode, name string) float64 {
	str := getProperty(node, name)
	if str == "" {
		return 0
	}
	val, _ := strconv.ParseFloat(str, 64)
	return val
}

func hasProperty(node *SGFNode, name string) bool {
	_, ok := node.Properties[name]
	return ok
}
