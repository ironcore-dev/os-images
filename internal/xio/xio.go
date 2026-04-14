// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package xio

import (
	"bytes"
	"errors"
	"io"
)

func ReaderAndCloser(r io.Reader, c io.Closer) io.ReadCloser {
	return struct {
		io.Reader
		io.Closer
	}{r, c}
}

type CloserFunc func() error

func (f CloserFunc) Close() error {
	return f()
}

type joinCloser []io.Closer

func (c joinCloser) Close() error {
	errs := make([]error, 0, len(c))
	for _, c := range c {
		if err := c.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func JoinCloser(closers ...io.Closer) io.Closer {
	return joinCloser(closers)
}

type prefixedWriter struct {
	prefix []byte
	writer io.Writer
}

func (pw *prefixedWriter) Write(p []byte) (n int, err error) {
	buf := make([]byte, len(pw.prefix)+len(p))
	copy(buf, pw.prefix)
	copy(buf[len(pw.prefix):], p)
	n, err = pw.writer.Write(buf)
	return max(0, n-len(pw.prefix)), err
}

// PrefixWriter creates a writer that forwards each write with the prefix prefixed to the given writer.
func PrefixWriter[Prefix ~string | ~[]byte](prefix Prefix, w io.Writer) io.Writer {
	return &prefixedWriter{
		prefix: []byte(prefix),
		writer: w,
	}
}

type LineWriter struct {
	buf bytes.Buffer
	wr  io.Writer
}

func (w *LineWriter) writeLine(line []byte) error {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return nil
	}
	_, err := w.wr.Write(trimmed)
	return err
}

func (w *LineWriter) Write(p []byte) (int, error) {
	n := len(p)
	for len(p) > 0 {
		i := bytes.IndexByte(p, '\n')
		if i < 0 {
			// No newline in remaining input — buffer it.
			w.buf.Write(p)
			break
		}

		line := p[:i+1]
		p = p[i+1:]

		if w.buf.Len() > 0 {
			// Complete the buffered partial line.
			w.buf.Write(line)
			if err := w.writeLine(w.buf.Bytes()); err != nil {
				return 0, err
			}
			w.buf.Reset()
		} else {
			// Full line in input — write directly, no buffering.
			if err := w.writeLine(line); err != nil {
				return 0, err
			}
		}
	}
	return n, nil
}

func (w *LineWriter) Flush() error {
	rest := bytes.TrimSpace(w.buf.Bytes())
	w.buf.Reset()
	if len(rest) == 0 {
		return nil
	}
	_, err := w.wr.Write(rest)
	return err
}

func NewLineWriter(w io.Writer) *LineWriter {
	return &LineWriter{
		wr: w,
	}
}

type WriterFunc func(p []byte) (n int, err error)

func (f WriterFunc) Write(p []byte) (n int, err error) { return f(p) }
