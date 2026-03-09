<?php

namespace Core\Mcp\Tools;

use Core\Mod\Content\Enums\ContentType;
use Core\Mod\Content\Models\ContentItem;
use Core\Mod\Content\Models\ContentRevision;
use Core\Mod\Content\Models\ContentTaxonomy;
use Core\Tenant\Models\Workspace;
use Core\Tenant\Services\EntitlementService;
use Illuminate\Contracts\JsonSchema\JsonSchema;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Str;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Tool;

/**
 * MCP tools for managing content items.
 *
 * Part of TASK-004 Phase 3: MCP Integration.
 *
 * Provides functionality for listing, reading, creating, updating,
 * and deleting content items for MCP agents.
 */
class ContentTools extends Tool
{
    protected string $description = 'Manage content items - list, read, create, update, and delete blog posts and pages';

    public function handle(Request $request): Response
    {
        $action = $request->get('action');
        $workspaceSlug = $request->get('workspace');

        // Resolve workspace
        $workspace = $this->resolveWorkspace($workspaceSlug);
        if (! $workspace && in_array($action, ['list', 'read', 'create', 'update', 'delete'])) {
            return Response::text(json_encode([
                'error' => 'Workspace is required. Provide a workspace slug.',
            ]));
        }

        return match ($action) {
            'list' => $this->listContent($workspace, $request),
            'read' => $this->readContent($workspace, $request),
            'create' => $this->createContent($workspace, $request),
            'update' => $this->updateContent($workspace, $request),
            'delete' => $this->deleteContent($workspace, $request),
            'taxonomies' => $this->listTaxonomies($workspace, $request),
            default => Response::text(json_encode([
                'error' => 'Invalid action. Available: list, read, create, update, delete, taxonomies',
            ])),
        };
    }

    /**
     * Resolve workspace from slug.
     */
    protected function resolveWorkspace(?string $slug): ?Workspace
    {
        if (! $slug) {
            return null;
        }

        return Workspace::where('slug', $slug)
            ->orWhere('id', $slug)
            ->first();
    }

    /**
     * Check entitlements for content operations.
     */
    protected function checkEntitlement(Workspace $workspace, string $action): ?array
    {
        $entitlements = app(EntitlementService::class);

        // Check if workspace has content MCP access
        $result = $entitlements->can($workspace, 'content.mcp_access');

        if ($result->isDenied()) {
            return ['error' => $result->reason ?? 'Content MCP access not available in your plan.'];
        }

        // For create operations, check content limits
        if ($action === 'create') {
            $limitResult = $entitlements->can($workspace, 'content.items');
            if ($limitResult->isDenied()) {
                return ['error' => $limitResult->reason ?? 'Content item limit reached.'];
            }
        }

        return null;
    }

    /**
     * List content items for a workspace.
     */
    protected function listContent(Workspace $workspace, Request $request): Response
    {
        $query = ContentItem::forWorkspace($workspace->id)
            ->native()
            ->with(['author', 'taxonomies']);

        // Filter by type (post/page)
        if ($type = $request->get('type')) {
            $query->where('type', $type);
        }

        // Filter by status
        if ($status = $request->get('status')) {
            if ($status === 'published') {
                $query->published();
            } elseif ($status === 'scheduled') {
                $query->scheduled();
            } else {
                $query->where('status', $status);
            }
        }

        // Search
        if ($search = $request->get('search')) {
            $query->where(function ($q) use ($search) {
                $q->where('title', 'like', "%{$search}%")
                    ->orWhere('content_html', 'like', "%{$search}%")
                    ->orWhere('excerpt', 'like', "%{$search}%");
            });
        }

        // Pagination
        $limit = min($request->get('limit', 20), 100);
        $offset = $request->get('offset', 0);

        $total = $query->count();
        $items = $query->orderByDesc('updated_at')
            ->skip($offset)
            ->take($limit)
            ->get();

        $result = [
            'items' => $items->map(fn (ContentItem $item) => [
                'id' => $item->id,
                'slug' => $item->slug,
                'title' => $item->title,
                'type' => $item->type,
                'status' => $item->status,
                'excerpt' => Str::limit($item->excerpt, 200),
                'author' => $item->author?->name,
                'categories' => $item->categories->pluck('name')->all(),
                'tags' => $item->tags->pluck('name')->all(),
                'word_count' => str_word_count(strip_tags($item->content_html ?? '')),
                'publish_at' => $item->publish_at?->toIso8601String(),
                'created_at' => $item->created_at->toIso8601String(),
                'updated_at' => $item->updated_at->toIso8601String(),
            ]),
            'total' => $total,
            'limit' => $limit,
            'offset' => $offset,
        ];

        return Response::text(json_encode($result, JSON_PRETTY_PRINT));
    }

    /**
     * Read full content of an item.
     */
    protected function readContent(Workspace $workspace, Request $request): Response
    {
        $identifier = $request->get('identifier');

        if (! $identifier) {
            return Response::text(json_encode(['error' => 'identifier (slug or ID) is required']));
        }

        $query = ContentItem::forWorkspace($workspace->id)->native();

        // Find by ID, slug, or wp_id
        if (is_numeric($identifier)) {
            $item = $query->where('id', $identifier)
                ->orWhere('wp_id', $identifier)
                ->first();
        } else {
            $item = $query->where('slug', $identifier)->first();
        }

        if (! $item) {
            return Response::text(json_encode(['error' => 'Content not found']));
        }

        // Load relationships
        $item->load(['author', 'taxonomies', 'revisions' => fn ($q) => $q->latest()->limit(5)]);

        // Return as markdown with frontmatter for AI context
        $format = $request->get('format', 'json');

        if ($format === 'markdown') {
            $markdown = $this->contentToMarkdown($item);

            return Response::text($markdown);
        }

        $result = [
            'id' => $item->id,
            'slug' => $item->slug,
            'title' => $item->title,
            'type' => $item->type,
            'status' => $item->status,
            'excerpt' => $item->excerpt,
            'content_html' => $item->content_html,
            'content_markdown' => $item->content_markdown,
            'author' => [
                'id' => $item->author?->id,
                'name' => $item->author?->name,
            ],
            'categories' => $item->categories->map(fn ($t) => [
                'id' => $t->id,
                'slug' => $t->slug,
                'name' => $t->name,
            ])->all(),
            'tags' => $item->tags->map(fn ($t) => [
                'id' => $t->id,
                'slug' => $t->slug,
                'name' => $t->name,
            ])->all(),
            'seo_meta' => $item->seo_meta,
            'publish_at' => $item->publish_at?->toIso8601String(),
            'revision_count' => $item->revision_count,
            'recent_revisions' => $item->revisions->map(fn ($r) => [
                'id' => $r->id,
                'revision_number' => $r->revision_number,
                'change_type' => $r->change_type,
                'created_at' => $r->created_at->toIso8601String(),
            ])->all(),
            'created_at' => $item->created_at->toIso8601String(),
            'updated_at' => $item->updated_at->toIso8601String(),
        ];

        return Response::text(json_encode($result, JSON_PRETTY_PRINT));
    }

    /**
     * Create new content.
     */
    protected function createContent(Workspace $workspace, Request $request): Response
    {
        // Check entitlements
        $entitlementError = $this->checkEntitlement($workspace, 'create');
        if ($entitlementError) {
            return Response::text(json_encode($entitlementError));
        }

        // Validate required fields
        $title = $request->get('title');
        if (! $title) {
            return Response::text(json_encode(['error' => 'title is required']));
        }

        $type = $request->get('type', 'post');
        if (! in_array($type, ['post', 'page'])) {
            return Response::text(json_encode(['error' => 'type must be post or page']));
        }

        $status = $request->get('status', 'draft');
        if (! in_array($status, ['draft', 'publish', 'future', 'private'])) {
            return Response::text(json_encode(['error' => 'status must be draft, publish, future, or private']));
        }

        // Generate slug
        $slug = $request->get('slug') ?: Str::slug($title);
        $baseSlug = $slug;
        $counter = 1;

        // Ensure unique slug within workspace
        while (ContentItem::forWorkspace($workspace->id)->where('slug', $slug)->exists()) {
            $slug = $baseSlug.'-'.$counter++;
        }

        // Parse markdown content if provided
        $content = $request->get('content', '');
        $contentHtml = $request->get('content_html');
        $contentMarkdown = $request->get('content_markdown', $content);

        // Convert markdown to HTML if only markdown provided
        if ($contentMarkdown && ! $contentHtml) {
            $contentHtml = Str::markdown($contentMarkdown);
        }

        // Handle scheduling
        $publishAt = null;
        if ($status === 'future') {
            $publishAt = $request->get('publish_at');
            if (! $publishAt) {
                return Response::text(json_encode(['error' => 'publish_at is required for scheduled content']));
            }
            $publishAt = \Carbon\Carbon::parse($publishAt);
        }

        // Create content item
        $item = ContentItem::create([
            'workspace_id' => $workspace->id,
            'content_type' => ContentType::NATIVE,
            'type' => $type,
            'status' => $status,
            'slug' => $slug,
            'title' => $title,
            'excerpt' => $request->get('excerpt'),
            'content_html' => $contentHtml,
            'content_markdown' => $contentMarkdown,
            'seo_meta' => $request->get('seo_meta'),
            'publish_at' => $publishAt,
            'last_edited_by' => Auth::id(),
        ]);

        // Handle categories
        if ($categories = $request->get('categories')) {
            $categoryIds = $this->resolveOrCreateTaxonomies($workspace, $categories, 'category');
            $item->taxonomies()->attach($categoryIds);
        }

        // Handle tags
        if ($tags = $request->get('tags')) {
            $tagIds = $this->resolveOrCreateTaxonomies($workspace, $tags, 'tag');
            $item->taxonomies()->attach($tagIds);
        }

        // Create initial revision
        $item->createRevision(Auth::user(), ContentRevision::CHANGE_EDIT, 'Created via MCP');

        // Record usage
        $entitlements = app(EntitlementService::class);
        $entitlements->recordUsage($workspace, 'content.items', 1, Auth::user(), [
            'source' => 'mcp',
            'content_id' => $item->id,
        ]);

        return Response::text(json_encode([
            'ok' => true,
            'item' => [
                'id' => $item->id,
                'slug' => $item->slug,
                'title' => $item->title,
                'type' => $item->type,
                'status' => $item->status,
                'url' => $this->getContentUrl($workspace, $item),
            ],
        ], JSON_PRETTY_PRINT));
    }

    /**
     * Update existing content.
     */
    protected function updateContent(Workspace $workspace, Request $request): Response
    {
        $identifier = $request->get('identifier');

        if (! $identifier) {
            return Response::text(json_encode(['error' => 'identifier (slug or ID) is required']));
        }

        $query = ContentItem::forWorkspace($workspace->id)->native();

        if (is_numeric($identifier)) {
            $item = $query->find($identifier);
        } else {
            $item = $query->where('slug', $identifier)->first();
        }

        if (! $item) {
            return Response::text(json_encode(['error' => 'Content not found']));
        }

        // Build update data
        $updateData = [];

        if ($request->has('title')) {
            $updateData['title'] = $request->get('title');
        }

        if ($request->has('excerpt')) {
            $updateData['excerpt'] = $request->get('excerpt');
        }

        if ($request->has('content') || $request->has('content_markdown')) {
            $contentMarkdown = $request->get('content_markdown') ?? $request->get('content');
            $updateData['content_markdown'] = $contentMarkdown;
            $updateData['content_html'] = $request->get('content_html') ?? Str::markdown($contentMarkdown);
        }

        if ($request->has('content_html') && ! $request->has('content_markdown')) {
            $updateData['content_html'] = $request->get('content_html');
        }

        if ($request->has('status')) {
            $status = $request->get('status');
            if (! in_array($status, ['draft', 'publish', 'future', 'private'])) {
                return Response::text(json_encode(['error' => 'status must be draft, publish, future, or private']));
            }
            $updateData['status'] = $status;

            if ($status === 'future' && $request->has('publish_at')) {
                $updateData['publish_at'] = \Carbon\Carbon::parse($request->get('publish_at'));
            }
        }

        if ($request->has('seo_meta')) {
            $updateData['seo_meta'] = $request->get('seo_meta');
        }

        if ($request->has('slug')) {
            $newSlug = $request->get('slug');
            if ($newSlug !== $item->slug) {
                // Check uniqueness
                if (ContentItem::forWorkspace($workspace->id)->where('slug', $newSlug)->where('id', '!=', $item->id)->exists()) {
                    return Response::text(json_encode(['error' => 'Slug already exists']));
                }
                $updateData['slug'] = $newSlug;
            }
        }

        $updateData['last_edited_by'] = Auth::id();

        // Update item
        $item->update($updateData);

        // Handle categories
        if ($request->has('categories')) {
            $categoryIds = $this->resolveOrCreateTaxonomies($workspace, $request->get('categories'), 'category');
            $item->categories()->sync($categoryIds);
        }

        // Handle tags
        if ($request->has('tags')) {
            $tagIds = $this->resolveOrCreateTaxonomies($workspace, $request->get('tags'), 'tag');
            $item->tags()->sync($tagIds);
        }

        // Create revision
        $changeSummary = $request->get('change_summary', 'Updated via MCP');
        $item->createRevision(Auth::user(), ContentRevision::CHANGE_EDIT, $changeSummary);

        $item->refresh()->load(['author', 'taxonomies']);

        return Response::text(json_encode([
            'ok' => true,
            'item' => [
                'id' => $item->id,
                'slug' => $item->slug,
                'title' => $item->title,
                'type' => $item->type,
                'status' => $item->status,
                'revision_count' => $item->revision_count,
                'url' => $this->getContentUrl($workspace, $item),
            ],
        ], JSON_PRETTY_PRINT));
    }

    /**
     * Delete content (soft delete).
     */
    protected function deleteContent(Workspace $workspace, Request $request): Response
    {
        $identifier = $request->get('identifier');

        if (! $identifier) {
            return Response::text(json_encode(['error' => 'identifier (slug or ID) is required']));
        }

        $query = ContentItem::forWorkspace($workspace->id)->native();

        if (is_numeric($identifier)) {
            $item = $query->find($identifier);
        } else {
            $item = $query->where('slug', $identifier)->first();
        }

        if (! $item) {
            return Response::text(json_encode(['error' => 'Content not found']));
        }

        // Create final revision before delete
        $item->createRevision(Auth::user(), ContentRevision::CHANGE_EDIT, 'Deleted via MCP');

        // Soft delete
        $item->delete();

        return Response::text(json_encode([
            'ok' => true,
            'deleted' => [
                'id' => $item->id,
                'slug' => $item->slug,
                'title' => $item->title,
            ],
        ], JSON_PRETTY_PRINT));
    }

    /**
     * List taxonomies (categories and tags).
     */
    protected function listTaxonomies(Workspace $workspace, Request $request): Response
    {
        $type = $request->get('type'); // category or tag

        $query = ContentTaxonomy::where('workspace_id', $workspace->id);

        if ($type) {
            $query->where('type', $type);
        }

        $taxonomies = $query->orderBy('name')->get();

        $result = [
            'taxonomies' => $taxonomies->map(fn ($t) => [
                'id' => $t->id,
                'type' => $t->type,
                'slug' => $t->slug,
                'name' => $t->name,
                'description' => $t->description,
            ])->all(),
            'total' => $taxonomies->count(),
        ];

        return Response::text(json_encode($result, JSON_PRETTY_PRINT));
    }

    /**
     * Resolve or create taxonomies from slugs/names.
     */
    protected function resolveOrCreateTaxonomies(Workspace $workspace, array $items, string $type): array
    {
        $ids = [];

        foreach ($items as $item) {
            $taxonomy = ContentTaxonomy::where('workspace_id', $workspace->id)
                ->where('type', $type)
                ->where(function ($q) use ($item) {
                    $q->where('slug', $item)
                        ->orWhere('name', $item);
                })
                ->first();

            if (! $taxonomy) {
                // Create new taxonomy
                $taxonomy = ContentTaxonomy::create([
                    'workspace_id' => $workspace->id,
                    'type' => $type,
                    'slug' => Str::slug($item),
                    'name' => $item,
                ]);
            }

            $ids[] = $taxonomy->id;
        }

        return $ids;
    }

    /**
     * Convert content item to markdown with frontmatter.
     */
    protected function contentToMarkdown(ContentItem $item): string
    {
        $frontmatter = [
            'title' => $item->title,
            'slug' => $item->slug,
            'type' => $item->type,
            'status' => $item->status,
            'author' => $item->author?->name,
            'categories' => $item->categories->pluck('name')->all(),
            'tags' => $item->tags->pluck('name')->all(),
            'created_at' => $item->created_at->toIso8601String(),
            'updated_at' => $item->updated_at->toIso8601String(),
        ];

        if ($item->publish_at) {
            $frontmatter['publish_at'] = $item->publish_at->toIso8601String();
        }

        if ($item->seo_meta) {
            $frontmatter['seo'] = $item->seo_meta;
        }

        $yaml = "---\n";
        foreach ($frontmatter as $key => $value) {
            if (is_array($value)) {
                $yaml .= "{$key}: ".json_encode($value)."\n";
            } else {
                $yaml .= "{$key}: {$value}\n";
            }
        }
        $yaml .= "---\n\n";

        // Prefer markdown content, fall back to stripping HTML
        $content = $item->content_markdown ?? strip_tags($item->content_html ?? '');

        return $yaml.$content;
    }

    /**
     * Get the public URL for content.
     */
    protected function getContentUrl(Workspace $workspace, ContentItem $item): string
    {
        $domain = $workspace->domain ?? config('app.url');
        $path = $item->type === 'post' ? "/blog/{$item->slug}" : "/{$item->slug}";

        return "https://{$domain}{$path}";
    }

    public function schema(JsonSchema $schema): array
    {
        return [
            'action' => $schema->string('Action: list, read, create, update, delete, taxonomies'),
            'workspace' => $schema->string('Workspace slug (required for most actions)')->nullable(),
            'identifier' => $schema->string('Content slug or ID (for read, update, delete)')->nullable(),
            'type' => $schema->string('Content type: post or page (for list filter or create)')->nullable(),
            'status' => $schema->string('Content status: draft, publish, future, private')->nullable(),
            'search' => $schema->string('Search term for list action')->nullable(),
            'limit' => $schema->integer('Max items to return (default 20, max 100)')->nullable(),
            'offset' => $schema->integer('Offset for pagination')->nullable(),
            'format' => $schema->string('Output format: json or markdown (for read action)')->nullable(),
            'title' => $schema->string('Content title (for create/update)')->nullable(),
            'slug' => $schema->string('URL slug (for create/update)')->nullable(),
            'excerpt' => $schema->string('Content excerpt/summary')->nullable(),
            'content' => $schema->string('Content body as markdown (for create/update)')->nullable(),
            'content_html' => $schema->string('Content body as HTML (optional, auto-generated from markdown)')->nullable(),
            'content_markdown' => $schema->string('Content body as markdown (alias for content)')->nullable(),
            'categories' => $schema->array('Array of category slugs or names')->nullable(),
            'tags' => $schema->array('Array of tag strings')->nullable(),
            'seo_meta' => $schema->array('SEO metadata: {title, description, keywords}')->nullable(),
            'publish_at' => $schema->string('ISO datetime for scheduled publishing (status=future)')->nullable(),
            'change_summary' => $schema->string('Summary of changes for revision history (update action)')->nullable(),
        ];
    }
}
