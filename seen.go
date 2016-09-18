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

	// Lists the editions in the database.
	ListEditions() (*SortedEditions, error)

	// Removes editions later than the given one from
	// the database.
	RemoveEditionsAfter(*Edition) error

	// Closes stuff.
	Close() error
}
