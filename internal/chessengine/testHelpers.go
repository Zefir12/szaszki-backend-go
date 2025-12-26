package chess

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

///### helpers

type PGNGame struct {
	Event  string
	White  string
	Black  string
	Result string
	Moves  []string
}

type PGNTest struct {
	Name     string
	PGN      string
	FinalFEN string
}

func ExtractFinalFEN(pgn string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(pgn))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[CurrentPosition") {
			// line format: [CurrentPosition "FEN"]
			start := strings.Index(line, "\"")
			end := strings.LastIndex(line, "\"")
			if start == -1 || end == -1 || end <= start {
				return "", errors.New("invalid CurrentPosition header format")
			}
			return line[start+1 : end], nil
		}
	}
	return "", errors.New("CurrentPosition header not found")
}

func LoadPGNTests(path string) ([]PGNTest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var games []PGNTest

	var currentName string
	var currentPGN []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			// Start of a new game
			if len(currentPGN) > 0 {
				// Save the previous game
				pgnText := strings.Join(currentPGN, "\n")
				finalFEN, err := ExtractFinalFEN(pgnText)
				if err != nil {
					return nil, fmt.Errorf("could not extract final FEN for game %s: %v", currentName, err)
				}
				games = append(games, PGNTest{
					Name:     currentName,
					PGN:      pgnText,
					FinalFEN: finalFEN,
				})
			}
			// Start new game
			currentName = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			currentPGN = []string{}
			continue
		}

		currentPGN = append(currentPGN, line)
	}

	// Add the last game
	if len(currentPGN) > 0 {
		pgnText := strings.Join(currentPGN, "\n")
		finalFEN, err := ExtractFinalFEN(pgnText)
		if err != nil {
			return nil, fmt.Errorf("could not extract final FEN for game %s: %v", currentName, err)
		}
		games = append(games, PGNTest{
			Name:     currentName,
			PGN:      pgnText,
			FinalFEN: finalFEN,
		})
	}

	return games, nil
}

func ParsePGN(filename string) ([]PGNGame, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var games []PGNGame
	var current PGNGame
	var movesSection strings.Builder

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "[Event ") {
			current.Event = extractPGNValue(line)
		} else if strings.HasPrefix(line, "[White ") {
			current.White = extractPGNValue(line)
		} else if strings.HasPrefix(line, "[Black ") {
			current.Black = extractPGNValue(line)
		} else if strings.HasPrefix(line, "[Result ") {
			current.Result = extractPGNValue(line)
		} else if line != "" && !strings.HasPrefix(line, "[") {
			movesSection.WriteString(line + " ")
		} else if line == "" && movesSection.Len() > 0 {
			// Parse moves
			current.Moves = parseMoveText(movesSection.String())
			games = append(games, current)
			current = PGNGame{}
			movesSection.Reset()
		}
	}

	return games, scanner.Err()
}

func ParsePGNReader(r io.Reader) ([]PGNGame, error) {
	var games []PGNGame
	var current PGNGame
	var movesSection strings.Builder

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "[Event ") {
			current.Event = extractPGNValue(line)
		} else if strings.HasPrefix(line, "[White ") {
			current.White = extractPGNValue(line)
		} else if strings.HasPrefix(line, "[Black ") {
			current.Black = extractPGNValue(line)
		} else if strings.HasPrefix(line, "[Result ") {
			current.Result = extractPGNValue(line)
		} else if line != "" && !strings.HasPrefix(line, "[") {
			movesSection.WriteString(line + " ")
		} else if line == "" && movesSection.Len() > 0 {
			current.Moves = parseMoveText(movesSection.String())
			games = append(games, current)
			current = PGNGame{}
			movesSection.Reset()
		}
	}

	if movesSection.Len() > 0 {
		current.Moves = parseMoveText(movesSection.String())
		games = append(games, current)
	}

	return games, scanner.Err()
}

func extractPGNValue(line string) string {
	start := strings.Index(line, "\"")
	end := strings.LastIndex(line, "\"")
	if start >= 0 && end > start {
		return line[start+1 : end]
	}
	return ""
}

func parseMoveText(text string) []string {
	text = strings.ReplaceAll(text, "\n", " ")

	replacements := []string{
		"+", "", "#", "",
		"!", "", "?", "",
	}

	for i := 0; i < len(replacements); i += 2 {
		text = strings.ReplaceAll(text, replacements[i], replacements[i+1])
	}

	text = strings.ReplaceAll(text, "1-0", "")
	text = strings.ReplaceAll(text, "0-1", "")
	text = strings.ReplaceAll(text, "1/2-1/2", "")

	tokens := strings.Fields(text)
	moves := []string{}

	for _, tok := range tokens {
		if strings.HasSuffix(tok, ".") {
			continue
		}
		moves = append(moves, tok)
	}

	return moves
}

func getBitboardByPiece(b *Board, color int8, piece int) Bitboard {
	switch piece {
	case Pawn:
		return b.Pawns[color]
	case Knight:
		return b.Knights[color]
	case Bishop:
		return b.Bishops[color]
	case Rook:
		return b.Rooks[color]
	case Queen:
		return b.Queens[color]
	case King:
		return b.Kings[color]
	}
	return 0
}

func SANToMove(board *Board, san string) (Move, error) {
	// --- 1. Cleanup ---
	san = strings.TrimSpace(san)
	san = strings.TrimRight(san, "+#")

	if san == "O-O" {
		if board.Flags&WhiteToMove != 0 {
			return Move{From: 4, To: 6}, nil
		}
		return Move{From: 60, To: 62}, nil
	}

	if san == "O-O-O" {
		if board.Flags&WhiteToMove != 0 {
			return Move{From: 4, To: 2}, nil
		}
		return Move{From: 60, To: 58}, nil
	}

	// --- 2. Promotion ---
	promotion := int8(0)
	if i := strings.Index(san, "="); i != -1 {
		switch san[i+1] {
		case 'Q':
			promotion = Queen
		case 'R':
			promotion = Rook
		case 'B':
			promotion = Bishop
		case 'N':
			promotion = Knight
		}
		san = san[:i]
	}

	// --- 3. Piece type ---
	piece := Pawn
	start := 0

	switch san[0] {
	case 'N':
		piece = Knight
		start = 1
	case 'B':
		piece = Bishop
		start = 1
	case 'R':
		piece = Rook
		start = 1
	case 'Q':
		piece = Queen
		start = 1
	case 'K':
		piece = King
		start = 1
	}

	// Remove capture marker
	san = strings.ReplaceAll(san, "x", "")

	// --- 4. Target square ---
	if len(san) < start+2 {
		return Move{}, fmt.Errorf("invalid SAN: %s", san)
	}

	file := san[len(san)-2] - 'a'
	rank := san[len(san)-1] - '1'
	to := int8(rank*8 + file)

	// --- 5. Disambiguation ---
	disFile := int8(-1)
	disRank := int8(-1)
	middle := san[start : len(san)-2]

	for _, ch := range middle {
		if ch >= 'a' && ch <= 'h' {
			disFile = int8(ch - 'a')
		}
		if ch >= '1' && ch <= '8' {
			disRank = int8(ch - '1')
		}
	}

	// --- 6. Find legal origin ---
	color := Black
	if board.Flags&WhiteToMove != 0 {
		color = White
	}

	var candidates []int8
	bb := getBitboardByPiece(board, int8(color), piece)

	// fmt.Printf("SAN: %s\n", san)
	// fmt.Printf("Piece: %d, Color: %d, Target square: %d\n", piece, color, to)
	// fmt.Printf("Disambiguation: file=%d, rank=%d\n", disFile, disRank)
	// fmt.Printf("Bitboard: %064b\n", bb)

	// First pass: strict SAN (with disambiguation)
	bbCopy := bb
	for bbCopy != 0 {
		from := int8(PopLSB(&bbCopy))

		if piece == Pawn {
			// Add file check for pawn moves without explicit capture notation
			fromFile := from % 8
			toFile := to % 8

			// For non-capture pawn moves, file must match
			if disFile == -1 && strings.Index(san, "x") == -1 {
				if fromFile != toFile {
					continue
				}
			}

			// Only consider pawns that can legally move forward
			if disFile != -1 && from%8 != disFile {
				continue
			}
		} else {
			if disFile != -1 && from%8 != disFile {
				continue
			}
			if disRank != -1 && from/8 != disRank {
				continue
			}
		}

		if IsMoveLegal(board, from, to, promotion) {
			//fmt.Printf("Move allowed: %d -> %d\n", from, to)
			return Move{From: from, To: to, Promotion: promotion}, nil
		} else {
			//fmt.Printf("Move blocked: %d -> %d\n", from, to)
		}
	}

	// Second pass: ignore disambiguation
	bbCopy = bb
	for bbCopy != 0 {
		from := int8(PopLSB(&bbCopy))

		if piece == Pawn {
			targetRank := int8(rank)
			if color == White && from/8 >= targetRank {
				continue
			}
			if color == Black && from/8 <= targetRank {
				continue
			}
		}

		if IsMoveLegal(board, from, to, promotion) {
			candidates = append(candidates, from)
		}
	}

	if len(candidates) == 1 {
		return Move{From: candidates[0], To: to, Promotion: promotion}, nil
	}

	return Move{}, fmt.Errorf("cannot resolve SAN move: %s", san)
}
