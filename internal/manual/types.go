package manual

import (
	"fmt"
	"regexp"
	"strings"
)

var refWithSectionRE = regexp.MustCompile(`^(.+)\(([^()\s]+)\)$`)

// PageRef identifies a manual page by name and, when known, section.
type PageRef struct {
	Name    string `json:"name"`
	Section string `json:"section,omitempty"`
}

// Page is a searchable manual page entry returned by apropos/man -k.
type Page struct {
	Ref         PageRef `json:"ref"`
	Description string  `json:"description,omitempty"`
}

// RawPage is rendered official man-page text before document parsing.
type RawPage struct {
	Ref  PageRef
	Text string
}

func NewPageRef(name, section string) PageRef {
	return PageRef{Name: strings.TrimSpace(name), Section: strings.TrimSpace(section)}
}

func ParsePageRef(input string) (PageRef, bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return PageRef{}, false
	}
	if match := refWithSectionRE.FindStringSubmatch(input); match != nil {
		name := strings.TrimSpace(match[1])
		section := strings.TrimSpace(match[2])
		if name == "" || section == "" {
			return PageRef{}, false
		}
		return PageRef{Name: name, Section: section}, true
	}
	return PageRef{Name: input}, true
}

func (r PageRef) String() string {
	if r.Section == "" {
		return r.Name
	}
	return fmt.Sprintf("%s(%s)", r.Name, r.Section)
}

func (r PageRef) Key() string {
	name := strings.ToLower(strings.TrimSpace(r.Name))
	section := strings.ToLower(strings.TrimSpace(r.Section))
	if section == "" {
		return name
	}
	return name + "(" + section + ")"
}

func (r PageRef) IsZero() bool {
	return strings.TrimSpace(r.Name) == ""
}
