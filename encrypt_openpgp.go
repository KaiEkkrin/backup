/* An encrypt implementation with openpgp. */

package main

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"io"
)

const (
	EncryptionType = "PGP SIGNATURE"
)

type EncryptWriter struct {
	ArmorWriter io.WriteCloser
	GpgWriter   io.WriteCloser
}

func (w *EncryptWriter) Write(p []byte) (n int, err error) {
	return w.GpgWriter.Write(p)
}

func (w *EncryptWriter) Close() error {
	gpgErr := w.GpgWriter.Close()
	armorErr := w.ArmorWriter.Close()
	if gpgErr != nil {
		return gpgErr
	} else {
		return armorErr
	}
}

type EncryptOpenpgp struct {
	Passphrase []byte
}

func (e *EncryptOpenpgp) WrapWriter(plainWriter io.WriteSeeker) (io.WriteCloser, error) {
	armor, err := armor.Encode(plainWriter, EncryptionType, nil)
	if err != nil {
		return nil, err
	}

	gpg, err := openpgp.SymmetricallyEncrypt(armor, e.Passphrase, nil, nil)
	if err != nil {
		armor.Close()
		return nil, err
	}

	return &EncryptWriter{armor, gpg}, nil
}

func (e *EncryptOpenpgp) WrapReader(plainReader io.ReadSeeker) (io.Reader, error) {
	armorBlock, err := armor.Decode(plainReader)
	if err != nil {
		return nil, err
	}

	if armorBlock.Type != EncryptionType {
		return nil, errors.New(fmt.Sprintf("Unexpected encryption type %s", armorBlock.Type))
	}

	prompt := func(keys []openpgp.Key, symmetric bool) (pp []byte, err error) {
		pp = e.Passphrase
		if !symmetric {
			err = errors.New("Expected symmetric encryption")
		}

		return pp, err
	}

	md, err := openpgp.ReadMessage(armorBlock.Body, nil, prompt, nil)
	if err != nil {
		return nil, err
	}

	// TODO Verification?  Except, we need to slurp the whole
	// message to check the signature...
	fmt.Printf("Opened message.  Encrypted %q, signed %q\n", md.IsEncrypted, md.IsSigned)
	if !md.IsEncrypted {
		return nil, errors.New("Unencrypted message not supported")
	}

	return md.UnverifiedBody, nil
}

func NewEncryptOpenpgp(passphrase string) *EncryptOpenpgp {
	return &EncryptOpenpgp{[]byte(passphrase)}
}
