/* Encrypts and applies error resistance with kblob. */

package main

import (
	"github.com/kaiekkrin/komblobulate"
	"io"
)

// Implements the KCodecParams interface.
// TODO : Configurability?  For now, I've just
// picked some sensible-seeming defaults.
type EncryptKblobParams struct {
	Password string
}

func (p *EncryptKblobParams) GetRsParams() (int, int, int) {
	return 508, 8, 1
}

func (p *EncryptKblobParams) GetAeadChunkSize() int {
	return 256 * 1024
}

func (p *EncryptKblobParams) GetAeadPassword() string {
	return p.Password
}

type EncryptKblob struct {
	Params *EncryptKblobParams
}

func (e *EncryptKblob) WrapWriter(writer io.WriteSeeker) (io.WriteCloser, error) {
	return komblobulate.NewWriter(writer, komblobulate.ResistType_Rs, komblobulate.CipherType_Aead, e.Params)
}

func (e *EncryptKblob) WrapReader(reader io.ReadSeeker) (io.Reader, error) {
	return komblobulate.NewReader(reader, e.Params)
}

func NewEncryptKblob(passphrase string) *EncryptKblob {
	return &EncryptKblob{&EncryptKblobParams{passphrase}}
}
