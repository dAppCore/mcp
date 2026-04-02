// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"path/filepath"
	"strings"
	"time"
)

type processRuntime struct {
	Command   string
	Args      []string
	Dir       string
	StartedAt time.Time
}

func (s *Service) recordProcessRuntime(id string, meta processRuntime) {
	if id == "" {
		return
	}

	s.processMu.Lock()
	defer s.processMu.Unlock()

	if s.processMeta == nil {
		s.processMeta = make(map[string]processRuntime)
	}
	s.processMeta[id] = meta
}

func (s *Service) processRuntimeFor(id string) (processRuntime, bool) {
	s.processMu.Lock()
	defer s.processMu.Unlock()

	meta, ok := s.processMeta[id]
	return meta, ok
}

func (s *Service) forgetProcessRuntime(id string) {
	if id == "" {
		return
	}

	s.processMu.Lock()
	defer s.processMu.Unlock()

	delete(s.processMeta, id)
}

func isTestProcess(command string, args []string) bool {
	base := strings.ToLower(filepath.Base(command))
	if base == "" {
		return false
	}

	switch base {
	case "go":
		return len(args) > 0 && strings.EqualFold(args[0], "test")
	case "cargo":
		return len(args) > 0 && strings.EqualFold(args[0], "test")
	case "npm", "pnpm", "yarn", "bun":
		for _, arg := range args {
			if strings.EqualFold(arg, "test") || strings.HasPrefix(strings.ToLower(arg), "test:") {
				return true
			}
		}
		return false
	case "pytest", "phpunit", "jest", "vitest", "rspec", "go-test":
		return true
	}

	return false
}

func (s *Service) emitTestResult(ctx context.Context, processID string, exitCode int, duration time.Duration, signal string, errText string) {
	defer s.forgetProcessRuntime(processID)

	meta, ok := s.processRuntimeFor(processID)
	if !ok || !isTestProcess(meta.Command, meta.Args) {
		return
	}

	if duration <= 0 && !meta.StartedAt.IsZero() {
		duration = time.Since(meta.StartedAt)
	}

	status := "failed"
	if signal != "" {
		status = "aborted"
	} else if exitCode == 0 {
		status = "passed"
	}

	payload := map[string]any{
		"id":      processID,
		"command": meta.Command,
		"args":    meta.Args,
		"status":  status,
		"passed":  status == "passed",
	}
	if !meta.StartedAt.IsZero() {
		payload["startedAt"] = meta.StartedAt
	}
	if duration > 0 {
		payload["duration"] = duration
	}
	if signal == "" || exitCode != 0 {
		payload["exitCode"] = exitCode
	}
	if signal != "" {
		payload["signal"] = signal
	}
	if errText != "" {
		payload["error"] = errText
	}

	s.ChannelSend(ctx, ChannelTestResult, payload)
}
