/* A sqlite3-backed seen database. */

package main

import (
    "database/sql"
    "encoding/base64"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "reflect"
    "time"
    _ "github.com/mattn/go-sqlite3"
    )

type SeenDb struct {
    Db *sql.DB
    E *Edition // My current edition

    // For re-encrypting the database when done:
    Enc Encrypt
    TempFile string
    Filename string
}

func (d *SeenDb) Update(filename string, mtimeNow time.Time, getHash func() ([]byte, error)) (updated bool, err error) {
    mtimeNowUnix := mtimeNow.Unix()

    tx, err := d.Db.Begin()
    if err != nil {
        return false, err
    }

    defer tx.Commit()

    // Find the most recent entry for this file:
    var mtimeThenUnix int64
    hashStr := ""
    err = func() error {
        rows, err := tx.Query(
            `select mtime, hash from files
            where filename=?
            order by mtime desc`, filename)
        if err != nil {
            return err
        }

        defer rows.Close()
        if rows.Next() {
            err = rows.Scan(&mtimeThenUnix, &hashStr)
        }

        return err
    }()

    if err != nil {
        return false, err
    }

    var hashThen, hashNow []byte
    if hashStr != "" {
        // There is an entry for this file.
        // If this file is up to date, we clearly don't
        // need a new edition:
        if mtimeNowUnix <= mtimeThenUnix {
            return false, nil
        }

        // Check the hashes; we only need a new edition if
        // the hash has changed
        hashThen, err = base64.StdEncoding.DecodeString(hashStr)
        if err != nil {
            return false, err
        }
    }

    hashNow, err = getHash()
    if err != nil {
        return false, err
    }

    if reflect.DeepEqual(hashNow, hashThen) {
        return false, nil
    }

    // If we got here, we do need a new edition,
    // insert it:
    _, err = tx.Exec(
        `insert into files values (?, ?, ?, ?)`,
        filename,
        d.E.Unix(),
        mtimeNowUnix,
        base64.StdEncoding.EncodeToString(hashNow))
    return true, err
}

func (d *SeenDb) RemoveEdition(edition Edition) (err error) {
    _, err = d.Db.Exec(
        `delete from files where edition=?`,
        edition.Unix())
    return err
}

func (d *SeenDb) Close() error {
    // Always make sure we delete the temp file:
    defer os.Remove(d.TempFile)

    // Close the database
    dbErr := d.Db.Close()

    // Re-encrypt the database file
    f, err := os.Open(d.TempFile)
    if err != nil {
        return err
    }
    defer f.Close()

    cipher, err := os.Create(d.Filename)
    if err != nil {
        return err
    }
    defer cipher.Close()

    plain, err := d.Enc.WrapWriter(cipher)
    if err != nil {
        return err
    }
    defer plain.Close()

    _, err = io.Copy(plain, f)
    if err != nil {
        return err
    }

    return dbErr
}

// Extracts the db into a temporary file, returning
// the file path.
func extractDb(filename string, encrypt Encrypt) (tempFile string, err error) {
    f, err := ioutil.TempFile("", "seen_db")
    if err != nil {
        return "", err
    }

    tempFile = f.Name()
    defer func() {
        f.Close()
        if err != nil {
            os.Remove(tempFile)
            tempFile = ""
        }
    }()

    if cipher, cipherErr := os.Open(filename); cipherErr == nil {
        fmt.Printf("Wrapping existing file...\n")
        plain, err := encrypt.WrapReader(cipher)
        if err == nil {
            _, err = io.Copy(f, plain)
        }
    } else {
        fmt.Printf("Creating new file... %s\n", tempFile)
        // Just remove that temporary file, so that the
        // db can create a new one in its place:
        // (yeah, yeah, this process isn't quite secure)
        os.Remove(tempFile)
    }

    return tempFile, err
}

func NewSeenDb(filename string, encrypt Encrypt, edition *Edition) (seenDb *SeenDb, err error) {
    // Read the database out into a temporary file:
    tempFile, err := extractDb(filename, encrypt)
    if err != nil {
        return nil, err
    }

    defer func() {
        if err != nil {
            os.Remove(tempFile)
        }
    }()

    db, err := sql.Open("sqlite3", tempFile)
    if err != nil {
        return nil, err
    }

    // Create table if not already there, ignoring result
    _, err = db.Exec(
        `create table files(
        filename text,
        edition integer,
        mtime integer,
        hash text,
        primary key (filename, edition))`)
    if err != nil {
        fmt.Printf("%s\n", err.Error())
    }

    return &SeenDb{db, edition, encrypt, tempFile, filename}, nil
}

