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

