/* An encryption interface for wrapping readers and writers. */

package main

import (
	"io"
)

type Encrypt interface {
	WrapWriter(io.WriteSeeker) (io.WriteCloser, error)
	WrapReader(io.ReadSeeker) (io.Reader, error)
}
