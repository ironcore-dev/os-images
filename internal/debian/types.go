// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package debian

import (
	"fmt"
)

const (
	Bullseye = Codename("bullseye")
	Bookworm = Codename("bookworm")
	Trixie   = Codename("trixie")
	Forky    = Codename("forky")

	LatestCodename = Trixie
)

func ParseCodename(codename string) (Codename, error) {
	c := Codename(codename)
	if _, ok := codenameToMajor[c]; ok {
		return c, nil
	}
	return "", fmt.Errorf("unknown codename %q", codename)
}

type Codename string

func (c Codename) Major() string {
	if major, ok := codenameToMajor[c]; ok {
		return major
	}
	return "invalid"
}

// codenameToMajor maps Debian release codenames to major version numbers.
var codenameToMajor = map[Codename]string{
	"bullseye": "11",
	"bookworm": "12",
	"trixie":   "13",
	"forky":    "14",
}
