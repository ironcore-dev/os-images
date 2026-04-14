// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// Suite is the suite to build from.
	// This can either be a codename (like trixie, bookworm) or
	// a symbolic name (like unstable, testing).
	Suite      string   `json:"suite"`
	Variant    string   `json:"variant"`
	Packages   []string `json:"packages"`
	Cmdline    string   `json:"cmdline"`
	Components []string `json:"components"`
}

func ReadConfig(data []byte) (*Config, error) {
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func ReadConfigFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ReadConfig(data)
}
