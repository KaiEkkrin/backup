/* A sortable collection of archive names. */

package main

import (
    "path/filepath"
    "strings"
    )

type ArchiveName struct {
    Name string
    E *Edition
}

type ArchiveNames struct {
    Prefix string
    Suffix string
    Names []ArchiveName
}

func (a *ArchiveNames) Append(dir, name string) error {
    trimmed := strings.Replace(name, a.Prefix, "", 1)
    trimmed = strings.Replace(trimmed, a.Suffix, "", 1)

    edition, err := EditionFromString(trimmed)
    if err != nil {
        return err
    }

    a.Names = append(a.Names, ArchiveName{
        filepath.Join(dir, name),
        edition})
    return nil
}

func (a *ArchiveNames) GetName(i int) string {
    return a.Names[i].Name
}

func (a *ArchiveNames) Len() int {
    return len(a.Names)
}


func (a *ArchiveNames) Less(i, j int) bool {
    return a.Names[i].E.When.Before(a.Names[j].E.When)
}

func (a *ArchiveNames) Swap(i, j int) {
    tmp := a.Names[i]
    a.Names[i] = a.Names[j]
    a.Names[j] = tmp
}

