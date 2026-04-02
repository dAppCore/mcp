package ide

import (
	"context"
	"fmt"
	"sync"
	"time"

	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	coreerr "forge.lthn.ai/core/go-log"
	"forge.lthn.ai/core/go-ws"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// errBridgeNotAvailable is returned when a tool requires the Laravel bridge
// but it has not been initialised (headless mode).
var errBridgeNotAvailable = coreerr.E("ide", "bridge not available", nil)

// Subsystem implements mcp.Subsystem and mcp.SubsystemWithShutdown for the IDE.
type Subsystem struct {
	cfg      Config
	bridge   *Bridge
	hub      *ws.Hub
	notifier coremcp.Notifier

	stateMu      sync.Mutex
	sessionOrder []string
	sessions     map[string]Session
	chats        map[string][]ChatMessage
	buildOrder   []string
	builds       map[string]BuildInfo
	buildLogMap  map[string][]string
	activity     []ActivityEvent
}

// New creates an IDE subsystem from a Config DTO.
//
// The ws.Hub is used for real-time forwarding; pass nil if headless
// (tools still work but real-time streaming is disabled).
func New(hub *ws.Hub, cfg Config) *Subsystem {
	cfg = cfg.WithDefaults()
	s := &Subsystem{
		cfg:         cfg,
		bridge:      nil,
		hub:         hub,
		sessions:    make(map[string]Session),
		chats:       make(map[string][]ChatMessage),
		builds:      make(map[string]BuildInfo),
		buildLogMap: make(map[string][]string),
	}
	if hub != nil {
		s.bridge = NewBridge(hub, cfg)
		s.bridge.SetObserver(func(msg BridgeMessage) {
			s.handleBridgeMessage(msg)
		})
	}
	return s
}

// Name implements mcp.Subsystem.
func (s *Subsystem) Name() string { return "ide" }

// RegisterTools implements mcp.Subsystem.
func (s *Subsystem) RegisterTools(server *mcp.Server) {
	s.registerChatTools(server)
	s.registerBuildTools(server)
	s.registerDashboardTools(server)
}

// Shutdown implements mcp.SubsystemWithShutdown.
func (s *Subsystem) Shutdown(_ context.Context) error {
	if s.bridge != nil {
		s.bridge.Shutdown()
	}
	return nil
}

// SetNotifier wires the shared MCP notifier into the IDE subsystem.
func (s *Subsystem) SetNotifier(n coremcp.Notifier) {
	s.notifier = n
}

// Bridge returns the Laravel WebSocket bridge (may be nil in headless mode).
func (s *Subsystem) Bridge() *Bridge { return s.bridge }

// StartBridge begins the background connection to the Laravel backend.
func (s *Subsystem) StartBridge(ctx context.Context) {
	if s.bridge != nil {
		s.bridge.Start(ctx)
	}
}

func (s *Subsystem) addSession(session Session) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if s.sessions == nil {
		s.sessions = make(map[string]Session)
	}
	if s.chats == nil {
		s.chats = make(map[string][]ChatMessage)
	}
	if _, exists := s.sessions[session.ID]; !exists {
		s.sessionOrder = append(s.sessionOrder, session.ID)
	}
	s.sessions[session.ID] = session
}

func (s *Subsystem) addBuild(build BuildInfo) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if s.builds == nil {
		s.builds = make(map[string]BuildInfo)
	}
	if s.buildLogMap == nil {
		s.buildLogMap = make(map[string][]string)
	}
	if _, exists := s.builds[build.ID]; !exists {
		s.buildOrder = append(s.buildOrder, build.ID)
	}
	if build.StartedAt.IsZero() {
		build.StartedAt = time.Now()
	}
	s.builds[build.ID] = build
}

func (s *Subsystem) listBuilds(repo string, limit int) []BuildInfo {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if len(s.buildOrder) == 0 {
		return []BuildInfo{}
	}

	if limit <= 0 {
		limit = len(s.buildOrder)
	}

	builds := make([]BuildInfo, 0, limit)
	for i := len(s.buildOrder) - 1; i >= 0; i-- {
		id := s.buildOrder[i]
		build, ok := s.builds[id]
		if !ok {
			continue
		}
		if repo != "" && build.Repo != repo {
			continue
		}
		builds = append(builds, build)
		if len(builds) >= limit {
			break
		}
	}
	return builds
}

func (s *Subsystem) appendBuildLog(buildID, line string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if s.buildLogMap == nil {
		s.buildLogMap = make(map[string][]string)
	}
	s.buildLogMap[buildID] = append(s.buildLogMap[buildID], line)
}

func (s *Subsystem) setBuildLogs(buildID string, lines []string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if s.buildLogMap == nil {
		s.buildLogMap = make(map[string][]string)
	}
	if len(lines) == 0 {
		s.buildLogMap[buildID] = []string{}
		return
	}
	out := make([]string, len(lines))
	copy(out, lines)
	s.buildLogMap[buildID] = out
}

func (s *Subsystem) buildLogTail(buildID string, tail int) []string {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	lines := s.buildLogMap[buildID]
	if len(lines) == 0 {
		return []string{}
	}
	if tail <= 0 || tail > len(lines) {
		tail = len(lines)
	}
	start := len(lines) - tail
	out := make([]string, tail)
	copy(out, lines[start:])
	return out
}

func (s *Subsystem) buildSnapshot(buildID string) (BuildInfo, bool) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	build, ok := s.builds[buildID]
	return build, ok
}

func (s *Subsystem) buildRepoCount() int {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	repos := make(map[string]struct{})
	for _, build := range s.builds {
		if build.Repo != "" {
			repos[build.Repo] = struct{}{}
		}
	}
	return len(repos)
}

func (s *Subsystem) listSessions() []Session {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if len(s.sessionOrder) == 0 {
		return []Session{}
	}

	result := make([]Session, 0, len(s.sessionOrder))
	for _, id := range s.sessionOrder {
		if session, ok := s.sessions[id]; ok {
			result = append(result, session)
		}
	}
	return result
}

func (s *Subsystem) appendChatMessage(sessionID, role, content string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if s.chats == nil {
		s.chats = make(map[string][]ChatMessage)
	}
	s.chats[sessionID] = append(s.chats[sessionID], ChatMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}

func (s *Subsystem) chatMessages(sessionID string) []ChatMessage {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	history := s.chats[sessionID]
	if len(history) == 0 {
		return []ChatMessage{}
	}
	out := make([]ChatMessage, len(history))
	copy(out, history)
	return out
}

func (s *Subsystem) recordActivity(typ, msg string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	s.activity = append(s.activity, ActivityEvent{
		Type:      typ,
		Message:   msg,
		Timestamp: time.Now(),
	})
}

func (s *Subsystem) activityFeed(limit int) []ActivityEvent {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if limit <= 0 || limit > len(s.activity) {
		limit = len(s.activity)
	}
	if limit == 0 {
		return []ActivityEvent{}
	}

	start := len(s.activity) - limit
	out := make([]ActivityEvent, limit)
	copy(out, s.activity[start:])
	return out
}

func (s *Subsystem) handleBridgeMessage(msg BridgeMessage) {
	switch msg.Type {
	case "build_status":
		if build, ok := buildInfoFromData(msg.Data); ok {
			s.addBuild(build)
			s.emitBuildLifecycle(build)
			if lines := buildLinesFromData(msg.Data); len(lines) > 0 {
				s.setBuildLogs(build.ID, lines)
			}
		}
	case "build_list":
		for _, build := range buildInfosFromData(msg.Data) {
			s.addBuild(build)
		}
	case "build_logs":
		buildID, lines := buildLogsFromData(msg.Data)
		if buildID != "" {
			s.setBuildLogs(buildID, lines)
		}
	case "session_list":
		for _, session := range sessionsFromData(msg.Data) {
			s.addSession(session)
		}
	case "session_create":
		if session, ok := sessionFromData(msg.Data); ok {
			s.addSession(session)
		}
	case "chat_history":
		if sessionID, messages := chatHistoryFromData(msg.Data); sessionID != "" {
			for _, message := range messages {
				s.appendChatMessage(sessionID, message.Role, message.Content)
			}
		}
	}
}

func (s *Subsystem) emitBuildLifecycle(build BuildInfo) {
	if s.notifier == nil {
		return
	}

	channel := ""
	switch build.Status {
	case "running", "in_progress", "started":
		channel = coremcp.ChannelBuildStart
	case "success", "succeeded", "completed", "passed":
		channel = coremcp.ChannelBuildComplete
	case "failed", "error":
		channel = coremcp.ChannelBuildFailed
	default:
		return
	}

	payload := map[string]any{
		"id":        build.ID,
		"repo":      build.Repo,
		"branch":    build.Branch,
		"status":    build.Status,
		"startedAt": build.StartedAt,
	}
	if build.Duration != "" {
		payload["duration"] = build.Duration
	}
	s.notifier.ChannelSend(context.Background(), channel, payload)
}

func buildInfoFromData(data any) (BuildInfo, bool) {
	m, ok := data.(map[string]any)
	if !ok {
		return BuildInfo{}, false
	}

	id, _ := m["buildId"].(string)
	if id == "" {
		id, _ = m["id"].(string)
	}
	if id == "" {
		return BuildInfo{}, false
	}

	build := BuildInfo{
		ID:     id,
		Repo:   stringFromAny(m["repo"]),
		Branch: stringFromAny(m["branch"]),
		Status: stringFromAny(m["status"]),
	}
	if build.Status == "" {
		build.Status = "unknown"
	}
	if startedAt, ok := m["startedAt"].(time.Time); ok {
		build.StartedAt = startedAt
	}
	if duration := stringFromAny(m["duration"]); duration != "" {
		build.Duration = duration
	}
	return build, true
}

func buildInfosFromData(data any) []BuildInfo {
	m, ok := data.(map[string]any)
	if !ok {
		return []BuildInfo{}
	}

	raw, ok := m["builds"].([]any)
	if !ok {
		return []BuildInfo{}
	}

	builds := make([]BuildInfo, 0, len(raw))
	for _, item := range raw {
		build, ok := buildInfoFromData(item)
		if ok {
			builds = append(builds, build)
		}
	}
	return builds
}

func buildLinesFromData(data any) []string {
	_, lines := buildLogsFromData(data)
	return lines
}

func buildLogsFromData(data any) (string, []string) {
	m, ok := data.(map[string]any)
	if !ok {
		return "", []string{}
	}

	buildID, _ := m["buildId"].(string)
	if buildID == "" {
		buildID, _ = m["id"].(string)
	}

	switch raw := m["lines"].(type) {
	case []any:
		lines := make([]string, 0, len(raw))
		for _, item := range raw {
			lines = append(lines, stringFromAny(item))
		}
		return buildID, lines
	case []string:
		lines := make([]string, len(raw))
		copy(lines, raw)
		return buildID, lines
	}

	if output := stringFromAny(m["output"]); output != "" {
		return buildID, []string{output}
	}

	return buildID, []string{}
}

func sessionsFromData(data any) []Session {
	m, ok := data.(map[string]any)
	if !ok {
		return []Session{}
	}

	raw, ok := m["sessions"].([]any)
	if !ok {
		return []Session{}
	}

	sessions := make([]Session, 0, len(raw))
	for _, item := range raw {
		session, ok := sessionFromData(item)
		if ok {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

func sessionFromData(data any) (Session, bool) {
	m, ok := data.(map[string]any)
	if !ok {
		return Session{}, false
	}

	id, _ := m["id"].(string)
	if id == "" {
		return Session{}, false
	}

	session := Session{
		ID:        id,
		Name:      stringFromAny(m["name"]),
		Status:    stringFromAny(m["status"]),
		CreatedAt: time.Now(),
	}
	if createdAt, ok := m["createdAt"].(time.Time); ok {
		session.CreatedAt = createdAt
	}
	if session.Status == "" {
		session.Status = "unknown"
	}
	return session, true
}

func chatHistoryFromData(data any) (string, []ChatMessage) {
	m, ok := data.(map[string]any)
	if !ok {
		return "", []ChatMessage{}
	}

	sessionID, _ := m["sessionId"].(string)
	if sessionID == "" {
		sessionID, _ = m["session_id"].(string)
	}

	raw, ok := m["messages"].([]any)
	if !ok {
		return sessionID, []ChatMessage{}
	}

	messages := make([]ChatMessage, 0, len(raw))
	for _, item := range raw {
		if msg, ok := chatMessageFromData(item); ok {
			messages = append(messages, msg)
		}
	}
	return sessionID, messages
}

func chatMessageFromData(data any) (ChatMessage, bool) {
	m, ok := data.(map[string]any)
	if !ok {
		return ChatMessage{}, false
	}

	role := stringFromAny(m["role"])
	content := stringFromAny(m["content"])
	if role == "" && content == "" {
		return ChatMessage{}, false
	}

	msg := ChatMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
	if ts, ok := m["timestamp"].(time.Time); ok {
		msg.Timestamp = ts
	}
	return msg, true
}

func stringFromAny(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return ""
	}
}

func newSessionID() string {
	return core.ID()
}
