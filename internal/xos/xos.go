// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package xos

import (
	"bytes"
	"io"
	"os"
)

func WriteFileReader(name string, rd io.Reader, perm os.FileMode) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, rd)
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}

func WriteTempFile(dir, pattern string, data []byte, perm os.FileMode) (string, error) {
	return WriteTempFileReader(dir, pattern, bytes.NewReader(data), perm)
}

func WriteTempFileReader(dir, pattern string, rd io.Reader, perm os.FileMode) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(f, rd); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}

	if err := os.Chmod(f.Name(), perm); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}

	return f.Name(), nil
}

func CopyFile(srcName, dstName string) error {
	in, err := os.Open(srcName)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	stat, err := in.Stat()
	if err != nil {
		return err
	}

	return WriteFileReader(dstName, in, stat.Mode())
}
