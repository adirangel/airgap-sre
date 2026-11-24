package footprint

import (
	"strings"
	"testing"
)

const sample = `
# Footprint

Some prose | with a pipe.

| ID | Item | Location | Approx. size | Ownership | Removal (PowerShell) |
| --- | --- | --- | --- | --- | --- |
| ` + "`tools-dir`" + ` | Tool binaries | ` + "`.tools/`" + ` | 200 MB | project-owned | ` + "`labctl cleanup`" + ` |
| ` + "`go-mod-cache`" + ` | Go module cache | ` + "`%USERPROFILE%\\go\\pkg\\mod`" + ` | 300 MB | shared (optional — affects other projects) | ` + "`go clean -modcache`" + ` |

| ID | Item | Location | Approx. size | Ownership | Removal (PowerShell) |
| --- | --- | --- | --- | --- | --- |
| ` + "`github-repo`" + ` | Repo | github.com | n/a | manual | ` + "`gh repo delete x \\| more`" + ` |
`

func TestParse(t *testing.T) {
	rows, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d: %+v", len(rows), rows)
	}
	if rows[0].ID != "tools-dir" || rows[0].Ownership != ProjectOwned || rows[0].Removal != "labctl cleanup" {
		t.Errorf("row 0 wrong: %+v", rows[0])
	}
	if rows[1].Ownership != Shared || rows[1].Removal != "go clean -modcache" {
		t.Errorf("row 1 wrong: %+v", rows[1])
	}
	if rows[2].Ownership != Manual || rows[2].Removal != "gh repo delete x | more" {
		t.Errorf("row 2 wrong: %+v", rows[2])
	}
	if got := len(Filter(rows, Shared)); got != 1 {
		t.Errorf("Filter(shared) = %d rows, want 1", got)
	}
}

func TestParseErrors(t *testing.T) {
	header := "| ID | Item | Location | Approx. size | Ownership | Removal |\n| --- | --- | --- | --- | --- | --- |\n"
	cases := map[string]string{
		"bad ownership": header + "| a | b | c | d | mine | e |\n",
		"wrong columns": header + "| a | b | c |\n",
		"duplicate id":  header + "| a | b | c | d | manual | e |\n| a | b | c | d | manual | e |\n",
	}
	for name, in := range cases {
		if _, err := Parse(strings.NewReader(in)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
