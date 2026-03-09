<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasTable('mcp_tool_metrics')) {
            Schema::create('mcp_tool_metrics', function (Blueprint $table) {
                $table->id();
                $table->string('tool_name');
                $table->string('workspace_id')->nullable();
                $table->unsignedInteger('call_count')->default(0);
                $table->unsignedInteger('error_count')->default(0);
                $table->unsignedInteger('total_duration_ms')->default(0);
                $table->unsignedInteger('min_duration_ms')->nullable();
                $table->unsignedInteger('max_duration_ms')->nullable();
                $table->date('date');
                $table->timestamps();

                $table->unique(['tool_name', 'workspace_id', 'date']);
                $table->index(['date', 'tool_name']);
                $table->index('workspace_id');
            });
        }

        // Table for tracking tool combinations (tools used together in sessions)
        if (! Schema::hasTable('mcp_tool_combinations')) {
            Schema::create('mcp_tool_combinations', function (Blueprint $table) {
                $table->id();
                $table->string('tool_a');
                $table->string('tool_b');
                $table->string('workspace_id')->nullable();
                $table->unsignedInteger('occurrence_count')->default(0);
                $table->date('date');
                $table->timestamps();

                $table->unique(['tool_a', 'tool_b', 'workspace_id', 'date']);
                $table->index(['date', 'occurrence_count']);
            });
        }
    }

    public function down(): void
    {
        Schema::dropIfExists('mcp_tool_combinations');
        Schema::dropIfExists('mcp_tool_metrics');
    }
};
