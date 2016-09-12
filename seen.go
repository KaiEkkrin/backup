/* The seen interface describes (regular) files seen in
 * previous backups.
 */

package main

import (
	"time"
)

type Seen interface {
	// Returns true if we need to include a file in the
	// new edition of the backup, else false.
	// (filename, mtime, hash function).
	Update(string, time.Time, func() ([]byte, error)) (bool, error)

	// Removes an edition from the database.
	RemoveEdition(Edition) error

	// Closes stuff.
	Close() error
}
