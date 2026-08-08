<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Resources;

use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Resource;

// Not final: tables() / describeTable() are protected seams the suite substitutes
// to describe a schema without a live connection.
// `final` contradicted that and made DatabaseSchemaTest a fatal error.
class DatabaseSchema extends Resource
{
    protected string $description = 'Database schema information for Host Hub';

    /**
     * Build the database-schema resource: enumerates every table and
     * returns a column-level description map encoded as pretty JSON.
     * Falls back to Laravel's Schema facade when the raw SHOW TABLES
     * query is not available (e.g. SQLite).
     *
     * Example:
     *
     *   $resp = (new DatabaseSchema())->handle($request);
     */
    public function handle(Request $request): Response
    {
        $schema = [];

        foreach ($this->tables() as $tableName) {
            $schema[$tableName] = $this->describeTable($tableName);
        }

        return Response::text((string) json_encode($schema, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES));
    }

    protected function tables(): array
    {
        try {
            return collect(DB::select('SHOW TABLES'))
                ->map(static fn (object $row): string => (string) array_values((array) $row)[0])
                ->all();
        } catch (\Throwable) {
            return Schema::getTableListing();
        }
    }

    protected function describeTable(string $tableName): array
    {
        $driver = DB::getDriverName();

        try {
            $statement = $driver === 'sqlite'
                ? 'PRAGMA table_info('.$this->quoteIdentifier($tableName, $driver).')'
                : 'DESCRIBE '.$this->quoteIdentifier($tableName, $driver);

            return array_map(static fn (object $column): array => (array) $column, DB::select($statement));
        } catch (\Throwable) {
            return [];
        }
    }

    protected function quoteIdentifier(string $identifier, string $driver): string
    {
        if ($driver === 'sqlite') {
            return '"'.str_replace('"', '""', $identifier).'"';
        }

        return '`'.str_replace('`', '``', $identifier).'`';
    }
}
