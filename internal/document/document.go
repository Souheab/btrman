package document

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"btrman/internal/manual"
)

type LineKind int

const (
	LineNormal LineKind = iota
	LineHeading
	LineOption
	LineExample
)

type Line struct {
	Text         string
	Kind         LineKind
	SectionIndex int
}

type Section struct {
	Title     string
	StartLine int
}

type RelatedLink struct {
	Ref manual.PageRef
}

type Document struct {
	Ref      manual.PageRef
	Title    string
	Lines    []Line
	Sections []Section
	Related  []RelatedLink
}

func Parse(ref manual.PageRef, text string) Document {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "")
	rawLines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(rawLines) == 1 && rawLines[0] == "" {
		rawLines = nil
	}

	doc := Document{Ref: ref, Title: ref.String()}
	currentSection := -1
	for i, raw := range rawLines {
		if IsHeading(raw) {
			doc.Sections = append(doc.Sections, Section{Title: strings.TrimSpace(raw), StartLine: i})
			currentSection = len(doc.Sections) - 1
		}
		kind := classifyLine(raw, currentSection, doc.Sections)
		doc.Lines = append(doc.Lines, Line{Text: raw, Kind: kind, SectionIndex: currentSection})
	}

	if len(doc.Sections) == 0 && len(doc.Lines) > 0 {
		doc.Sections = append(doc.Sections, Section{Title: "DOCUMENT", StartLine: 0})
		for i := range doc.Lines {
			doc.Lines[i].SectionIndex = 0
		}
	}

	if title := firstNonEmptyLine(rawLines); title != "" {
		doc.Title = strings.TrimSpace(title)
	}
	for _, link := range ExtractRelated(rawLines) {
		if link.Ref.Key() == ref.Key() {
			continue
		}
		doc.Related = append(doc.Related, link)
	}
	return doc
}

func IsHeading(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || len(trimmed) > 72 {
		return false
	}
	if leadingWhitespace(line) > 4 {
		return false
	}
	hasLetter := false
	for _, r := range trimmed {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
			if unicode.IsLower(r) {
				return false
			}
		case unicode.IsDigit(r), r == ' ', r == '_', r == '-', r == '/', r == '&':
			continue
		default:
			return false
		}
	}
	return hasLetter
}

func (d Document) PlainLines() []string {
	lines := make([]string, len(d.Lines))
	for i, line := range d.Lines {
		lines[i] = line.Text
	}
	return lines
}

func (d Document) SectionForLine(line int) int {
	if len(d.Sections) == 0 {
		return -1
	}
	idx := sort.Search(len(d.Sections), func(i int) bool {
		return d.Sections[i].StartLine > line
	}) - 1
	if idx < 0 {
		return 0
	}
	return idx
}

func (d Document) SectionStart(index int) int {
	if index < 0 || index >= len(d.Sections) {
		return 0
	}
	return d.Sections[index].StartLine
}

func classifyLine(line string, currentSection int, sections []Section) LineKind {
	if IsHeading(line) {
		return LineHeading
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return LineNormal
	}
	if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "--") || optionLineRE.MatchString(trimmed) {
		return LineOption
	}
	if currentSection >= 0 && currentSection < len(sections) {
		section := sections[currentSection].Title
		if section == "EXAMPLES" || section == "SYNOPSIS" {
			if leadingWhitespace(line) >= 4 {
				return LineExample
			}
		}
	}
	return LineNormal
}

var optionLineRE = regexp.MustCompile(`^-[A-Za-z0-9?],\s+--?[A-Za-z0-9][A-Za-z0-9-]*|^--?[A-Za-z0-9][A-Za-z0-9-]*`)
var relatedRE = regexp.MustCompile(`\b([A-Za-z0-9_.+:/\[\]-]+)\(([^)\s]+)\)`)

func ExtractRelated(lines []string) []RelatedLink {
	seen := map[string]struct{}{}
	var related []RelatedLink
	for _, line := range lines {
		for _, match := range relatedRE.FindAllStringSubmatch(line, -1) {
			ref := manual.PageRef{Name: match[1], Section: match[2]}
			key := ref.Key()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			related = append(related, RelatedLink{Ref: ref})
		}
	}
	return related
}

func firstNonEmptyLine(lines []string) string {
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			return line
		}
	}
	return ""
}

func leadingWhitespace(line string) int {
	count := 0
	for _, r := range line {
		if r != ' ' && r != '\t' {
			break
		}
		count++
	}
	return count
}
