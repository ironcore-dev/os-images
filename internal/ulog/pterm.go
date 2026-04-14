// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ulog

import (
	"fmt"
	"strings"

	"github.com/ironcore-dev/os-images/internal/quantity"
	"github.com/pterm/pterm"
)

func formatMessage(name, msg string, keysAndValues ...any) string {
	msg = strings.TrimSpace(fmt.Sprintf("[%s] %s", name, msg))
	if len(keysAndValues) == 0 {
		return msg
	}
	var b strings.Builder
	b.WriteString(msg)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		_, _ = fmt.Fprintf(&b, " %s=%s", strings.TrimSpace(fmt.Sprint(keysAndValues[i])), strings.TrimSpace(fmt.Sprint(keysAndValues[i+1])))
	}
	return b.String()
}

type ptermLogger struct {
	logger pterm.Logger
	args   []pterm.LoggerArgument
}

func PTerm() Logger {
	return &ptermLogger{
		logger: pterm.DefaultLogger,
	}
}

func (p *ptermLogger) newArgsWithKeysAndValues(args []pterm.LoggerArgument, keysAndValues ...any) []pterm.LoggerArgument {
	if len(keysAndValues) == 0 {
		return args
	}

	newArgs := make([]pterm.LoggerArgument, len(args)+len(keysAndValues)/2)
	copy(newArgs, args)
	copy(newArgs[len(args):], p.logger.Args(keysAndValues...))
	return newArgs
}

func (p *ptermLogger) Info(msg string, keysAndValues ...any) {
	p.logger.Info(msg, p.newArgsWithKeysAndValues(p.args, keysAndValues...))
}

func (p *ptermLogger) Error(err error, msg string, keysAndValues ...any) {
	p.logger.Error(msg, p.newArgsWithKeysAndValues(p.args, append([]any{"error", err}, keysAndValues...)...))
}

func (p *ptermLogger) With(keysAndValues ...any) Logger {
	return &ptermLogger{
		logger: p.logger,
		args:   p.newArgsWithKeysAndValues(p.args, keysAndValues...),
	}
}

type ptermProgress struct {
	name    string
	printer *pterm.ProgressbarPrinter
}

func (p *ptermLogger) Progress(max *quantity.Quantity, name string, keysAndValues ...any) Progress {
	printer, err := pterm.DefaultProgressbar.WithTotal(int(max.Value())).Start(formatMessage(name, "", keysAndValues))
	if err != nil {
		panic(err)
	}

	return &ptermProgress{
		name:    name,
		printer: printer,
	}
}

func (pg *ptermProgress) Update(msg string, keysAndValues ...any) {
	pg.printer.UpdateTitle(formatMessage(pg.name, msg, keysAndValues...))
}

func (pg *ptermProgress) Write(p []byte) (n int, err error) {
	pg.printer.Add(len(p))
	return len(p), nil
}

func (pg *ptermProgress) Close() error {
	_, err := pg.printer.Stop()
	return err
}

func (pg *ptermProgress) Set(v int64) {
	pg.printer.Current = int(v)
}

type ptermGroup struct {
	name    string
	printer *pterm.SpinnerPrinter
}

func (p *ptermLogger) Group(name string, keysAndValues ...any) Group {
	printer, err := pterm.DefaultSpinner.Start(formatMessage(name, "", keysAndValues...))
	if err != nil {
		panic(err)
	}

	return &ptermGroup{
		name:    name,
		printer: printer,
	}
}

func (p *ptermGroup) Info(msg string, keysAndValues ...any) {
	p.printer.UpdateText(formatMessage(p.name, msg, keysAndValues...))
}

func (p *ptermGroup) Fail(err error, msg string, keysAndValues ...any) {
	p.printer.Fail(formatMessage(p.name, msg, append([]any{"error", err}, keysAndValues...)...))
}

func (p *ptermGroup) Success(msg string, keysAndValues ...any) {
	p.printer.Success(formatMessage(p.name, msg, keysAndValues...))
}
