package main

import (
	"path/filepath"
)

type Filter interface {
	// Tests whether to include a path.
	Include(string) bool

	// Only applies to include filters:
	// add a new pattern to the include list.
	// Returns true if something was added,
	// else false.
	AddInclude(string) bool
}

// Tests a path and all its roots, returning as
// soon as a test function call returns true.
func testPathRoots(path string, test func(string) bool, base bool) bool {
	if test(path) {
		return true
	}

	parent, _ := filepath.Split(path)
	if len(parent) > 0 {
		// Really important -- chop the trailing
		// separator character (otherwise filepath.Split()
		// won't manage to split recursively)
		return testPathRoots(parent[:len(parent)-1], test, base)
	} else {
		return base
	}
}

// All our include patterns should go in the same
// include filter, so that we can accept as
// soon as any one of them matches.
type IncludeFilter struct {
	Patterns []string
}

func (f *IncludeFilter) includeInternal(path string) bool {
	for i := 0; i < len(f.Patterns); i++ {
		if testPathRoots(path, func(p string) bool {
			included, err := filepath.Match(f.Patterns[i], p)
			if err != nil {
				panic(err)
			}

			return included
		}, false) {
			return true
		}
	}

	return false
}

func (f *IncludeFilter) Include(path string) bool {
	if f.includeInternal(path) {
		return true
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}

	return f.includeInternal(absPath)
}

func (f *IncludeFilter) AddInclude(pattern string) bool {
	f.Patterns = append(f.Patterns, pattern)
	return true
}

// The exclude filter is one file at a time:
type ExcludeFilter struct {
	Pattern string
}

func (f *ExcludeFilter) includeInternal(path string) bool {
	return !testPathRoots(path, func(p string) bool {
		excluded, err := filepath.Match(f.Pattern, p)
		if err != nil {
			panic(err)
		}

		return excluded
	}, false)
}

func (f *ExcludeFilter) Include(path string) bool {
	if !f.includeInternal(path) {
		return false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}

	if !f.includeInternal(absPath) {
		return false
	}

	// Also test leaf name:
	_, leaf := filepath.Split(path)
	return f.includeInternal(leaf)
}

func (f *ExcludeFilter) AddInclude(pattern string) bool {
	// Do nothing.
	return false
}

type Filters struct {
	F []Filter
}

func (f *Filters) AddIncludeToExisting(pattern string) bool {
	if len(pattern) > 0 {
		// This goes in all applicable filters:
		added := false
		for i := 0; i < len(f.F); i++ {
			if f.F[i].AddInclude(pattern) {
				added = true
			}
		}

		return added
	} else {
		// Handle the blank pattern case --
		// this is irrelevant
		return true
	}
}

func (f *Filters) AddInclude(pattern string) bool {
	if !f.AddIncludeToExisting(pattern) {
		// We need a new include filter here.
		f.F = append(f.F, &IncludeFilter{[]string{pattern}})
	}

	return true
}

func (f *Filters) AddExclude(pattern string) {
	if len(pattern) > 0 {
		f.F = append(f.F, &ExcludeFilter{pattern})
	}
}

func (f *Filters) WithIncludes(patterns []string) *Filters {
	withFilters := &Filters{f.F[:]}
	for i := 0; i < len(patterns); i++ {
		withFilters.AddInclude(patterns[i])
	}

	return withFilters
}

func (f *Filters) WithExcludes(patterns []string) *Filters {
	withFilters := &Filters{f.F[:]}
	for i := 0; i < len(patterns); i++ {
		withFilters.AddExclude(patterns[i])
	}

	return withFilters
}

func (f *Filters) Include(path string) bool {
	//fmt.Printf("Testing against %d filters\n", len(f.F))
	for i := 0; i < len(f.F); i++ {
		if !f.F[i].Include(path) {
			//fmt.Printf("Filter %d (%q) says no\n", i, f.F[i])
			return false
		}
	}

	return true
}
