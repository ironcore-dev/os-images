// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package xhash

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
	"io"
)

type checksumReader struct {
	r        io.Reader
	h        hash.Hash
	expected []byte
}

func (cr *checksumReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	if errors.Is(err, io.EOF) {
		if actual := cr.h.Sum(nil); !bytes.Equal(actual, cr.expected) {
			return n, fmt.Errorf("checksum mismatch: got %x, want %x", actual, cr.expected)
		}
	}
	return n, err
}

func ChecksumReader(r io.Reader, h hash.Hash, expected []byte) io.Reader {
	return &checksumReader{
		r:        io.TeeReader(r, h),
		h:        h,
		expected: expected,
	}
}
