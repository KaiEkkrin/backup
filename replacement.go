package main

import (
	"fmt"
	"os"
	"strings"
)

type Replacement struct {
	Archived    string
	Replacement string
}

func buildReplacements(param string) (repl []Replacement) {
	sep := fmt.Sprintf("%c", os.PathListSeparator)
	flat := strings.Split(param, sep)
	if len(flat) == 1 {
		// No replacements
		return
	} else if (len(flat) % 2) != 0 {
		fmt.Printf("Bad replacements list : %s\n", param)
		os.Exit(1)
	}

	for i := 0; i < len(flat); i += 2 {
		repl = append(repl, Replacement{flat[i], flat[i+1]})
	}

	return
}

type ReplStart struct {
	Repl []Replacement
}

func (r *ReplStart) Replace(path string) string {
	for i := 0; i < len(r.Repl); i++ {
		if strings.HasPrefix(path, r.Repl[i].Archived) {
			return strings.Replace(path, r.Repl[i].Archived, r.Repl[i].Replacement, 1)
		}
	}

	return path
}

func NewReplStart(param string) *ReplStart {
	return &ReplStart{buildReplacements(param)}
}
