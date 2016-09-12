/* Windows specific package for handling uid/gid in
 * backup.
 */

package main

import (
	"archive/tar"
	"os"
)

func AssignUserIds(info os.FileInfo, hdr *tar.Header) {
	// Do nothing.  Windows doesn't support these.
}

func RestoreOwnership(path string, hdr *tar.Header) error {
	// Do nothing.  Windows doesn't support these.
	return nil
}
