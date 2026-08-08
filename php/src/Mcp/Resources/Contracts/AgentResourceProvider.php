<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Resources\Contracts;

/**
 * Supplies the agent-domain resource surface — plans, phase checklists, plan
 * state, session context — to this package's MCP transports.
 *
 * The contract is owned here and implemented downstream, which is the whole
 * point: McpApiController used to render those resources itself, importing
 * Core\Mod\Agentic\Models\AgentPlan and AgentSession directly. That made the
 * shared package depend on its own consumer. A library must not import the
 * thing that imports it, so the rendering moves to whoever owns the data and
 * this package asks for it by contract.
 *
 * Implementations are bound into the container under this interface name.
 * Nothing is bound when the agent module is absent, and the controller answers
 * "unavailable" rather than pretending — the resources are optional to this
 * package, not to the protocol.
 *
 * read() only, deliberately. An earlier draft also required entries() for
 * listing, which nothing here ever called: this package's
 * GET servers/{id}/resources lists a server's own configured resources, a
 * different concept. A contract method with no consumer is over-specification
 * that an implementer has to satisfy for nothing, so it is not asked for.
 * Providers are free to offer listing for their own transports —
 * dappcore/agent's registry does, for its stdio server.
 *
 * Satisfied structurally as well as nominally: a provider that cannot name this
 * interface (because it maintains its own copy of Core\Mcp rather than
 * depending on this package) binds under the interface name and implements
 * read(), and McpApiController accepts it.
 */
interface AgentResourceProvider
{
    /**
     * Read one resource, or null when nothing serves that URI.
     *
     * Null covers both "no resource matches this shape" and "the shape matched
     * but names something that does not exist", so a caller can answer a proper
     * error either way instead of returning prose like "Plan not found" as the
     * resource body — which a client cannot tell apart from a real document
     * whose text happens to say so.
     *
     * @return array{uri: string, mimeType: string, text: string}|null
     *
     * @example
     * $provider->read('plans://all');
     */
    public function read(string $uri): ?array;
}
