// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ulog

import (
	"context"
	"io"

	"github.com/ironcore-dev/os-images/internal/quantity"
	"github.com/ironcore-dev/os-images/internal/xio"
)

type Logger interface {
	Info(msg string, keysAndValues ...any)
	Error(err error, msg string, keysAndValues ...any)
	With(keysAndValues ...any) Logger

	Progress(max *quantity.Quantity, msg string, keysAndValues ...any) Progress
	Group(name string, keysAndValues ...any) Group
}

type Progress interface {
	Update(msg string, keysAndValues ...any)
	io.WriteCloser
	Set(v int64)
}

type Group interface {
	Info(msg string, keysAndValues ...any)
	Fail(err error, msg string, keysAndValues ...any)
	Success(msg string, keysAndValues ...any)
}

func ProgressReadCloser(pg Progress, rc io.ReadCloser) io.ReadCloser {
	rd := io.TeeReader(rc, pg)
	return xio.ReaderAndCloser(
		rd,
		xio.JoinCloser(pg, rc),
	)
}

func GroupWriter(group Group) *xio.LineWriter {
	return xio.NewLineWriter(xio.WriterFunc(func(p []byte) (n int, err error) {
		group.Info(string(p))
		return 0, nil
	}))
}

type contextKey int

const (
	logContextKey contextKey = iota
)

func Default() Logger {
	return DefaultSlog()
}

func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(logContextKey).(Logger); ok {
		return logger
	}
	return Default()
}

func NewContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, logContextKey, logger)
}
