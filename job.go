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
    //Excludes []string
}

func RunJob(jobPath string) (err error) {
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
        fmt.Printf("Found job %s with path %s\n", job.BaseName, job.Path)
    }

    return nil
}

