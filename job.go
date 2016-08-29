/* A running backup job. */

package main

import (
    "archive/tar"
    "compress/gzip"
    "crypto/sha256"
    "fmt"
    "io"
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

func getHash(filename string) (hashBytes []byte, err error) {
    f, err := os.Open(filename)
    if err != nil {
        return hashBytes, err
    }
    defer f.Close()

    h := sha256.New()
    _, err = io.Copy(h, f)
    if err != nil {
        return hashBytes, err
    }

    return h.Sum(hashBytes), nil
}

func copyInto(filename string, writer io.Writer) error {
    f, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer f.Close()

    _, err = io.Copy(writer, f)
    return err
}

func (r *RunningJob) Run(excludes []string) (err error) {
    // TODO Proper log file and summary on stdout
    fmt.Printf("Running backup %s ...\n", r.J.BaseName)

    // Construct the absolute exclude paths:
    var absExcludes []string
    for i := 0; i < len(excludes); i++ {
        absExclude, err := filepath.Abs(excludes[i])
        if err != nil {
            return err
        }

        fmt.Printf("  Excluding: %s\n", absExclude)
        absExcludes = append(absExcludes, absExclude)
    }

    // Open up the database:
    // TODO: Encryption (here and below!)
    fmt.Printf("Opening database %s\n", r.GetDbFilename())
    seenDb, err := NewSeenDb(r.GetDbFilename(), r.E)
    if err != nil {
        return err
    }
    defer seenDb.Close()

    // Open up the new archive:
    fmt.Printf("Opening new archive %s\n", r.GetNewEditionFilename())
    archFile, err := os.Create(r.GetNewEditionFilename())
    if err != nil {
        return err
    }
    defer archFile.Close()

    archGz := gzip.NewWriter(archFile)
    defer archGz.Close()

    archTar := tar.NewWriter(archGz)
    defer archTar.Close()

    // Now we can walk the tree scooping everything.
    // TODO: How to avoid transitioning across filesystems?
    err = filepath.Walk(r.J.Path, func(path string, info os.FileInfo, walkErr error) error {

        getIgnoreValue := func() error {
            if info.IsDir() {
                return filepath.SkipDir
            } else {
                return nil
            }
        }

        // If there was a problem, log it, and probably
        // ignore it:
        if walkErr != nil {
            fmt.Printf("%s : %s\n", path, walkErr.Error())
            return getIgnoreValue()
        }

        // Check whether to skip this.  If it's a directory,
        // we'll skip the whole directory.
        absPath, err := filepath.Abs(path)
        if err != nil {
            return err
        }

        for i := 0; i < len(absExcludes); i++ {
            if absPath == absExcludes[i] {
                fmt.Printf("%s : Excluded\n", path)
                return getIgnoreValue()
            }
        }

        // Work out whether to include it in the archive.
        // We'll 
        includeThis := true
        mode := info.Mode()
        if (mode & os.ModeTemporary) != 0 {
            includeThis = false
            fmt.Printf("%s : Skipping temporary file\n", path)
        } else if (mode & os.ModeDevice) != 0 {
            includeThis = false
            fmt.Printf("%s : Skipping device file\n", path)
        } else if (mode & os.ModeNamedPipe) != 0 {
            includeThis = false
            fmt.Printf("%s : Skipping pipe file\n", path)
        } else if (mode & os.ModeSocket) != 0 {
            includeThis = false
            fmt.Printf("%s : Skipping socket file\n", path)
        } else if (mode & os.ModeType) == 0 {
            // This is a regular file; look it up against
            // the database
            includeThis, err = seenDb.Update(absPath, info.ModTime(), func() ([]byte, error) {
                return getHash(absPath)
            })

            if err != nil {
                // TODO: Failed to read the file.  Remove this
                // edition of it from the database, complain
                // and carry on.
                fmt.Printf("%s : Failed db update : %s\n", path, err.Error())
                return err
            }

            if !includeThis {
                fmt.Printf("%s : Already up to date\n", path)
            }
        }
            
        if includeThis {
            // If it's a symlink, read the link target:
            link := ""
            if (mode & os.ModeSymlink) != 0 {
                link, err = os.Readlink(path)
                if err != nil {
                    fmt.Printf("%s : Failed to follow symlink\n", path)
                    // TODO : Remove this edition from the db,
                    // and carry on
                    return err
                }
            }

            // Write the header.
            // Make sure I fix the name, which is defaulted
            // to the leaf name here:
            hdr, err := tar.FileInfoHeader(info, link)
            if err != nil {
                return err
            }

            tarPath := filepath.Clean(path)
            if info.IsDir() {
                tarPath = fmt.Sprintf("%s%c", tarPath, os.PathSeparator)
            }

            hdr.Name = tarPath

            err = archTar.WriteHeader(hdr)
            if err != nil {
                return err
            }

            // If this is a real file, write the contents:
            if (mode & os.ModeType) == 0 {
                err = copyInto(path, archTar)
                if err != nil {
                    // TODO: remove from the db, continue
                    fmt.Printf("%s : Failed to archive : %s\n", path, err.Error())
                    return err
                }
            }

            fmt.Printf("%s : Archived\n", path)
        }

        return nil
    })

    // TODO : Search the db for files that no longer
    // exist and blank entries?

    return err
}

