/* A transaction in the seen database. */

package main

import (
    "database/sql"
    )

type SeenTransaction struct {
    Tx *sql.Tx
    GetLatestMtimeHash *sql.Stmt
    InsertNewEdition *sql.Stmt
    RemoveEdition *sql.Stmt
}

func (tx *SeenTransaction) Close() error {
    return tx.Tx.Commit()
}

func NewSeenTransaction(db *sql.DB) (*SeenTransaction, error) {
    tx, err := db.Begin()
    if err != nil {
        return nil, err
    }

    getLatestMtimeHash, err := tx.Prepare(
        `select mtime, hash from files
        where filename=?
        order by mtime desc`)
    if err != nil {
        return nil, err
    }

    insertNewEdition, err := tx.Prepare(
        `insert into files values (?, ?, ?, ?)`)
    if err != nil {
        return nil, err
    }

    removeEdition, err := tx.Prepare(
        `delete from files where edition=?`)
    if err != nil {
        return nil, err
    }

    return &SeenTransaction{tx, getLatestMtimeHash, insertNewEdition, removeEdition}, nil
}

