<?php

declare(strict_types=1);

namespace Core\Mcp\Tools;

use Illuminate\Contracts\JsonSchema\JsonSchema;
use Illuminate\Support\Facades\Config;
use Illuminate\Support\Facades\DB;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Tool;

class DescribeTable extends Tool
{
    protected string $description = 'Describe a database table, including columns and indexes';

    public function handle(Request $request): Response
    {
        $table = trim((string) $request->input('table', ''));

        if ($table === '') {
            return $this->errorResponse('Table name is required');
        }

        if (! $this->isValidTableName($table)) {
            return $this->errorResponse('Invalid table name. Use only letters, numbers, and underscores.');
        }

        if ($this->isBlockedTable($table)) {
            return $this->errorResponse(sprintf("Access to table '%s' is not permitted", $table));
        }

        try {
            $columns = DB::select("SHOW FULL COLUMNS FROM `{$table}`");
            $indexes = DB::select("SHOW INDEX FROM `{$table}`");
        } catch (\Throwable $e) {
            report($e);

            return $this->errorResponse(sprintf('Unable to describe table "%s"', $table));
        }

        $result = [
            'table' => $table,
            'columns' => array_map(
                fn (object $column): array => $this->normaliseColumn((array) $column),
                $columns
            ),
            'indexes' => $this->normaliseIndexes($indexes),
        ];

        return Response::text(json_encode($result, JSON_PRETTY_PRINT));
    }

    public function schema(JsonSchema $schema): array
    {
        return [
            'table' => $schema->string('Database table name to inspect'),
        ];
    }

    /**
     * Validate the table name before interpolating it into SQL.
     */
    private function isValidTableName(string $table): bool
    {
        return (bool) preg_match('/^[A-Za-z0-9_]+$/', $table);
    }

    /**
     * Check whether the table is blocked by configuration or is a system table.
     */
    private function isBlockedTable(string $table): bool
    {
        $blockedTables = Config::get('mcp.database.blocked_tables', []);

        if (in_array($table, $blockedTables, true)) {
            return true;
        }

        $systemTables = ['information_schema', 'mysql', 'performance_schema', 'sys'];

        return in_array(strtolower($table), $systemTables, true);
    }

    /**
     * Normalise a SHOW FULL COLUMNS row into a predictable array shape.
     *
     * @param  array<string, mixed>  $column
     * @return array<string, mixed>
     */
    private function normaliseColumn(array $column): array
    {
        return [
            'field' => $column['Field'] ?? null,
            'type' => $column['Type'] ?? null,
            'collation' => $column['Collation'] ?? null,
            'null' => $column['Null'] ?? null,
            'key' => $column['Key'] ?? null,
            'default' => $column['Default'] ?? null,
            'extra' => $column['Extra'] ?? null,
            'privileges' => $column['Privileges'] ?? null,
            'comment' => $column['Comment'] ?? null,
        ];
    }

    /**
     * Group SHOW INDEX rows by index name.
     *
     * @param  array<int, object>  $indexes
     * @return array<int, array<string, mixed>>
     */
    private function normaliseIndexes(array $indexes): array
    {
        $grouped = [];

        foreach ($indexes as $index) {
            $row = (array) $index;
            $name = (string) ($row['Key_name'] ?? 'unknown');

            if (! isset($grouped[$name])) {
                $grouped[$name] = [
                    'name' => $name,
                    'unique' => ! (bool) ($row['Non_unique'] ?? 1),
                    'type' => $row['Index_type'] ?? null,
                    'columns' => [],
                ];
            }

            $grouped[$name]['columns'][] = [
                'name' => $row['Column_name'] ?? null,
                'order' => $row['Seq_in_index'] ?? null,
                'collation' => $row['Collation'] ?? null,
                'cardinality' => $row['Cardinality'] ?? null,
                'sub_part' => $row['Sub_part'] ?? null,
                'nullable' => $row['Null'] ?? null,
                'comment' => $row['Comment'] ?? null,
            ];
        }

        return array_values($grouped);
    }

    private function errorResponse(string $message): Response
    {
        return Response::text(json_encode([
            'error' => $message,
            'code' => 'VALIDATION_ERROR',
        ]));
    }
}
