/* Describes a job as parsed from the jobs file. */

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
    Excludes []string
}

func RunBackup(jobPath string) (err error) {
    // Decree an edition for this backup:
    edition := EditionFromNow()
    fmt.Printf("Running backup edition %s\n", edition.String())

    // Read that job file in:
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

        // TODO run job here :)
        fmt.Printf("%s : Backing up %s ...\n", job.BaseName, job.Path)
        if len(job.Excludes) > 0 {
            for i := 0; i < len(job.Excludes); i++ {
                fmt.Printf("%s : Excluding %s\n", job.BaseName, job.Excludes[i])
            }
        }
    }

    return nil
}

