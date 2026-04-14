// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log/slog"
	"os"

	osimages "github.com/ironcore-dev/os-images/cli/os-images"
	"github.com/ironcore-dev/os-images/internal/signals"
)

func main() {
	type Coder interface {
		ExitCode() int
	}

	ctx := signals.SetupSignalHandler()
	if err := osimages.Command().ExecuteContext(ctx); err != nil {
		slog.Error(err.Error())
		if coder, ok := err.(Coder); ok {
			os.Exit(coder.ExitCode())
		}
		os.Exit(1)
	}
}
