module dappco.re/go/mcp

go 1.26.2

require (
	dappco.re/go/ai v0.8.0-alpha.1.0.20260425225549-d43f4dbd25b8
	dappco.re/go/api v0.8.0-alpha.1.0.20260427143506-f95002059d53
	dappco.re/go/cli v0.9.0
	dappco.re/go/process v0.8.0-alpha.1.0.20260427150750-5bd42af95f46
	dappco.re/go/rag v0.8.0-alpha.1.0.20260427161922-2a59096f2aca
	dappco.re/go/webview v0.8.0-alpha.1.0.20260425135446-1c47ae2c183c
	dappco.re/go/ws v0.8.0-alpha.1.0.20260427142937-36f01754d2e9
	github.com/gin-gonic/gin v1.12.0
	github.com/gorilla/websocket v1.5.3
	github.com/modelcontextprotocol/go-sdk v1.5.0
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/rogpeppe/go-internal v1.14.1 // indirect

require (
	dappco.re/go v0.9.0
	github.com/bytedance/gopkg v0.1.4 // indirect
	github.com/bytedance/sonic v1.15.0 // indirect
	github.com/bytedance/sonic/loader v0.5.0 // indirect
	github.com/charmbracelet/x/ansi v0.11.6 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.1 // indirect
	github.com/goccy/go-json v0.10.6
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.21 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.59.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.1 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.mongodb.org/mongo-driver/v2 v2.5.0 // indirect
	golang.org/x/arch v0.25.0 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/term v0.42.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace dappco.re/go/ai => ./internal/shims/go-ai

replace dappco.re/go/api => ./internal/shims/go-api

replace dappco.re/go/cli => ../cli

replace dappco.re/go/i18n => github.com/dappcore/go-i18n v0.8.0-alpha.1

replace dappco.re/go/process => ./internal/shims/go-process

replace dappco.re/go/rag => ./internal/shims/go-rag

replace dappco.re/go/webview => ./internal/shims/go-webview

replace dappco.re/go/ws => ./internal/shims/go-ws
