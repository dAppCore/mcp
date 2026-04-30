<?php

namespace Core\Mcp\Tools;

use Illuminate\Contracts\JsonSchema\JsonSchema;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Tool;

class GetStats extends Tool
{
    protected string $description = 'Get current system statistics for Host Hub';

    public function handle(Request $request): Response
    {
        $stats = [
            'total_sites' => 6,
            'active_users' => 128,
            'page_views_30d' => 12500,
            'server_load' => '23%',
        ];

        return Response::text(json_encode($stats, JSON_PRETTY_PRINT));
    }

    public function schema(JsonSchema $schema): array
    {
        return [];
    }
}
