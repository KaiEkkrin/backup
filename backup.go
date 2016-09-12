/* Describes a backup as parsed from the jobs file. */

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Job struct {
	// The base file name for tar and db files that get
	// created.
	BaseName string

	// The path to archive.
	Path string

	// Path glob strings to exclude.  (Leaf name, or
	// whole path).
	Excludes []string

	// The passphrase to encrypt with.
	Passphrase string
}

func readRunningJobs(jobPath string, edition *Edition) (runningJobs []*RunningJob, err error) {
	f, err := os.Open(jobPath)
	if err != nil {
		return runningJobs, err
	}

	defer f.Close()
	decoder := json.NewDecoder(f)

	// Check eof, because merlin's version of go doesn't have decoder.More()
	finished := false
	for !finished {
		var job Job
		err = decoder.Decode(&job)
		if err != nil {
			if err == io.EOF {
				finished = true
				err = nil
			} else {
				return runningJobs, err
			}
		} else {
			runningJobs = append(runningJobs, &RunningJob{job, edition})
		}
	}

	return runningJobs, err
}

func RunBackup(jobPath string) (err error) {
	// Decree an edition for this backup:
	edition := EditionFromNow()
	fmt.Printf("Running backup edition %s\n", edition.String())

	// Read that job file in, and compose a list
	// of backup jobs:
	runningJobs, err := readRunningJobs(jobPath, edition)
	if err != nil {
		return err
	}

	// Compose the list of non-job specific excludes out of
	// all running jobs (all jobs must exclude these!)
	var nonSpecificExcludes []string
	for i := 0; i < len(runningJobs); i++ {
		excl, err := runningJobs[i].GetNonSpecificExcludes()
		if err != nil {
			return err
		}

		for j := 0; j < len(excl); j++ {
			absExcl, err := filepath.Abs(excl[j])
			if err != nil {
				return err
			}

			nonSpecificExcludes = append(nonSpecificExcludes, absExcl)
		}
	}

	// Run all the jobs
	for i := 0; i < len(runningJobs); i++ {
		encrypt := NewEncryptOpenpgp(runningJobs[i].J.Passphrase)
		err = runningJobs[i].DoBackup(nonSpecificExcludes, encrypt)
		if err != nil {
			return err
		}
	}

	return nil
}

func RunRestore(jobPath string, prefix string) (err error) {
	// We don't need an edition here:
	runningJobs, err := readRunningJobs(jobPath, nil)
	if err != nil {
		return err
	}

	// Run all the jobs
	for i := 0; i < len(runningJobs); i++ {
		encrypt := NewEncryptOpenpgp(runningJobs[i].J.Passphrase)
		err = runningJobs[i].DoRestore(prefix, encrypt)
		if err != nil {
			return err
		}
	}

	return nil
}
