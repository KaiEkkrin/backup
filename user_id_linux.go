/* Linux specific package for handling uid/gid in
 * backup.
 */

package main

import (
	"archive/tar"
	"os"
	"syscall"
)

func AssignUserIds(info os.FileInfo, hdr *tar.Header) {
	if sys, found := info.Sys().(*syscall.Stat_t); found {
		hdr.Uid = int(sys.Uid)
		hdr.Gid = int(sys.Gid)

		// TODO Extended attributes
	}
}

func RestoreOwnership(path string, hdr *tar.Header) error {
	return os.Chown(path, hdr.Uid, hdr.Gid)
}
