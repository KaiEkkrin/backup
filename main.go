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
     * TODO : Support removing old editions, etc.
     */
    jobs := flag.String("job", "backup.json", "Json file describing the backup job")
    flag.Parse()

    err := RunBackup(*jobs)
    if err != nil {
        fmt.Printf("%s\n", err.Error())
        os.Exit(1)
    } else {
        os.Exit(0)
    }
}

