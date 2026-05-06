// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ulog

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ironcore-dev/os-images/internal/quantity"
)

type SLog struct {
	logger *slog.Logger
}

func (s *SLog) Group(name string, keysAndValues ...any) Group {
	return &slogGroup{
		logger: s.logger.With(append([]any{"group", name}, keysAndValues...)...),
	}
}

type slogGroup struct {
	logger *slog.Logger
}

func (s *slogGroup) Info(msg string, keysAndValues ...any) {
	s.logger.Info(msg, keysAndValues...)
}

func (s *slogGroup) Fail(err error, msg string, keysAndValues ...any) {
	s.logger.Error(msg, append([]any{"error", err}, keysAndValues...)...)
}

func (s *slogGroup) Success(msg string, keysAndValues ...any) {
	s.logger.Info(msg, append([]any{"success", true}, keysAndValues...)...)
}

func (s *SLog) Info(msg string, keysAndValues ...any) {
	s.logger.Log(context.Background(), slog.LevelInfo, msg, keysAndValues...)
}

func (s *SLog) Error(err error, msg string, keysAndValues ...any) {
	s.logger.Error(msg, append([]any{"error", err}, keysAndValues...)...)
}

func (s *SLog) With(args ...any) Logger {
	return &SLog{logger: s.logger.With(args...)}
}

func DefaultSlog() *SLog {
	return &SLog{logger: slog.Default()}
}

type SLogProgress struct {
	mu          sync.Mutex
	written     int64
	lastEmit    time.Time
	lastWritten int64

	closed bool
	done   chan struct{}

	granularity   time.Duration
	max           *quantity.Quantity
	logger        *slog.Logger
	msg           string
	keysAndValues []any
}

func (sp *SLogProgress) Update(msg string, keysAndValues ...any) {
	//TODO implement me
	panic("implement me")
}

func (sp *SLogProgress) Write(p []byte) (n int, err error) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.written += int64(len(p))

	sp.unsafeEmitIfNecessary()

	return len(p), nil
}

func (sp *SLogProgress) emit() {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.unsafeEmitIfNecessary()
}

func (sp *SLogProgress) unsafeShouldEmit() bool {
	now := time.Now()

	// Don't emit if we're closed.
	if sp.closed {
		return false
	}

	// Emit if we didn't emit ever.
	if sp.lastEmit.IsZero() {
		return true
	}

	// Don't emit if nothing changed.
	if sp.lastWritten == sp.written {
		return false
	}

	// Max value 0 means we're in a spinner-like mode, so we should emit.
	if sp.max == nil || sp.max.Value() == 0 {
		return true
	}

	return now.Sub(sp.lastEmit) >= sp.granularity
}

func (sp *SLogProgress) unsafeEmitIfNecessary() {
	if sp.closed {
		return
	}

	if sp.unsafeShouldEmit() {
		sp.unsafeEmit()
	}
}

func (sp *SLogProgress) unsafeEmit() {
	var values []any
	if sp.max != nil && sp.max.Value() > 0 {
		values = append(values, "Max", sp.max.String())
	}

	format := quantity.Binary
	if sp.max != nil {
		format = sp.max.Format
	}

	values = append(values, "Written", quantity.New(format, sp.written).String())

	sp.logger.Log(context.Background(), slog.LevelInfo, sp.msg, append(values, sp.keysAndValues...)...)

	sp.lastEmit = time.Now()
	sp.lastWritten = sp.written
}

func (sp *SLogProgress) Stop() {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if !sp.closed {
		sp.closed = true
		close(sp.done)
		sp.unsafeEmitIfNecessary()
	}
}

func (sp *SLogProgress) Close() error {
	sp.Stop()
	return nil
}

func (sp *SLogProgress) Set(v int64) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.written = v
}

func (s *SLog) Progress(max *quantity.Quantity, msg string, keysAndValues ...any) Progress {
	done := make(chan struct{})

	sp := &SLogProgress{
		done:          done,
		max:           max,
		logger:        s.logger,
		msg:           msg,
		keysAndValues: keysAndValues,
		granularity:   time.Second,
	}

	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			select {
			case <-done:
				return
			case <-t.C:
				sp.emit()
			}
		}
	}()

	return sp
}
