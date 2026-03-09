<?php

declare(strict_types=1);

namespace Mod\Mcp\Services;

use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Cache;
use Mod\Agentic\Models\AgentPlan;
use Mod\Agentic\Models\AgentSession;

/**
 * Agent Session Service - manages session persistence for agent continuity.
 *
 * Provides session creation, retrieval, and resumption capabilities
 * for multi-agent handoff and long-running tasks.
 */
class AgentSessionService
{
    /**
     * Cache prefix for session state.
     */
    protected const CACHE_PREFIX = 'mcp_session:';

    /**
     * Get the cache TTL from config.
     */
    protected function getCacheTtl(): int
    {
        return (int) config('mcp.session.cache_ttl', 86400);
    }

    /**
     * Start a new session.
     */
    public function start(
        string $agentType,
        ?AgentPlan $plan = null,
        ?int $workspaceId = null,
        array $initialContext = []
    ): AgentSession {
        $session = AgentSession::start($plan, $agentType);

        if ($workspaceId !== null) {
            $session->update(['workspace_id' => $workspaceId]);
        }

        if (! empty($initialContext)) {
            $session->updateContextSummary($initialContext);
        }

        // Cache the active session ID for quick lookup
        $this->cacheActiveSession($session);

        return $session;
    }

    /**
     * Get an active session by ID.
     */
    public function get(string $sessionId): ?AgentSession
    {
        return AgentSession::where('session_id', $sessionId)->first();
    }

    /**
     * Resume an existing session.
     */
    public function resume(string $sessionId): ?AgentSession
    {
        $session = $this->get($sessionId);

        if (! $session) {
            return null;
        }

        // Only resume if paused or was handed off
        if ($session->status === AgentSession::STATUS_PAUSED) {
            $session->resume();
        }

        // Update activity timestamp
        $session->touchActivity();

        // Cache as active
        $this->cacheActiveSession($session);

        return $session;
    }

    /**
     * Get active sessions for a workspace.
     */
    public function getActiveSessions(?int $workspaceId = null): Collection
    {
        $query = AgentSession::active();

        if ($workspaceId !== null) {
            $query->where('workspace_id', $workspaceId);
        }

        return $query->orderBy('last_active_at', 'desc')->get();
    }

    /**
     * Get sessions for a specific plan.
     */
    public function getSessionsForPlan(AgentPlan $plan): Collection
    {
        return AgentSession::forPlan($plan)
            ->orderBy('created_at', 'desc')
            ->get();
    }

    /**
     * Get the most recent session for a plan.
     */
    public function getLatestSessionForPlan(AgentPlan $plan): ?AgentSession
    {
        return AgentSession::forPlan($plan)
            ->orderBy('created_at', 'desc')
            ->first();
    }

    /**
     * End a session.
     */
    public function end(string $sessionId, string $status = AgentSession::STATUS_COMPLETED, ?string $summary = null): ?AgentSession
    {
        $session = $this->get($sessionId);

        if (! $session) {
            return null;
        }

        $session->end($status, $summary);

        // Remove from active cache
        $this->clearCachedSession($session);

        return $session;
    }

    /**
     * Pause a session for later resumption.
     */
    public function pause(string $sessionId): ?AgentSession
    {
        $session = $this->get($sessionId);

        if (! $session) {
            return null;
        }

        $session->pause();

        return $session;
    }

    /**
     * Prepare a session for handoff to another agent.
     */
    public function prepareHandoff(
        string $sessionId,
        string $summary,
        array $nextSteps = [],
        array $blockers = [],
        array $contextForNext = []
    ): ?AgentSession {
        $session = $this->get($sessionId);

        if (! $session) {
            return null;
        }

        $session->prepareHandoff($summary, $nextSteps, $blockers, $contextForNext);

        return $session;
    }

    /**
     * Get handoff context from a session.
     */
    public function getHandoffContext(string $sessionId): ?array
    {
        $session = $this->get($sessionId);

        if (! $session) {
            return null;
        }

        return $session->getHandoffContext();
    }

    /**
     * Create a follow-up session continuing from a previous one.
     */
    public function continueFrom(string $previousSessionId, string $newAgentType): ?AgentSession
    {
        $previousSession = $this->get($previousSessionId);

        if (! $previousSession) {
            return null;
        }

        // Get the handoff context
        $handoffContext = $previousSession->getHandoffContext();

        // Create new session with context from previous
        $newSession = $this->start(
            $newAgentType,
            $previousSession->plan,
            $previousSession->workspace_id,
            [
                'continued_from' => $previousSessionId,
                'previous_agent' => $previousSession->agent_type,
                'handoff_notes' => $handoffContext['handoff_notes'] ?? null,
                'inherited_context' => $handoffContext['context_summary'] ?? null,
            ]
        );

        // Mark previous session as handed off
        $previousSession->end('handed_off', 'Handed off to '.$newAgentType);

        return $newSession;
    }

    /**
     * Store custom state in session cache for fast access.
     */
    public function setState(string $sessionId, string $key, mixed $value, ?int $ttl = null): void
    {
        $cacheKey = self::CACHE_PREFIX.$sessionId.':'.$key;
        Cache::put($cacheKey, $value, $ttl ?? $this->getCacheTtl());
    }

    /**
     * Get custom state from session cache.
     */
    public function getState(string $sessionId, string $key, mixed $default = null): mixed
    {
        $cacheKey = self::CACHE_PREFIX.$sessionId.':'.$key;

        return Cache::get($cacheKey, $default);
    }

    /**
     * Check if a session exists and is valid.
     */
    public function exists(string $sessionId): bool
    {
        return AgentSession::where('session_id', $sessionId)->exists();
    }

    /**
     * Check if a session is active.
     */
    public function isActive(string $sessionId): bool
    {
        $session = $this->get($sessionId);

        return $session !== null && $session->isActive();
    }

    /**
     * Get session statistics.
     */
    public function getSessionStats(?int $workspaceId = null, int $days = 7): array
    {
        $query = AgentSession::where('created_at', '>=', now()->subDays($days));

        if ($workspaceId !== null) {
            $query->where('workspace_id', $workspaceId);
        }

        $sessions = $query->get();

        $byStatus = $sessions->groupBy('status')->map->count();
        $byAgent = $sessions->groupBy('agent_type')->map->count();

        $completedSessions = $sessions->where('status', AgentSession::STATUS_COMPLETED);
        $avgDuration = $completedSessions->avg(fn ($s) => $s->getDuration() ?? 0);

        return [
            'total' => $sessions->count(),
            'active' => $sessions->where('status', AgentSession::STATUS_ACTIVE)->count(),
            'by_status' => $byStatus->toArray(),
            'by_agent_type' => $byAgent->toArray(),
            'avg_duration_minutes' => round($avgDuration, 1),
            'period_days' => $days,
        ];
    }

    /**
     * Clean up stale sessions (active but not touched in X hours).
     */
    public function cleanupStaleSessions(int $hoursInactive = 24): int
    {
        $cutoff = now()->subHours($hoursInactive);

        $staleSessions = AgentSession::active()
            ->where('last_active_at', '<', $cutoff)
            ->get();

        foreach ($staleSessions as $session) {
            $session->fail('Session timed out due to inactivity');
            $this->clearCachedSession($session);
        }

        return $staleSessions->count();
    }

    /**
     * Cache the active session for quick lookup.
     */
    protected function cacheActiveSession(AgentSession $session): void
    {
        $cacheKey = self::CACHE_PREFIX.'active:'.$session->session_id;
        Cache::put($cacheKey, [
            'session_id' => $session->session_id,
            'agent_type' => $session->agent_type,
            'plan_id' => $session->agent_plan_id,
            'workspace_id' => $session->workspace_id,
            'started_at' => $session->started_at?->toIso8601String(),
        ], $this->getCacheTtl());
    }

    /**
     * Clear cached session data.
     */
    protected function clearCachedSession(AgentSession $session): void
    {
        $cacheKey = self::CACHE_PREFIX.'active:'.$session->session_id;
        Cache::forget($cacheKey);
    }
}
