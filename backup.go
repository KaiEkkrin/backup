/* Describes a backup as parsed from the jobs file. */

package main

import (
    "encoding/json"
    "fmt"
    "os"
    )

type Job struct {
    // The base file name for tar and db files that get
    // created.
    BaseName string

    // The path to archive.
    Path string

    // Regular expression strings to exclude.
    // TODO Actually compile these into regexps, etc.
    // TODO Or: make it globs instead?  More intuitive?
    Excludes []string
}

func RunBackup(jobPath string) (err error) {
    // Decree an edition for this backup:
    edition := EditionFromNow()
    fmt.Printf("Running backup edition %s\n", edition.String())

    // Read that job file in, and compose a list
    // of backup jobs:
    var runningJobs []*RunningJob

    f, err := os.Open(jobPath)
    if err != nil {
        return err
    }

    defer f.Close()
    decoder := json.NewDecoder(f)
    for decoder.More() {
        var job Job
        err = decoder.Decode(&job)
        if err != nil {
            return err
        }

        runningJobs = append(runningJobs, &RunningJob{job, edition})
    }

    // Compose the list of non-job specific excludes out of
    // all running jobs (all jobs must exclude these!)
    var nonSpecificExcludes []string
    for i := 0; i < len(runningJobs); i++ {
        excl, err := runningJobs[i].GetNonSpecificExcludes()
        if err != nil {
            return err
        }

        nonSpecificExcludes = append(nonSpecificExcludes, excl...)
    }

    // Run all the jobs
    for i := 0; i < len(runningJobs); i++ {
        err = runningJobs[i].Run(nonSpecificExcludes)
        if err != nil {
            return err
        }
    }

    return nil
}

