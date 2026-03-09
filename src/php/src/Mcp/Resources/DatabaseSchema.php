<?php

namespace Core\Mcp\Resources;

use Illuminate\Support\Facades\DB;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Resource;

class DatabaseSchema extends Resource
{
    protected string $description = 'Database schema information for Host Hub';

    public function handle(Request $request): Response
    {
        $schema = collect(DB::select('SHOW TABLES'))
            ->mapWithKeys(function ($table) {
                $tableName = array_values((array) $table)[0];
                $columns = DB::select("DESCRIBE {$tableName}");

                return [$tableName => $columns];
            })
            ->toArray();

        return Response::text(json_encode($schema, JSON_PRETTY_PRINT));
    }
}
