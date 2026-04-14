// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package cloudimages

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const SnapshotDateFormat = "20060102"

type Snapshot struct {
	date   time.Time
	number int
}

func (s *Snapshot) version()        {}
func (s *Snapshot) Date() time.Time { return s.date }
func (s *Snapshot) Number() int     { return s.number }
func (s *Snapshot) String() string {
	return fmt.Sprintf("%s-%d", s.date.Format(SnapshotDateFormat), s.number)
}

func ParseSnapshot(s string) (*Snapshot, error) {
	parts := strings.SplitN(s, "-", 3)
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed snapshot %q", s)
	}

	date, err := time.Parse(SnapshotDateFormat, parts[0])
	if err != nil {
		return nil, fmt.Errorf("malformed snapshot %q: %v", s, err)
	}

	number, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("malformed snapshot %q: %v", s, err)
	}

	return &Snapshot{date: date, number: number}, nil
}

func MustParseSnapshot(s string) *Snapshot {
	snapshot, err := ParseSnapshot(s)
	if err != nil {
		panic(err)
	}
	return snapshot
}

type latest struct{}

func (latest) version()       {}
func (latest) String() string { return "latest" }

func LatestVersion() Version {
	return latest{}
}

type Version interface {
	fmt.Stringer
	version()
}

type Daily struct {
	Snapshot *Snapshot
}

func (d *Daily) version() {}
func (d *Daily) String() string {
	return fmt.Sprintf("daily/%s", d.Snapshot)
}

func ParseDaily(s string) (*Daily, error) {
	rest, ok := strings.CutPrefix(s, "daily/")
	if !ok {
		return nil, fmt.Errorf("malformed daily %q", s)
	}

	snapshot, err := ParseSnapshot(rest)
	if err != nil {
		return nil, fmt.Errorf("malformed daily %q: %v", s, err)
	}

	return &Daily{Snapshot: snapshot}, nil
}

func ParseVersion(s string) (Version, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "latest" {
		return LatestVersion(), nil
	}

	switch {
	case s == "lastest":
		return LatestVersion(), nil
	case strings.HasPrefix(s, "daily/"):
		return ParseDaily(s)
	default:
		snapshot, err := ParseSnapshot(s)
		if err != nil {
			return nil, fmt.Errorf("malformed version %q", s)
		}
		return snapshot, nil
	}
}
