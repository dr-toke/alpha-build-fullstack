<script lang="ts">
	// Migrated from ../dr-toke/apps/web/app/forum/post/page.tsx.
	import { page } from '$app/state';
	import CatalogShell from '$sections/products/CatalogShell.svelte';
	import CommentThread from '$sections/community/CommentThread.svelte';
	import { Remote } from '$lib/api/remote.svelte';
	import { renderMarkdown } from '$lib/api/markdown';

	interface ForumPost {
		id: string;
		slug: string;
		title: string;
		body: string;
		author_note: string | null;
		tags: string[];
		created_at: string;
	}

	const slug = $derived(page.url.searchParams.get('slug'));

	const remote = new Remote<{ post: ForumPost }>();
	$effect(() => remote.load(slug ? `/api/forum/posts/${encodeURIComponent(slug)}` : null));

	function fmtDate(iso: string): string {
		const d = new Date(iso);
		if (Number.isNaN(d.getTime())) return '';
		return d.toLocaleDateString('en-IN', { year: 'numeric', month: 'long', day: 'numeric' });
	}
</script>

<svelte:head>
	<title>{remote.data?.post.title ?? 'Forum'} — Dr Toke</title>
</svelte:head>

<CatalogShell>
	<main class="max-w-[760px] mx-auto px-8 py-16">
		{#if !slug}
			<p class="text-cream2 py-20 text-center">No post specified.</p>
		{:else if remote.loading}
			<div class="py-10">
				<div class="h-10 w-3/4 rounded animate-pulse mb-4" style="background: var(--bg2);"></div>
				<div class="h-64 rounded animate-pulse" style="background: var(--bg2);"></div>
			</div>
		{:else if remote.error || !remote.data}
			<p class="text-cream2 py-20 text-center">
				Post not found.
				<a href="/forum" class="text-gold hover:underline">Back to forum</a>
			</p>
		{:else}
			{@const post = remote.data.post}

			<a href="/forum" class="text-[0.78rem] text-gold hover:underline">← Back to forum</a>

			<article class="mt-4">
				<div class="flex items-center gap-2 mb-3 flex-wrap">
					{#each post.tags as t (t)}
						<span
							class="text-[0.58rem] uppercase tracking-[0.06em] px-2 py-[2px] rounded-[2px] text-gold"
							style="background: rgba(200,168,75,0.1); border: 1px solid rgba(200,168,75,0.3);"
						>
							{t}
						</span>
					{/each}
				</div>

				<h1
					class="font-display font-bold text-cream leading-[1.12] mb-2"
					style="font-size: clamp(2rem, 4vw, 3rem);"
				>
					{post.title}
				</h1>
				<p class="text-[0.78rem] text-cream2 opacity-70 mb-8">
					{post.author_note ?? 'Dr Toke'} · {fmtDate(post.created_at)}
				</p>

				<!-- Safe: renderMarkdown escapes ALL input HTML first, then emits only its
				     own markup (http/https links only). Body is admin-authored. -->
				<div class="prose-toke text-cream2">
					<!-- eslint-disable-next-line svelte/no-at-html-tags -->
					{@html renderMarkdown(post.body)}
				</div>
			</article>

			<CommentThread targetType="post" targetId={post.slug} />
		{/if}
	</main>
</CatalogShell>
