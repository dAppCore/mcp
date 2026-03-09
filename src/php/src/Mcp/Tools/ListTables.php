<?php

namespace Core\Mcp\Tools;

use Illuminate\Contracts\JsonSchema\JsonSchema;
use Illuminate\Support\Facades\DB;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Tool;

class ListTables extends Tool
{
    protected string $description = 'List all database tables';

    public function handle(Request $request): Response
    {
        $tables = collect(DB::select('SHOW TABLES'))
            ->map(fn ($table) => array_values((array) $table)[0])
            ->toArray();

        return Response::text(json_encode($tables, JSON_PRETTY_PRINT));
    }

    public function schema(JsonSchema $schema): array
    {
        return [];
    }
}
