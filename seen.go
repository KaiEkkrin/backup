/* The seen interface describes (regular) files seen in
 * previous backups.
 */

package main

import (
	"time"
)

type Seen interface {
	// Includes the file in the new edition of the backup
	// if required, using the supplied function.
	// (filename, mtime, hash function, include function).
	Update(string, time.Time, func() ([]byte, error), func() error) error

	// Removes an edition from the database.
	RemoveEdition(Edition) error

	// Closes stuff.
	Close() error
}
