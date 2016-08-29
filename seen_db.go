/* A sqlite3-backed seen database. */

package main

import (
    "database/sql"
    "encoding/base64"
    "fmt"
    "reflect"
    "time"
    _ "github.com/mattn/go-sqlite3"
    )

type SeenDb struct {
    Db *sql.DB
    E *Edition // My current edition
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
    return d.Db.Close()
}

func NewSeenDb(filename string, edition *Edition) (*SeenDb, error) {
    db, err := sql.Open("sqlite3", filename)
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

    return &SeenDb{db, edition}, nil
}

