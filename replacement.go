package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Replacement interface {
	Replace(string) string
}

type ReplStart struct {
	Archived    string
	Replacement string
}

func (r *ReplStart) Replace(path string) string {
	if strings.HasPrefix(path, r.Archived) {
		return strings.Replace(path, r.Archived, r.Replacement, 1)
	} else {
		return path
	}
}

type ReplAny struct {
	Archived    string
	Replacement string
}

func (r *ReplAny) Replace(path string) string {
	return strings.Replace(path, r.Archived, r.Replacement, 1)
}

type ReplAll struct {
	Archived    string
	Replacement string
}

func (r *ReplAll) Replace(path string) string {
	return strings.Replace(path, r.Archived, r.Replacement, -1)
}

type Replacements struct {
	R []Replacement
}

func (r *Replacements) addReplacements(param string, makeRepl func(string, string) Replacement) (err error) {
	sep := fmt.Sprintf("%c", os.PathListSeparator)
	flat := strings.Split(param, sep)
	if len(flat) == 1 {
		// No replacements
		return
	} else if (len(flat) % 2) != 0 {
		err = errors.New(fmt.Sprintf("Bad replacements list : %s\n", param))
	} else {
		for i := 0; i < len(flat); i += 2 {
			r.R = append(r.R, makeRepl(flat[i], flat[i+1]))
		}
	}

	return
}

func (r *Replacements) AddReplStart(param string) (err error) {
	return r.addReplacements(param, func(archived string, replacement string) Replacement {
		return &ReplStart{archived, replacement}
	})
}

func (r *Replacements) AddReplAny(param string) (err error) {
	return r.addReplacements(param, func(archived string, replacement string) Replacement {
		return &ReplAny{archived, replacement}
	})
}

func (r *Replacements) AddReplAll(param string) (err error) {
	return r.addReplacements(param, func(archived string, replacement string) Replacement {
		return &ReplAll{archived, replacement}
	})
}

func (r *Replacements) Replace(path string) string {
	for i := 0; i < len(r.R); i++ {
		path = r.R[i].Replace(path)
	}

	return path
}
