<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasTable('mcp_tool_versions')) {
            Schema::create('mcp_tool_versions', function (Blueprint $table) {
                $table->id();
                $table->string('server_id', 64)->index();
                $table->string('tool_name', 128);
                $table->string('version', 32); // semver: 1.0.0, 2.1.0-beta, etc.
                $table->json('input_schema')->nullable();
                $table->json('output_schema')->nullable();
                $table->text('description')->nullable();
                $table->text('changelog')->nullable();
                $table->text('migration_notes')->nullable(); // guidance for upgrading from previous version
                $table->boolean('is_latest')->default(false);
                $table->timestamp('deprecated_at')->nullable();
                $table->timestamp('sunset_at')->nullable(); // after this date, version is blocked
                $table->timestamps();

                // Unique constraint: one version per tool per server
                $table->unique(['server_id', 'tool_name', 'version'], 'mcp_tool_versions_unique');

                // Index for finding latest versions
                $table->index(['server_id', 'tool_name', 'is_latest'], 'mcp_tool_versions_latest');

                // Index for finding deprecated/sunset versions
                $table->index(['deprecated_at', 'sunset_at'], 'mcp_tool_versions_lifecycle');
            });
        }
    }

    public function down(): void
    {
        Schema::dropIfExists('mcp_tool_versions');
    }
};
