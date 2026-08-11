<script lang="ts">
	// Port of BrandsGrid.tsx + BrandFilter.tsx from ../dr-toke/apps/web.
	// Filter state persists in the URL (?filter=) for sharability.
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Remote } from '$lib/api/remote.svelte';
	import type { BrandListResponse } from '$lib/api/catalog';
	import BrandCard from './BrandCard.svelte';

	const FILTERS: { key: string; label: string }[] = [
		{ key: 'all', label: 'All' },
		{ key: 'tincture', label: 'Tinctures' },
		{ key: 'edible', label: 'Edibles' },
		{ key: 'topical', label: 'Topicals' },
		{ key: 'vapeable', label: 'Vapeables' },
		{ key: 'smokable', label: 'Smokables' },
		{ key: 'extract', label: 'Extracts' },
		{ key: 'beverage', label: 'Beverages' },
		{ key: 'nutrition', label: 'Nutrition' },
		{ key: 'pet', label: 'Pet' }
	];

	// This route is ssr=false, so reading searchParams is client-only-safe.
	const filter = $derived(page.url.searchParams.get('filter') ?? 'all');

	const remote = new Remote<BrandListResponse>();
	$effect(() => remote.load('/api/brands'));

	const brands = $derived(remote.data?.brands ?? []);

	// Categories actually present across loaded brands → drive the filter chips.
	const available = $derived(Array.from(new Set(brands.flatMap((b) => b.categories))).sort());

	const filtered = $derived(
		filter === 'all' ? brands : brands.filter((b) => b.categories.includes(filter))
	);

	const shown = $derived(FILTERS.filter((f) => f.key === 'all' || available.includes(f.key)));

	function setFilter(key: string) {
		const params = new URLSearchParams(page.url.searchParams.toString());
		if (key === 'all') params.delete('filter');
		else params.set('filter', key);
		goto(`/brands?${params.toString()}`, { noScroll: true, keepFocus: true });
	}
</script>

<div class="reveal flex flex-wrap gap-[0.55rem] mb-8">
	{#each shown as f (f.key)}
		{@const active = filter === f.key || (f.key === 'all' && filter === 'all')}
		<button
			onclick={() => setFilter(f.key)}
			class="text-[0.8rem] px-[0.9rem] py-[0.35rem] rounded-[2px] tracking-[0.05em] font-medium transition-all duration-200 cursor-pointer border"
			style={`background: ${active ? 'var(--gold)' : 'transparent'}; color: ${active ? 'var(--bg)' : 'var(--cream2)'}; border-color: ${active ? 'var(--gold)' : 'rgba(58,122,47,0.4)'}; font-family: var(--font-body), sans-serif;`}
			aria-pressed={active}
		>
			{f.label}
		</button>
	{/each}
</div>

{#if remote.error}
	<p class="text-cream2 text-base py-12 text-center">
		Could not load the brand directory. The data service may be offline.{' '}
		<span class="opacity-60">({remote.error.message})</span>
	</p>
{:else if remote.loading}
	<div class="grid gap-[1.1rem]" style="grid-template-columns: repeat(auto-fill, minmax(310px, 1fr));">
		{#each Array(6) as _, i (i)}
			<div
				class="rounded-[4px] h-[220px] animate-pulse"
				style="background: var(--bg2); border: 1px solid rgba(58,122,47,0.15);"
			></div>
		{/each}
	</div>
{:else if filtered.length > 0}
	<div class="grid gap-[1.1rem]" style="grid-template-columns: repeat(auto-fill, minmax(310px, 1fr));">
		{#each filtered as b, i (b.id)}
			<BrandCard brand={b} index={i} />
		{/each}
	</div>
{:else}
	<p class="text-cream2 text-base py-12 text-center">
		No brands in this category yet.{' '}
		<a href="/brands" class="text-gold hover:underline">View all brands →</a>
	</p>
{/if}
