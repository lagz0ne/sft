package format

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ANSI — raw codes, no library (à la agent-browser)
const (
	Reset  = "\x1b[0m"
	Bold   = "\x1b[1m"
	Dim    = "\x1b[2m"
	Red    = "\x1b[31m"
	Green  = "\x1b[32m"
	Yellow = "\x1b[33m"
	Cyan   = "\x1b[36m"
)

var (
	TTY      bool
	JSONMode bool
)

func Init(jsonMode bool) {
	JSONMode = jsonMode
	if jsonMode {
		TTY = false
		return
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return
	}
	TTY = fi.Mode()&os.ModeCharDevice != 0
}

func C(code, text string) string {
	if !TTY {
		return text
	}
	return code + text + Reset
}

// --- Status icons ---

func OK(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", C(Green, "✓"), msg)
}

func Warn(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", C(Yellow, "⚠"), msg)
}

func Err(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", C(Red, "✗"), msg)
}

// --- JSON output ---

func JSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

// --- Table output ---

type Column struct {
	Key    string
	Header string
	Bool   bool // render 0/1 as yes/no
	Right  bool // right-align
}

var Queries = map[string][]Column{
	"screens": {
		{Key: "name", Header: "NAME"},
		{Key: "description", Header: "DESCRIPTION"},
	},
	"regions": {
		{Key: "name", Header: "NAME"},
		{Key: "parent_name", Header: "PARENT"},
		{Key: "event_count", Header: "EVENTS", Right: true},
		{Key: "has_states", Header: "STATES", Bool: true},
		{Key: "description", Header: "DESCRIPTION"},
	},
	"events": {
		{Key: "event", Header: "EVENT"},
		{Key: "emitted_by", Header: "FROM"},
		{Key: "handled_at", Header: "HANDLED BY"},
		{Key: "from_state", Header: "FROM STATE"},
		{Key: "to_state", Header: "TO STATE"},
		{Key: "action", Header: "ACTION"},
	},
	"states": {
		{Key: "on_event", Header: "ON EVENT"},
		{Key: "from_state", Header: "FROM"},
		{Key: "to_state", Header: "TO"},
		{Key: "action", Header: "ACTION"},
	},
	"flows": {
		{Key: "name", Header: "NAME"},
		{Key: "on_event", Header: "TRIGGER"},
		{Key: "sequence", Header: "SEQUENCE"},
	},
	"tags": {
		{Key: "tag", Header: "TAG"},
		{Key: "entity_type", Header: "TYPE"},
		{Key: "entity_name", Header: "ENTITY"},
	},
}

func Table(queryName string, rows []map[string]any) {
	if JSONMode || !TTY {
		JSON(rows)
		return
	}

	cols, ok := Queries[queryName]
	if !ok {
		JSON(rows)
		return
	}

	if len(rows) == 0 {
		fmt.Println(C(Dim, "(no results)"))
		return
	}

	// Compute widths
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len(c.Header)
	}
	cells := make([][]string, len(rows))
	for r, row := range rows {
		cells[r] = make([]string, len(cols))
		for i, c := range cols {
			s := fmtCell(row[c.Key], c.Bool)
			cells[r][i] = s
			if len(s) > widths[i] {
				widths[i] = len(s)
			}
		}
	}

	// Cap wide columns
	for i, c := range cols {
		if c.Key == "description" || c.Key == "sequence" {
			if widths[i] > 50 {
				widths[i] = 50
			}
		}
	}

	// Header
	var hdr strings.Builder
	for i, c := range cols {
		if i > 0 {
			hdr.WriteString("  ")
		}
		if c.Right {
			hdr.WriteString(fmt.Sprintf("%*s", widths[i], c.Header))
		} else {
			hdr.WriteString(fmt.Sprintf("%-*s", widths[i], c.Header))
		}
	}
	fmt.Println(C(Dim, hdr.String()))

	// Rows
	for _, row := range cells {
		for i, c := range cols {
			if i > 0 {
				fmt.Print("  ")
			}
			s := row[i]
			if len(s) > widths[i] {
				s = s[:widths[i]-1] + "…"
			}
			if c.Right {
				fmt.Printf("%*s", widths[i], s)
			} else {
				fmt.Printf("%-*s", widths[i], s)
			}
		}
		fmt.Println()
	}
}

func fmtCell(v any, asBool bool) string {
	if v == nil {
		return "—"
	}
	switch val := v.(type) {
	case string:
		if val == "" {
			return "—"
		}
		return val
	case int64:
		if asBool {
			if val > 0 {
				return "yes"
			}
			return "no"
		}
		return fmt.Sprintf("%d", val)
	default:
		s := fmt.Sprintf("%v", val)
		if asBool {
			if s == "1" || s == "true" {
				return "yes"
			}
			return "no"
		}
		if s == "" {
			return "—"
		}
		return s
	}
}

// --- Impact display ---

type Impact struct {
	Entity string `json:"entity"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Detail string `json:"detail,omitempty"`
}

// Impacts renders impact as JSON (stdout) or tree (stderr) for standalone `sft impact`.
func Impacts(entityType, name string, impacts []Impact) {
	if JSONMode || !TTY {
		JSON(impacts)
		return
	}
	impactTree(os.Stderr, entityType, name, impacts)
}

// ImpactInfo renders impact to stderr always (for rm/mv context).
func ImpactInfo(entityType, name string, impacts []Impact) {
	if len(impacts) == 0 {
		return
	}
	impactTree(os.Stderr, entityType, name, impacts)
}

func impactTree(w *os.File, entityType, name string, impacts []Impact) {
	if len(impacts) == 0 {
		fmt.Fprintf(w, "%s %s %s has no dependents\n", C(Green, "✓"), entityType, C(Bold, name))
		return
	}

	fmt.Fprintf(w, "%s %s\n", C(Bold, name), C(Dim, "("+entityType+")"))

	// Group by entity type
	groups := make(map[string][]Impact)
	order := []string{}
	for _, imp := range impacts {
		key := imp.Entity
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], imp)
	}

	for gi, key := range order {
		imps := groups[key]
		last := gi == len(order)-1
		branch := "├──"
		if last {
			branch = "└──"
		}
		branch = C(Dim, branch)

		if len(imps) == 1 {
			fmt.Fprintf(w, "%s %s\n", branch, formatImpactLine(imps[0]))
		} else {
			fmt.Fprintf(w, "%s %s\n", branch, C(Dim, key+"s"))
			for ii, imp := range imps {
				sub := "│   ├──"
				if last {
					sub = "    ├──"
				}
				if ii == len(imps)-1 {
					if last {
						sub = "    └──"
					} else {
						sub = "│   └──"
					}
				}
				sub = C(Dim, sub)
				fmt.Fprintf(w, "%s %s\n", sub, formatImpactLine(imp))
			}
		}
	}
}

func formatImpactLine(imp Impact) string {
	s := imp.Name
	if imp.Detail != "" {
		s += " " + C(Dim, imp.Detail)
	}
	return s
}

// --- Validation display ---

type Finding struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func Findings(findings []Finding) {
	if JSONMode || !TTY {
		JSON(findings)
		return
	}

	errors, warnings := 0, 0
	for _, f := range findings {
		var icon string
		switch f.Severity {
		case "error":
			icon = C(Red, "✗")
			errors++
		case "warning":
			icon = C(Yellow, "⚠")
			warnings++
		}
		rule := C(Dim, f.Rule)
		fmt.Fprintf(os.Stderr, "%s %s  %s\n", icon, rule, f.Message)
	}

	if len(findings) == 0 {
		fmt.Fprintf(os.Stderr, "%s no issues\n", C(Green, "✓"))
	} else {
		fmt.Fprintf(os.Stderr, "\n")
		parts := []string{}
		if errors > 0 {
			parts = append(parts, C(Red, fmt.Sprintf("%d errors", errors)))
		} else {
			parts = append(parts, "0 errors")
		}
		if warnings > 0 {
			parts = append(parts, C(Yellow, fmt.Sprintf("%d warnings", warnings)))
		} else {
			parts = append(parts, "0 warnings")
		}
		fmt.Fprintf(os.Stderr, "%s\n", strings.Join(parts, ", "))
	}
}
