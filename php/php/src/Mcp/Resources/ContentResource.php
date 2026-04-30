<?php

namespace Core\Mcp\Resources;

use Core\Mod\Content\Models\ContentItem;
use Core\Tenant\Models\Workspace;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Resource;

/**
 * MCP Resource for content items.
 *
 * Part of TASK-004 Phase 3: MCP Integration.
 *
 * URI format: content://{workspace}/{slug}
 * Returns content as markdown for AI context.
 */
class ContentResource extends Resource
{
    protected string $description = 'Content items from the CMS - returns markdown for AI context';

    public function handle(Request $request): Response
    {
        $uri = $request->get('uri', '');

        // Parse URI: content://{workspace}/{slug}
        if (! str_starts_with($uri, 'content://')) {
            return Response::text('Invalid URI format. Expected: content://{workspace}/{slug}');
        }

        $path = substr($uri, 10); // Remove 'content://'
        $parts = explode('/', $path, 2);

        if (count($parts) < 2) {
            return Response::text('Invalid URI format. Expected: content://{workspace}/{slug}');
        }

        [$workspaceSlug, $contentSlug] = $parts;

        // Resolve workspace
        $workspace = Workspace::where('slug', $workspaceSlug)
            ->orWhere('id', $workspaceSlug)
            ->first();

        if (! $workspace) {
            return Response::text("Workspace not found: {$workspaceSlug}");
        }

        // Find content item
        $item = ContentItem::forWorkspace($workspace->id)
            ->native()
            ->where('slug', $contentSlug)
            ->first();

        if (! $item) {
            // Try by ID
            if (is_numeric($contentSlug)) {
                $item = ContentItem::forWorkspace($workspace->id)
                    ->native()
                    ->find($contentSlug);
            }
        }

        if (! $item) {
            return Response::text("Content not found: {$contentSlug}");
        }

        // Load relationships
        $item->load(['author', 'taxonomies']);

        // Return as markdown with frontmatter
        $markdown = $this->contentToMarkdown($item, $workspace);

        return Response::text($markdown);
    }

    /**
     * Convert content item to markdown with frontmatter.
     */
    protected function contentToMarkdown(ContentItem $item, Workspace $workspace): string
    {
        $md = "---\n";
        $md .= "title: \"{$item->title}\"\n";
        $md .= "slug: {$item->slug}\n";
        $md .= "workspace: {$workspace->slug}\n";
        $md .= "type: {$item->type}\n";
        $md .= "status: {$item->status}\n";

        if ($item->author) {
            $md .= "author: {$item->author->name}\n";
        }

        $categories = $item->categories->pluck('name')->all();
        if (! empty($categories)) {
            $md .= 'categories: ['.implode(', ', $categories)."]\n";
        }

        $tags = $item->tags->pluck('name')->all();
        if (! empty($tags)) {
            $md .= 'tags: ['.implode(', ', $tags)."]\n";
        }

        if ($item->publish_at) {
            $md .= 'publish_at: '.$item->publish_at->toIso8601String()."\n";
        }

        $md .= 'created_at: '.$item->created_at->toIso8601String()."\n";
        $md .= 'updated_at: '.$item->updated_at->toIso8601String()."\n";

        if ($item->seo_meta) {
            if (isset($item->seo_meta['title'])) {
                $md .= "seo_title: \"{$item->seo_meta['title']}\"\n";
            }
            if (isset($item->seo_meta['description'])) {
                $md .= "seo_description: \"{$item->seo_meta['description']}\"\n";
            }
        }

        $md .= "---\n\n";

        // Add excerpt if available
        if ($item->excerpt) {
            $md .= "> {$item->excerpt}\n\n";
        }

        // Prefer markdown content, fall back to stripping HTML (clean > original)
        $content = $item->content_markdown
            ?? strip_tags($item->content_html_clean ?? $item->content_html_original ?? '');
        $md .= $content;

        return $md;
    }

    /**
     * Get list of available content resources.
     *
     * This is called when MCP lists available resources.
     */
    public static function list(): array
    {
        $resources = [];

        // Get all workspaces with content
        $workspaces = Workspace::whereHas('contentItems', function ($q) {
            $q->native()->where('status', 'publish');
        })->get();

        foreach ($workspaces as $workspace) {
            // Get published content for this workspace
            $items = ContentItem::forWorkspace($workspace->id)
                ->native()
                ->published()
                ->orderByDesc('updated_at')
                ->limit(50)
                ->get(['id', 'slug', 'title', 'type']);

            foreach ($items as $item) {
                $resources[] = [
                    'uri' => "content://{$workspace->slug}/{$item->slug}",
                    'name' => $item->title,
                    'description' => ucfirst($item->type).": {$item->title}",
                    'mimeType' => 'text/markdown',
                ];
            }
        }

        return $resources;
    }
}
