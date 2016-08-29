/* A running backup job. */

package main

import (
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "strings"
    )

const (
    ArchiveSuffix = ".tar.gz"
    DbSuffix = "_seen.db"
    )

type RunningJob struct {
    J Job
    E *Edition
}

func (r *RunningJob) GetDir() string {
    return filepath.Dir(r.J.BaseName)
}

func (r *RunningJob) GetDbFilename() string {
    return fmt.Sprintf("%s%s", r.J.BaseName, DbSuffix)
}

func (r *RunningJob) GetNewEditionFilename() string {
    return fmt.Sprintf("%s_%s%s", r.J.BaseName, r.E.String(), ArchiveSuffix)
}

func (r *RunningJob) GetOldEditionFilenames() (names []string, err error) {
    infos, err := ioutil.ReadDir(r.GetDir())
    if err != nil {
        return names, err
    }

    for i := 0; i < len(infos); i++ {
        if ((infos[i].Mode() & os.ModeType) == 0) {
            filename := infos[i].Name()

            // TODO Case sensitivity (or not).  For now,
            // I'm case sensitive.
            if strings.HasPrefix(filename, r.J.BaseName) && strings.HasSuffix(filename, ArchiveSuffix) {
                names = append(names, filename)
            }
        }
    }

    return names, nil
}

func (r *RunningJob) GetNonSpecificExcludes() (names []string, err error) {
    names, err = r.GetOldEditionFilenames()
    if err == nil {
        names = append(names, r.GetNewEditionFilename(), r.GetDbFilename())
    }

    return names, err
}

func (r *RunningJob) Run(excludes []string) (err error) {
    // TODO TODO :)
    return err
}

