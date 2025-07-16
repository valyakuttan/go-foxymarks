package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"filippo.io/age"
)

// HashEqual checks if the hash values of src file equal to dst file
func HashEqual(src, dst string) bool {
	s, err := os.Open(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not open %q: %s\n", src, err)
		os.Exit(0)
	}

	h1 := sha256.New()
	_, err = io.Copy(h1, s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not read from %q: %s\n", src, err)
		os.Exit(0)
	}

	d, err := os.Open(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not open %q: %s\n", dst, err)
		os.Exit(0)
	}
	h2 := sha256.New()
	_, err = io.Copy(h2, d)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not read from %q: %v\n", dst, err)
		os.Exit(0)
	}

	return bytes.Equal(h1.Sum(nil), h2.Sum(nil))
}

func EncryptData(data io.Reader, secret string) ([]byte, error) {
	recipient, err := age.NewScryptRecipient(secret)
	if err != nil {
		return nil, err
	}

	out := &bytes.Buffer{}

	w, err := age.Encrypt(out, recipient)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(w, data); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func DecryptData(data io.Reader, secret string) ([]byte, error) {
	identity, err := age.NewScryptIdentity(secret)
	if err != nil {
		return nil, err
	}

	r, err := age.Decrypt(data, identity)
	if err != nil {
		return nil, err
	}

	out := &bytes.Buffer{}
	if _, err := io.Copy(out, r); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func RandBytes(n int) []byte {
	data := make([]byte, n)
	rand.Read(data)
	return data
}
