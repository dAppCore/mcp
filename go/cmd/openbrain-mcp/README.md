# openbrain-mcp

`openbrain-mcp` is a thin stdio MCP wrapper for the OpenBrain tools registered in `pkg/mcp/brain`.

Install:

```sh
go install dappco.re/go/mcp/cmd/openbrain-mcp@latest
```

Add it to Claude Code:

```sh
claude mcp add openbrain -- openbrain-mcp --brain-url=http://127.0.0.1:8000/v1/brain --api-key=$OPENBRAIN_API_KEY
```

The wrapper exposes:

- `brain_remember`
- `brain_recall`
- `brain_forget`
- `brain_list`

Flags:

- `--brain-url`: OpenBrain BrainService URL. Defaults to `http://127.0.0.1:8000/v1/brain`.
- `--api-key`: OpenBrain API key. Defaults to `OPENBRAIN_API_KEY`.

The process logs to stderr only. Stdout is reserved for MCP framing.
