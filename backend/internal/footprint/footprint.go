// Package footprint parses docs/FOOTPRINT.md, the single source of truth for
// everything this project leaves on a machine.
package footprint

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Ownership classes used in the "Ownership" column.
const (
	ProjectOwned = "project-owned"
	Shared       = "shared"
	Manual       = "manual"
)

// Row is one line of a FOOTPRINT.md table.
type Row struct {
	ID        string
	Item      string
	Location  string
	Size      string
	Ownership string // normalised to one of the constants above
	Removal   string // PowerShell command, backticks stripped
}

// ParseFile parses the FOOTPRINT.md at path.
func ParseFile(path string) ([]Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return Parse(f)
}

// Parse reads every markdown table whose header starts with an "ID" column.
// Expected columns: ID | Item | Location | Approx. size | Ownership | Removal.
func Parse(r io.Reader) ([]Row, error) {
	var rows []Row
	seen := map[string]bool{}
	inTable := false
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "|") {
			inTable = false
			continue
		}
		cells := splitRow(line)
		if len(cells) == 0 {
			continue
		}
		if strings.EqualFold(cells[0], "ID") {
			inTable = true
			continue
		}
		if !inTable || isSeparator(cells) {
			continue
		}
		if len(cells) != 6 {
			return nil, fmt.Errorf("FOOTPRINT.md line %d: want 6 columns, got %d", lineNo, len(cells))
		}
		row := Row{
			ID:       strip(cells[0]),
			Item:     cells[1],
			Location: strip(cells[2]),
			Size:     cells[3],
			Removal:  strip(cells[5]),
		}
		own, err := normaliseOwnership(cells[4])
		if err != nil {
			return nil, fmt.Errorf("FOOTPRINT.md line %d: %w", lineNo, err)
		}
		row.Ownership = own
		if row.ID == "" {
			return nil, fmt.Errorf("FOOTPRINT.md line %d: empty ID", lineNo)
		}
		if seen[row.ID] {
			return nil, fmt.Errorf("FOOTPRINT.md line %d: duplicate ID %q", lineNo, row.ID)
		}
		seen[row.ID] = true
		rows = append(rows, row)
	}
	return rows, sc.Err()
}

// Filter returns the rows with the given ownership.
func Filter(rows []Row, ownership string) []Row {
	var out []Row
	for _, r := range rows {
		if r.Ownership == ownership {
			out = append(out, r)
		}
	}
	return out
}

func splitRow(line string) []string {
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	// Honour escaped pipes inside cells.
	const placeholder = "\x00"
	line = strings.ReplaceAll(line, `\|`, placeholder)
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(strings.ReplaceAll(parts[i], placeholder, "|"))
	}
	return parts
}

func isSeparator(cells []string) bool {
	for _, c := range cells {
		if strings.Trim(c, "-: ") != "" {
			return false
		}
	}
	return true
}

func strip(s string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(s), "`"))
}

func normaliseOwnership(s string) (string, error) {
	v := strings.ToLower(strip(s))
	switch {
	case strings.HasPrefix(v, ProjectOwned):
		return ProjectOwned, nil
	case strings.HasPrefix(v, Shared):
		return Shared, nil
	case strings.HasPrefix(v, Manual):
		return Manual, nil
	}
	return "", fmt.Errorf("unknown ownership %q (want project-owned, shared or manual)", s)
}
