<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Tests\Feature;

use Core\Mcp\Controllers\McpApiController;
use Core\Mcp\Resources\Contracts\AgentResourceProvider;
use Tests\TestCase;

/**
 * The provider is resolved structurally, not nominally.
 *
 * dappcore/agent maintains its own copy of Core\Mcp rather than depending on
 * this package, so it cannot name AgentResourceProvider in an implements
 * clause. Requiring instanceof would leave the plans:// and sessions://
 * endpoints dead permanently rather than until a binding arrived.
 */
class AgentResourceProviderBindingTest extends TestCase
{
    private function resolve(): ?object
    {
        $controller = new class extends McpApiController
        {
            public function provider(): ?object
            {
                return $this->agentResourceProvider();
            }
        };

        return $controller->provider();
    }

    public function test_provider_good_accepts_a_duck_typed_binding(): void
    {
        // No implements clause — exactly what a package that cannot name the
        // interface is able to bind.
        $this->app->bind(AgentResourceProvider::class, fn (): object => new class
        {
            public function read(string $uri): ?array
            {
                return ['uri' => $uri, 'mimeType' => 'text/markdown', 'text' => '# Plan'];
            }
        });

        $provider = $this->resolve();

        $this->assertNotNull($provider);
        $this->assertSame('# Plan', $provider->read('plans://all')['text']);
    }

    public function test_provider_good_accepts_a_nominal_implementation(): void
    {
        $this->app->bind(AgentResourceProvider::class, fn (): AgentResourceProvider => new class implements AgentResourceProvider
        {
            public function read(string $uri): ?array
            {
                return null;
            }
        });

        $this->assertInstanceOf(AgentResourceProvider::class, $this->resolve());
    }

    public function test_provider_bad_rejects_an_object_without_read(): void
    {
        // Half a contract would fatal mid-request; the controller must see
        // nothing rather than something unusable.
        $this->app->bind(AgentResourceProvider::class, fn (): object => new class
        {
            public function somethingElse(): void {}
        });

        $this->assertNull($this->resolve());
    }

    public function test_provider_ugly_is_null_when_nothing_is_bound(): void
    {
        // The package installed without any agent module — a legitimate deploy.
        $this->assertNull($this->resolve());
    }
}
