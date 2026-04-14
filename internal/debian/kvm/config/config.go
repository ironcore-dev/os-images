// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Codename string `json:"codename"`
	Version  string `json:"version"`
	Cmdline  string `json:"cmdline"`
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
