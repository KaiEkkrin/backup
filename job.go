/* A running backup job.
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
)

const (
	ArchiveSuffix  = ".tar.kblob"
	DbSuffix       = "_seen.db.kblob"
	Unpack_Test    = 0
	Unpack_Restore = 1
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
		if (infos[i].Mode() & os.ModeType) == 0 {
			filename := infos[i].Name()

			// TODO Case sensitivity (or not).  For now,
			// I'm case sensitive.
			if strings.HasPrefix(filename, r.J.BaseName+"_") && strings.HasSuffix(filename, ArchiveSuffix) {
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

func (r *RunningJob) DoBackup(filter *Filters, encrypt Encrypt) (err error) {
	// TODO Proper log file and summary on stdout
	fmt.Printf("Running backup %s ...\n", r.J.BaseName)

	// Construct the full filter (out of the general ones
	// and the specific ones to this job)
	fullFilter := filter.WithExcludes(r.J.Excludes)

	// If we have an include filter, add the root path of
	// the job to it, otherwise everything will be
	// excluded and that would be bad :)
	fullFilter.AddIncludeToExisting(r.J.Path)

	// Open up the database:
	fmt.Printf("Opening database %s\n", r.GetDbFilename())
	seenDb, err := NewSeenDb(r.GetDbFilename(), encrypt, r.E)
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

	archPlain, err := encrypt.WrapWriter(archFile)
	if err != nil {
		return err
	}
	defer archPlain.Close()

	archGz := gzip.NewWriter(archPlain)
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

		if !fullFilter.Include(path) {
			fmt.Printf("%s : Excluded\n", path)
			return getIgnoreValue()
		}

		// Work out whether to include it in the archive.
		mode := info.Mode()
		if (mode & os.ModeTemporary) != 0 {
			fmt.Printf("%s : Skipping temporary file\n", path)
		} else if (mode & os.ModeDevice) != 0 {
			fmt.Printf("%s : Skipping device file\n", path)
		} else if (mode & os.ModeNamedPipe) != 0 {
			fmt.Printf("%s : Skipping pipe file\n", path)
		} else if (mode & os.ModeSocket) != 0 {
			fmt.Printf("%s : Skipping socket file\n", path)
		} else if (mode & os.ModeType) == 0 {
			// This is a regular file; look it up against
			// the database
			err = seenDb.Update(absPath, info.ModTime(), func() ([]byte, error) {
				return getHash(absPath)
			}, func() (err error) {
				return r.backupFile(path, info, mode, archTar)
			})

			if err != nil {
				// Report errors and continue, to do a best-effort backup.
				fmt.Printf("%s : %s\n", path, err.Error())
			}
		} else {
			// This is something like a directory.
			// It doesn't go in the database, but it does
			// go in the tar file:
			err = r.backupFile(path, info, mode, archTar)
			if err != nil {
				fmt.Printf("%s : %s\n", path, err.Error())
			}
		}

		return nil
	})

	// TODO : Search the db for files that no longer
	// exist and blank entries?

	return err
}

func (r *RunningJob) backupFile(path string, info os.FileInfo, mode os.FileMode, archTar *tar.Writer) (err error) {

	// If it's a symlink, read the link target:
	link := ""
	if (mode & os.ModeSymlink) != 0 {
		link, err = os.Readlink(path)
		if err != nil {
			return
		}
	}

	// Write the header.
	// Make sure I fix the name, which is defaulted
	// to the leaf name here:
	hdr, err := tar.FileInfoHeader(info, link)
	if err != nil {
		return
	}

	tarPath := filepath.Clean(path)
	if info.IsDir() {
		tarPath = fmt.Sprintf("%s%c", tarPath, os.PathSeparator)
	}

	hdr.Name = tarPath

	// ...and the uid and gid;
	// this is platform specific
	AssignUserIds(info, hdr)

	// ...and the modification time.
	// Note that it looks like the AccessTime field doesn't work,
	// so I'm ignoring it
	hdr.ModTime = info.ModTime()

	err = archTar.WriteHeader(hdr)
	if err != nil {
		return
	}

	// If this is a real file, write the contents:
	if (mode & os.ModeType) == 0 {
		err = copyInto(path, archTar)
		if err != nil {
			return
		}
	}

	return
}

// `what' should be one of: Unpack_Test, Unpack_Restore
func (r *RunningJob) DoUnpack(filter Filter, prefix string, repl Replacement, encrypt Encrypt, what int) (err error) {
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

	var unpackFile func(string, *tar.Header, io.Reader) error
	if what == Unpack_Test {
		unpackFile = testFile
	} else if what == Unpack_Restore {
		unpackFile = restoreFile
	}

	for i := 0; i < archives.Len(); i++ {
		err = unpackArchive(archives.GetName(i), filter, prefix, repl, encrypt, unpackFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func unpackArchive(archive string, filter Filter, prefix string, repl Replacement, encrypt Encrypt, unpackFile func(string, *tar.Header, io.Reader) error) (err error) {
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

	archPlain, err := encrypt.WrapReader(archFile)
	if err != nil {
		return err
	}

	archGz, err := gzip.NewReader(archPlain)
	if err != nil {
		return err
	}
	defer archGz.Close()

	archTar := tar.NewReader(archGz)

	errorCount := 0
	var readErr error
	for readErr == nil {
		includeFile := false

		var hdr *tar.Header
		hdr, readErr = archTar.Next()
		if readErr != nil && readErr != io.EOF {
			return readErr
		} else if readErr == nil {
			includeFile = filter.Include(hdr.Name)
		}

		if includeFile {
			replacedPath := repl.Replace(hdr.Name)
			restoredPath := filepath.Join(prefix, replacedPath)
			restoreErr := unpackFile(restoredPath, hdr, archTar)
			if restoreErr != nil {
				fmt.Printf("%s : %s\n", restoredPath, restoreErr.Error())
				errorCount += 1
			}
		}
	}

	if errorCount > 0 {
		err = errors.New(fmt.Sprintf("Finished with %d errors", errorCount))
	}

	return err
}

// File unpack functions ...

func testFile(restoredPath string, hdr *tar.Header, archTar io.Reader) (err error) {
	fmt.Printf("%s\n", restoredPath)
	return nil
}

func restoreFile(restoredPath string, hdr *tar.Header, archTar io.Reader) (err error) {
	info := hdr.FileInfo()
	mode := info.Mode()

	if info.IsDir() {
		err = os.Mkdir(restoredPath, mode.Perm())
	} else if (mode & os.ModeSymlink) != 0 {
		err = os.Symlink(hdr.Linkname, restoredPath)
	} else if (mode & os.ModeType) == 0 {
		// This is a regular file, write its contents
		err = copyOutOf(restoredPath, archTar)
	}

	// Restore this thing's mode, ownership, etc
	if err == nil {
		err = os.Chmod(restoredPath, mode.Perm())
	}

	if err == nil {
		err = RestoreOwnership(restoredPath, hdr)
	}

	if err == nil {
		// I'm not trying to restore the access time, because
		// it seems we can't store it correctly.
		// It's not terribly important anyway :)
		err = os.Chtimes(restoredPath, hdr.ModTime, hdr.ModTime)
	}

	// TODO Extended attributes
	return
}
