<?php

namespace Core\Mcp\Tools;

use Illuminate\Contracts\JsonSchema\JsonSchema;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Tool;

class ListRoutes extends Tool
{
    protected string $description = 'List all web routes in the application';

    public function handle(Request $request): Response
    {
        $routes = collect(app('router')->getRoutes())
            ->map(fn ($route) => [
                'uri' => $route->uri(),
                'methods' => $route->methods(),
                'name' => $route->getName(),
            ])
            ->values()
            ->toArray();

        return Response::text(json_encode($routes, JSON_PRETTY_PRINT));
    }

    public function schema(JsonSchema $schema): array
    {
        return [];
    }
}
