package main

import (
    "flag"
    "fmt"
    "os"
    )

func main() {
    /* Our argument will be a json file that describes
     * the backup job(s) to run.
     * That file itself is an encoding of the Job
     * structure. (backup.go)
     * TODO : Support:
     * - Removing old editions?
     * - Excludes in restore?
     * - Extra excludes on the command line (useful for restore)?
     * - Decrypt only (producing the .tar.gz)?
     * - Log file and stats printed?
     */
    backup := flag.Bool("backup", false, "Set this to do a backup")
    restore := flag.Bool("restore", false, "Set this to do a restore")
    jobs := flag.String("job", "backup.json", "Json file describing the backup job")
    prefix := flag.String("prefix", "", "Optional restore prefix")
    flag.Parse()

    if *backup == *restore {
        fmt.Printf("Must specify either backup or restore\n")
        os.Exit(1)
    }

    var err error
    if *backup {
        err = RunBackup(*jobs)
    } else {
        err = RunRestore(*jobs, *prefix)
    }

    if err != nil {
        fmt.Printf("%s\n", err.Error())
        os.Exit(1)
    } else {
        os.Exit(0)
    }
}

