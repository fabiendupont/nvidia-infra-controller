// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package switchconfig

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

//go:embed templates/*.tmpl
var defaultTemplates embed.FS

// RenderContext is the data model passed to all CLI config templates.
// It is populated from NicoSwitchService.GetSwitch and GetSwitchRoutingConfig.
type RenderContext struct {
	Switch        *providerv1.NicoSwitch
	RoutingConfig *providerv1.SwitchRoutingConfig
	SiteID        string
	// Extra holds arbitrary provider-side metadata (e.g. rack neighbours,
	// cable map) that templates may use but NICo does not model centrally.
	Extra map[string]interface{}
}

// RenderCLI renders a CLI-format switch configuration using Go text/template
// with Sprig functions. templateSrc is the template string; ctx is the data
// context populated from NicoSwitchService responses.
func RenderCLI(templateSrc string, ctx RenderContext) ([]byte, error) {
	tmpl, err := template.New("switch-config").
		Funcs(sprig.TxtFuncMap()).
		Parse(templateSrc)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return buf.Bytes(), nil
}

// DefaultTemplate returns the embedded default template for the given platform
// and format. Platform is e.g. "cumulus", "sonic", "nvos". Format is "cli".
// Returns an error if no default exists for the combination.
func DefaultTemplate(platform, format string) (string, error) {
	name := fmt.Sprintf("templates/%s-%s.tmpl", platform, format)
	data, err := defaultTemplates.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("no default template for platform=%q format=%q: %w", platform, format, err)
	}
	return string(data), nil
}
