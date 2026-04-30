package ide

import (
	"time"
)

// moved AX-7 triplet TestConfig_DefaultConfig_Good
func TestConfig_DefaultConfig_Good(t *T) {
	cfg := DefaultConfig()
	AssertEqual(t, "ws://localhost:9876/ws", cfg.LaravelWSURL)
	AssertEqual(t, ".", cfg.WorkspaceRoot)
}

// moved AX-7 triplet TestConfig_DefaultConfig_Bad
func TestConfig_DefaultConfig_Bad(t *T) {
	cfg := DefaultConfig()
	AssertNotEmpty(t, cfg.LaravelWSURL)
	AssertEqual(t, 2*time.Second, cfg.ReconnectInterval)
}

// moved AX-7 triplet TestConfig_DefaultConfig_Ugly
func TestConfig_DefaultConfig_Ugly(t *T) {
	cfg := DefaultConfig()
	AssertEqual(t, 30*time.Second, cfg.MaxReconnectInterval)
	AssertEqual(t, "", cfg.Token)
}

// moved AX-7 triplet TestConfig_Config_WithDefaults_Good
func TestConfig_Config_WithDefaults_Good(t *T) {
	cfg := (Config{ReconnectInterval: time.Millisecond}).WithDefaults()
	AssertEqual(t, time.Millisecond, cfg.ReconnectInterval)
	AssertNotEmpty(t, cfg.LaravelWSURL)
}

// moved AX-7 triplet TestConfig_Config_WithDefaults_Bad
func TestConfig_Config_WithDefaults_Bad(t *T) {
	cfg := (Config{}).WithDefaults()
	AssertEqual(t, "ws://localhost:9876/ws", cfg.LaravelWSURL)
	AssertEqual(t, ".", cfg.WorkspaceRoot)
}

// moved AX-7 triplet TestConfig_Config_WithDefaults_Ugly
func TestConfig_Config_WithDefaults_Ugly(t *T) {
	cfg := (Config{LaravelWSURL: "ws://custom", WorkspaceRoot: "/tmp/work"}).WithDefaults()
	AssertEqual(t, "ws://custom", cfg.LaravelWSURL)
	AssertEqual(t, "/tmp/work", cfg.WorkspaceRoot)
}
