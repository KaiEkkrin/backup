/* An encryption interface for wrapping readers and writers. */

package main

import (
    "io"
    )

type Encrypt interface {
    WrapWriter(io.Writer) (io.WriteCloser, error)
    WrapReader(io.Reader) (io.Reader, error)
}

