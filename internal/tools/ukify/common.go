// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ukify

func buildArgs(linux, initrd, output string, opts Options) []string {
	args := []string{"build",
		"--linux", linux,
		"--initrd", initrd,
		"--output", output,
	}
	if opts.Stub != "" {
		args = append(args, "--stub", opts.Stub)
	}
	if opts.Cmdline != "" {
		args = append(args, "--cmdline", opts.Cmdline)
	}
	return args
}
