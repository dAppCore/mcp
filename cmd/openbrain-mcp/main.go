// SPDX-License-Identifier: EUPL-1.2

// openbrain-mcp exposes the OpenBrain MCP tools over stdio for Claude Code.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	core "dappco.re/go"
	coreerr "dappco.re/go/log"
	"dappco.re/go/mcp/pkg/mcp"
	"dappco.re/go/mcp/pkg/mcp/brain"
)

const defaultBrainURL = "http://127.0.0.1:8000/v1/brain"

var (
	brainURLFlag = flag.String("brain-url", defaultBrainURL, "OpenBrain BrainService URL")
	apiKeyFlag   = flag.String("api-key", "", "OpenBrain API key (defaults to OPENBRAIN_API_KEY)")
)

func main() {
	if err := run(); err != nil {
		coreerr.Error("openbrain-mcp failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	if err := configureBrainEnv(*brainURLFlag, *apiKeyFlag); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svc, err := mcp.New(mcp.Options{
		Subsystems: []mcp.Subsystem{
			brain.NewDirect(),
		},
	})
	if err != nil {
		return core.E("openbrain-mcp.run", "create MCP service", err)
	}
	defer shutdownService(svc)

	if err := svc.ServeStdio(ctx); err != nil && !core.Is(err, context.Canceled) {
		return core.E("openbrain-mcp.run", "serve stdio", err)
	}
	return nil
}

func configureBrainEnv(brainURL, apiKey string) error {
	baseURL := directBrainBaseURL(brainURL)
	if baseURL == "" {
		baseURL = directBrainBaseURL(defaultBrainURL)
	}
	if err := os.Setenv("CORE_BRAIN_URL", baseURL); err != nil {
		return core.E("openbrain-mcp.configure", "set CORE_BRAIN_URL", err)
	}

	key := core.Trim(apiKey)
	if key == "" {
		key = core.Trim(core.Env("OPENBRAIN_API_KEY"))
	}
	if key == "" {
		return nil
	}
	if err := os.Setenv("CORE_BRAIN_KEY", key); err != nil {
		return core.E("openbrain-mcp.configure", "set CORE_BRAIN_KEY", err)
	}
	return nil
}

func directBrainBaseURL(brainURL string) string {
	baseURL := core.Trim(brainURL)
	baseURL = core.TrimSuffix(baseURL, "/")
	baseURL = core.TrimSuffix(baseURL, "/v1/brain")
	return core.TrimSuffix(baseURL, "/")
}

func shutdownService(svc *mcp.Service) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := svc.Shutdown(ctx); err != nil {
		coreerr.Error("openbrain-mcp shutdown failed", "err", err)
	}
}
