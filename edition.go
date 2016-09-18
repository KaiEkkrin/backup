/* Represents the particular point a file was backed up.
 * Each individual backup archive should have a unique
 * Edition.
 */

package main

import (
	"time"
)

const (
	/* This time string is meant to be included in filenames,
	 * so we avoid bad characters like :
	 */
	TimeFormat = "2006-01-02 15-04-05 MST"
)

type Edition struct {
	When time.Time
}

func (e *Edition) String() string {
	return e.When.Format(TimeFormat)
}

func (e *Edition) Unix() int64 {
	return e.When.Unix()
}

func EditionFromNow() *Edition {
	return &Edition{time.Now()}
}

func EditionFromString(str string) (*Edition, error) {
	when, err := time.Parse(TimeFormat, str)
	if err != nil {
		return nil, err
	} else {
		return &Edition{when}, nil
	}
}

func EditionFromUnix(unix int64) *Edition {
	return &Edition{time.Unix(unix, 0)}
}

// For sorting them:

type SortedEditions struct {
	E []*Edition
}

func (s *SortedEditions) Len() int {
	return len(s.E)
}

func (s *SortedEditions) Less(i, j int) bool {
	return s.E[i].When.Before(s.E[j].When)
}

func (s *SortedEditions) Swap(i, j int) {
	tmp := s.E[i]
	s.E[i] = s.E[j]
	s.E[j] = tmp
}

func (s *SortedEditions) At(i int) *Edition {
	return s.E[i]
}

func (s *SortedEditions) Append(edition *Edition) {
	s.E = append(s.E, edition)
}
