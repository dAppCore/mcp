<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mcp\Models\McpToolVersion;
use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

/**
 * Add a sortable version key to mcp_tool_versions.
 *
 * scopeOrderByVersion sorted with three nested SUBSTRING_INDEX calls, which is
 * MySQL-only — the scope could not execute on sqlite at all, so nothing that
 * ordered versions was testable. Storing a normalised key moves the parsing to
 * write time and leaves every read a plain indexed ORDER BY, rather than
 * spreading driver-specific SQL through every query that touches versions.
 *
 * The human-readable `version` column stays the display truth; this column is
 * only ever compared.
 */
return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasTable('mcp_tool_versions')) {
            return;
        }

        if (! Schema::hasColumn('mcp_tool_versions', 'version_sort')) {
            Schema::table('mcp_tool_versions', function (Blueprint $table) {
                $table->string('version_sort', 64)->nullable()->after('version');

                $table->index(['server_id', 'tool_name', 'version_sort'], 'mcp_tool_versions_sort');
            });
        }

        // Backfill existing rows, chunked rather than loaded whole: this table
        // grows a row per tool per release and a deployed instance may hold a
        // lot of them.
        //
        // chunkById, not chunk: the filter is whereNull('version_sort') and the
        // loop fills that same column in, so each processed row leaves the
        // result set. chunk() pages with OFFSET, so the set shrinking under it
        // would skip a whole page's worth of rows for every page written —
        // silently backfilling roughly half the table. chunkById pages with
        // `where id > <last seen>`, which is stable under that mutation.
        DB::table('mcp_tool_versions')
            ->select('id', 'version')
            ->whereNull('version_sort')
            ->chunkById(500, function ($rows) {
                foreach ($rows as $row) {
                    DB::table('mcp_tool_versions')
                        ->where('id', $row->id)
                        ->update(['version_sort' => McpToolVersion::versionSortKey((string) $row->version)]);
                }
            });
    }

    public function down(): void
    {
        if (! Schema::hasColumn('mcp_tool_versions', 'version_sort')) {
            return;
        }

        Schema::table('mcp_tool_versions', function (Blueprint $table) {
            $table->dropIndex('mcp_tool_versions_sort');
            $table->dropColumn('version_sort');
        });
    }
};
