// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package xdocker

import (
	"fmt"
	"strings"
)

type BindOption uint8

const (
	ReadOnly BindOption = 1 << iota
	Z
	RPrivate
	Private
	RShared
	Shared
	RSlave
	Slave
)

func Bind(src, dst string, opt BindOption) string {
	bind := fmt.Sprintf("%s:%s", src, dst)
	if opt == 0 {
		return bind
	}

	var sb strings.Builder
	writeOptIfNecessary := func(flag BindOption, s string) {
		if flag&opt != 0 {
			if sb.Len() > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(s)
		}
	}

	writeOptIfNecessary(ReadOnly, "ro")
	writeOptIfNecessary(Z, "z")
	writeOptIfNecessary(RPrivate, "rprivate")
	writeOptIfNecessary(Private, "private")
	writeOptIfNecessary(RShared, "rshared")
	writeOptIfNecessary(Shared, "shared")
	writeOptIfNecessary(RSlave, "rslave")
	writeOptIfNecessary(Slave, "slave")
	return fmt.Sprintf("%s:%s", bind, sb.String())
}
