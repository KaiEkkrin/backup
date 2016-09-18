/* A sqlite3-backed seen database. */

package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"time"
)

type SeenDb struct {
	Db *sql.DB
	E  *Edition // My current edition

	// For performance, we'll retain a single transaction.
	// TODO : Should I commit it and recreate it every now and
	// again to avoid devouring loads of memory?
	Tx *SeenTransaction

	// For re-encrypting the database when done:
	Enc      Encrypt
	TempFile string
	Filename string
}

func (d *SeenDb) Update(filename string, mtimeNow time.Time, getHash func() ([]byte, error), includeFile func() error) (err error) {
	mtimeNowUnix := mtimeNow.Unix()

	// Find the most recent entry for this file:
	var mtimeThenUnix int64
	hashStr := ""
	err = func() error {
		rows, err := d.Tx.GetLatestMtimeHash.Query(filename)
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
		return
	}

	var hashThen, hashNow []byte
	if hashStr != "" {
		// There is an entry for this file.
		// If this file is up to date, we clearly don't
		// need a new edition:
		if mtimeNowUnix <= mtimeThenUnix {
			return
		}

		// Check the hashes; we only need a new edition if
		// the hash has changed
		hashThen, err = base64.StdEncoding.DecodeString(hashStr)
		if err != nil {
			return
		}
	}

	hashNow, err = getHash()
	if err != nil {
		return
	}

	if reflect.DeepEqual(hashNow, hashThen) {
		return
	}

	// Include the file:
	err = includeFile()
	if err != nil {
		return
	}

	// We included the file successfully, update
	// the database:
	_, err = d.Tx.InsertNewEdition.Exec(
		filename,
		d.E.Unix(),
		mtimeNowUnix,
		base64.StdEncoding.EncodeToString(hashNow))
	return
}

func (d *SeenDb) ListEditions() (editions *SortedEditions, err error) {
	editionsUnixMap := make(map[int64]struct{})

	var rows *sql.Rows
	rows, err = d.Tx.ListEditions.Query()
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var editionUnix int64
		err = rows.Scan(&editionUnix)
		if err != nil {
			return
		}

		editionsUnixMap[editionUnix] = struct{}{}
	}

	editions = new(SortedEditions)
	for key := range editionsUnixMap {
		editions.Append(EditionFromUnix(key))
	}

	sort.Sort(editions)
	return
}

func (d *SeenDb) RemoveEdition(edition Edition) (err error) {
	_, err = d.Tx.RemoveEdition.Exec(edition.Unix())
	return err
}

func (d *SeenDb) Close() error {
	// Always make sure we delete the temp file:
	defer os.Remove(d.TempFile)

	// Complete the transaction
	txErr := d.Tx.Close()

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

	if dbErr != nil {
		return dbErr
	}

	return txErr
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
	// TODO Store mtime nanos as well ?
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

	// Open my starting transaction
	tx, err := NewSeenTransaction(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &SeenDb{db, edition, tx, encrypt, tempFile, filename}, nil
}
