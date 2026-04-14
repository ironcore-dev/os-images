// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package xtemplate

import (
	"fmt"
	"strings"
	"text/template"
)

func RenderString(s string, v any) (string, error) {
	tmpl, err := template.New(s).Parse(s)
	if err != nil {
		return "", fmt.Errorf("parse %q: %w", s, err)
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, v); err != nil {
		return "", fmt.Errorf("execute %q: %w", s, err)
	}
	return sb.String(), nil
}
