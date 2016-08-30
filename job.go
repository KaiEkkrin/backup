/* A running backup job.
 * TODO Replace the gzip with just the openpgp compression?
*/

package main

import (
    "archive/tar"
    "compress/gzip"
    "crypto/sha256"
    "errors"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "golang.org/x/crypto/openpgp"
    "golang.org/x/crypto/openpgp/armor"
    )

const (
    ArchiveSuffix = ".tar.gpg"
    DbSuffix = "_seen.db"
    EncryptionType = "PGP SIGNATURE"
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

func (r *RunningJob) GetOldEditionFilenames() (names *ArchiveNames, err error) {
    dir := r.GetDir()
    infos, err := ioutil.ReadDir(dir)
    if err != nil {
        return nil, err
    }

    names = &ArchiveNames{r.J.BaseName + "_", ArchiveSuffix, []ArchiveName{}}
    for i := 0; i < len(infos); i++ {
        if ((infos[i].Mode() & os.ModeType) == 0) {
            filename := infos[i].Name()

            // TODO Case sensitivity (or not).  For now,
            // I'm case sensitive.
            if strings.HasPrefix(filename, r.J.BaseName + "_") && strings.HasSuffix(filename, ArchiveSuffix) {
                err = names.Append(dir, filename)
                if err != nil {
                    return nil, err
                }
            }
        }
    }

    return names, nil
}

func (r *RunningJob) GetNonSpecificExcludes() (names []string, err error) {
    archiveNames, err := r.GetOldEditionFilenames()
    if err == nil {
        for i := 0; i < archiveNames.Len(); i++ {
            names = append(names, archiveNames.GetName(i))
        }

        names = append(names, r.GetNewEditionFilename(), r.GetDbFilename())
    }

    return names, err
}

func (r *RunningJob) GetPassphrase() []byte {
    return []byte(r.J.Passphrase)
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

func copyOutOf(filename string, reader io.Reader) error {
    f, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer f.Close()

    _, err = io.Copy(f, reader)
    return err
}

func (r *RunningJob) DoBackup(excludes []string) (err error) {
    // TODO Proper log file and summary on stdout
    fmt.Printf("Running backup %s ...\n", r.J.BaseName)

    // Construct all excludes (out of the general ones
    // and the specific ones to this job)
    allExcludes := append(excludes, r.J.Excludes...)

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

    archArmor, err := armor.Encode(archFile, EncryptionType, nil)
    if err != nil {
        return err
    }
    defer archArmor.Close()

    archGpg, err := openpgp.SymmetricallyEncrypt(archArmor, r.GetPassphrase(), nil, nil)
    if err != nil {
        return err
    }
    defer archGpg.Close() 

    archGz := gzip.NewWriter(archGpg)
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

        for i := 0; i < len(allExcludes); i++ {
            // Check the whole path
            excluded, err := filepath.Match(allExcludes[i], absPath)
            if err != nil {
                return err
            }

            // If that didn't match, check the leaf name
            if !excluded {
                excluded, err = filepath.Match(allExcludes[i], info.Name())
                if err != nil {
                    return err
                }
            }

            if excluded {
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

            // ...and the uid and gid...
            // TODO TODO (unix only)
            // TODO also atime and mtime

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

func (r *RunningJob) DoRestore(prefix string) (err error) {
    // TODO Again, proper log file and summary on stdout
    fmt.Printf("Running restore %s...\n", r.J.BaseName)

    // For now, we will simply unpack everything in order.
    // TODO : Cross reference with the database, avoid
    // unpacking old stuff?  (Allows delete?)
    archives, err := r.GetOldEditionFilenames()
    if err != nil {
        return err
    }

    sort.Sort(archives)

    passphrase := r.GetPassphrase()
    for i := 0; i < archives.Len(); i++ {
        err = restoreArchive(archives.GetName(i), prefix, passphrase)
        if err != nil {
            return err
        }       
    }

    return nil
}

func restoreArchive(archive string, prefix string, passphrase []byte) (err error) {
    fmt.Printf("Restoring %s...\n", archive)

    if len(prefix) > 0 {
        err = os.MkdirAll(prefix, 0777)
        if err != nil {
            return err
        }
    }

    archFile, err := os.Open(archive)
    if err != nil {
        return err
    }
    defer archFile.Close()

    archArmorBlock, err := armor.Decode(archFile)
    if err != nil {
        return err
    }

    if archArmorBlock.Type != EncryptionType {
        return errors.New(fmt.Sprintf("Unexpected encryption type %s", archArmorBlock.Type))
    }

    prompt := func(keys []openpgp.Key, symmetric bool) (pp []byte, err error) {
        pp = passphrase
        if !symmetric {
            err = errors.New("Expected symmetric encryption")
        }

        return pp, err
    }

    archMd, err := openpgp.ReadMessage(archArmorBlock.Body, nil, prompt, nil)
    if err != nil {
        return err
    }

    // TODO Verification?
    fmt.Printf("Opened message. Encrypted %q, signed %q\n", archMd.IsEncrypted, archMd.IsSigned)

    archGz, err := gzip.NewReader(archMd.UnverifiedBody)
    if err != nil {
        return err
    }
    defer archGz.Close()

    archTar := tar.NewReader(archGz)

    var readErr error
    for readErr == nil {
        var hdr *tar.Header
        hdr, readErr = archTar.Next()
        if readErr != nil && readErr != io.EOF {
            return readErr
        } else if readErr == nil {

            restoredPath := filepath.Join(prefix, hdr.Name)
            info := hdr.FileInfo()
            mode := info.Mode()
            
            wroteSomething := false
            if info.IsDir() {
                err = os.Mkdir(restoredPath, mode.Perm())
                if err != nil {
                    fmt.Printf("%s : %s\n", restoredPath, err.Error())
                } else {
                    wroteSomething = true
                }
            } else if (mode & os.ModeSymlink) != 0 {
                err = os.Symlink(hdr.Linkname, restoredPath)
                if err != nil {
                    fmt.Printf("%s : %s\n", restoredPath, err.Error())
                } else {
                    wroteSomething = true
                }
            } else if (mode & os.ModeType) == 0 {
                // This is a regular file, write its contents
                err = copyOutOf(restoredPath, archTar)
                if err != nil {
                    fmt.Printf("%s : %s\n", restoredPath, err.Error())
                } else {
                    wroteSomething = true
                }
            }
        
            if wroteSomething {
                // Restore this thing's mode, ownership, etc
                err = os.Chmod(restoredPath, mode.Perm())
                if err != nil {
                    fmt.Printf("%s : %s\n", restoredPath, err.Error())
                }
                
                // TODO uid and gid (unix only)
                // TODO atime and mtime
            }
        }
    }

    return nil
}

