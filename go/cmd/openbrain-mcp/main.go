// SPDX-License-Identifier: EUPL-1.2

// openbrain-mcp exposes the OpenBrain MCP tools over stdio for Claude Code.
package main

import (
	"context"
	"flag"
	"time"

	. "dappco.re/go"
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
		Error("openbrain-mcp failed", "err", err)
		Exit(1)
	}
}

func run() (
	_ error, // result
) {
	flag.Parse()

	if err := configureBrainEnv(*brainURLFlag, *apiKeyFlag); err != nil {
		return err
	}

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	svc, err := mcp.New(mcp.Options{
		Subsystems: []mcp.Subsystem{
			brain.NewDirect(),
		},
	})
	if err != nil {
		return E("openbrain-mcp.run", "create MCP service", err)
	}
	defer shutdownService(svc)

	if err := svc.ServeStdio(ctx); err != nil && !Is(err, context.Canceled) {
		return E("openbrain-mcp.run", "serve stdio", err)
	}
	return nil
}

func configureBrainEnv(
	brainURL,
	apiKey string,
) (
	_ error, // result
) {
	baseURL := directBrainBaseURL(brainURL)
	if baseURL == "" {
		baseURL = directBrainBaseURL(defaultBrainURL)
	}
	if r := Setenv("CORE_BRAIN_URL", baseURL); !r.OK {
		err, _ := r.Value.(error)
		return E("openbrain-mcp.configure", "set CORE_BRAIN_URL", err)
	}

	key := Trim(apiKey)
	if key == "" {
		key = Trim(Env("OPENBRAIN_API_KEY"))
	}
	if key == "" {
		return nil
	}
	if r := Setenv("CORE_BRAIN_KEY", key); !r.OK {
		err, _ := r.Value.(error)
		return E("openbrain-mcp.configure", "set CORE_BRAIN_KEY", err)
	}
	return nil
}

func directBrainBaseURL(brainURL string) string {
	baseURL := Trim(brainURL)
	baseURL = TrimSuffix(baseURL, "/")
	baseURL = TrimSuffix(baseURL, "/v1/brain")
	return TrimSuffix(baseURL, "/")
}

func shutdownService(svc *mcp.Service) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := svc.Shutdown(ctx); err != nil {
		Error("openbrain-mcp shutdown failed", "err", err)
	}
}
