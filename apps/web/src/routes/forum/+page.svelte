<script lang="ts">
	// Migrated from ../dr-toke/apps/web/app/forum/page.tsx (community tier,
	// so it keeps the exact dr-toke look).
	import CatalogShell from '$sections/products/CatalogShell.svelte';
	import { Remote } from '$lib/api/remote.svelte';

	interface ForumPostSummary {
		id: string;
		slug: string;
		title: string;
		author_note: string | null;
		meta_desc: string | null;
		tags: string[];
		created_at: string;
		comment_count: number;
	}

	const remote = new Remote<{ posts: ForumPostSummary[] }>();
	$effect(() => remote.load('/api/forum/posts'));

	const posts = $derived(remote.data?.posts ?? []);

	function fmtDate(iso: string): string {
		const d = new Date(iso);
		if (Number.isNaN(d.getTime())) return '';
		return d.toLocaleDateString('en-IN', { year: 'numeric', month: 'short', day: 'numeric' });
	}
</script>

<svelte:head>
	<title>Forum & Editorial — Dr Toke</title>
	<meta
		name="description"
		content="Long-form pieces from the Dr Toke team — harm reduction, law, product science — open for community discussion. Pseudonymous, no PII."
	/>
</svelte:head>

<CatalogShell>
	<main class="max-w-[900px] mx-auto px-8 py-20">
		<p class="text-[0.68rem] tracking-[0.22em] uppercase text-gold font-semibold font-display mb-1">
			The Commons
		</p>
		<h1
			class="font-display font-bold text-cream leading-[1.15] mb-[0.45rem]"
			style="font-size: clamp(1.9rem, 3.8vw, 2.9rem);"
		>
			Forum &amp; Editorial
		</h1>
		<p class="text-cream2 text-base max-w-[600px] leading-[1.7] mb-10">
			Long-form pieces from the Dr Toke team — harm reduction, law, product science — open for
			community discussion. Pseudonymous, no PII.
		</p>

		{#if remote.error}
			<p class="text-cream2 py-12 text-center">
				Could not load the forum. The data service may be offline.
			</p>
		{:else if remote.loading}
			<div class="flex flex-col gap-4">
				{#each [0, 1, 2] as i (i)}
					<div class="h-28 rounded-[5px] animate-pulse" style="background: var(--bg2);"></div>
				{/each}
			</div>
		{:else if posts.length === 0}
			<p class="text-cream2 py-12 text-center opacity-70">No posts published yet. Check back soon.</p>
		{:else}
			<div class="flex flex-col gap-4">
				{#each posts as post, i (post.id)}
					<a
						href={`/forum/post?slug=${encodeURIComponent(post.slug)}`}
						class={`card-hover reveal reveal-delay-${Math.min(i + 1, 5)} rounded-[5px] p-5 block`}
						style="background: var(--bg2); border: 1px solid rgba(58,122,47,0.25);"
					>
						<div class="flex items-center gap-2 mb-2 flex-wrap">
							{#each post.tags as t (t)}
								<span
									class="text-[0.58rem] uppercase tracking-[0.06em] px-2 py-[2px] rounded-[2px] text-gold"
									style="background: rgba(200,168,75,0.1); border: 1px solid rgba(200,168,75,0.3);"
								>
									{t}
								</span>
							{/each}
						</div>
						<h2 class="font-display font-bold text-cream text-[1.3rem] leading-tight mb-1">
							{post.title}
						</h2>
						{#if post.meta_desc}
							<p class="text-cream2 text-[0.88rem] leading-relaxed mb-2">{post.meta_desc}</p>
						{/if}
						<p class="text-[0.72rem] text-cream2 opacity-70">
							{post.author_note ?? 'Dr Toke'} · {fmtDate(post.created_at)} · {post.comment_count}
							comment{post.comment_count === 1 ? '' : 's'}
						</p>
					</a>
				{/each}
			</div>
		{/if}
	</main>
</CatalogShell>
