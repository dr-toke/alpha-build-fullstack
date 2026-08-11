<script lang="ts">
	// Exact port of ../dr-toke/apps/web/components/products/CatalogGrid.tsx:
	// filter state lives in the URL (shareable), any filter change resets to
	// page 1, data comes from GET /api/products.
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Remote } from '$lib/api/remote.svelte';
	import type { ProductListResponse } from '$lib/api/catalog';
	import ProductCard from './ProductCard.svelte';

	const CATEGORIES = [
		'tincture',
		'edible',
		'topical',
		'smokable',
		'vapeable',
		'extract',
		'beverage',
		'nutrition',
		'pet'
	] as const;

	const EXTRACTS = [
		{ key: '', label: 'All extracts' },
		{ key: 'full_spectrum', label: 'Full Spectrum' },
		{ key: 'broad_spectrum', label: 'Broad Spectrum' },
		{ key: 'isolate', label: 'Isolate' },
		{ key: 'vijaya', label: 'Vijaya' }
	] as const;

	const SORTS = [
		{ key: 'value', label: 'Best ₹/mg' },
		{ key: 'price', label: 'Lowest price' },
		{ key: 'new', label: 'Newest' }
	] as const;

	// Scopes ₹/mg + the value sort to one cannabinoid, so CBD and THC products
	// aren't ranked against each other on incomparable bases.
	const BASES = [
		{ key: '', label: '₹/mg: dominant' },
		{ key: 'cbd', label: '₹/mg: CBD only' },
		{ key: 'thc', label: '₹/mg: THC only' }
	] as const;

	const PER_PAGE = 24;

	// This route is ssr=false, so reading searchParams is client-only-safe.
	const category = $derived(page.url.searchParams.get('category') ?? '');
	const extract = $derived(page.url.searchParams.get('extract') ?? '');
	const brand = $derived(page.url.searchParams.get('brand') ?? '');
	const sort = $derived(page.url.searchParams.get('sort') ?? 'value');
	const basis = $derived(page.url.searchParams.get('basis') ?? '');
	const verifiedOnly = $derived(page.url.searchParams.get('verified') === 'true');
	const pageNum = $derived(Math.max(1, parseInt(page.url.searchParams.get('page') ?? '1', 10) || 1));

	function setParam(updates: Record<string, string | null>) {
		const next = new URLSearchParams(page.url.searchParams.toString());
		for (const [k, v] of Object.entries(updates)) {
			if (v === null || v === '') next.delete(k);
			else next.set(k, v);
		}
		// Any filter change resets to page 1.
		if (!('page' in updates)) next.delete('page');
		goto(`/products?${next.toString()}`, { noScroll: true, keepFocus: true });
	}

	// Build API query.
	const query = $derived.by(() => {
		const q = new URLSearchParams({ sort, limit: String(PER_PAGE), page: String(pageNum) });
		if (category) q.set('category', category);
		if (extract) q.set('extract', extract);
		if (brand) q.set('brand', brand);
		if (basis) q.set('basis', basis);
		if (verifiedOnly) q.set('verified', 'true');
		return q.toString();
	});

	const remote = new Remote<ProductListResponse>();
	$effect(() => remote.load(`/api/products?${query}`));

	const products = $derived(remote.data?.products ?? []);
	const total = $derived(remote.data?.total ?? 0);
	const pages = $derived(Math.ceil(total / PER_PAGE));

	function pill(active: boolean): string {
		return active
			? 'background: var(--gold); color: var(--bg); border-color: var(--gold);'
			: 'background: transparent; color: var(--cream2); border-color: rgba(58,122,47,0.4);';
	}

	const selectStyle = 'background: var(--bg2); border: 1px solid rgba(58,122,47,0.4);';
</script>

<!-- Filters -->
<div class="flex flex-col gap-3 mb-8">
	<div class="flex flex-wrap gap-2">
		<button
			onclick={() => setParam({ category: null })}
			class="text-[0.78rem] px-3 py-[0.3rem] rounded-[2px] border cursor-pointer"
			style={pill(category === '')}
		>
			All
		</button>
		{#each CATEGORIES as c (c)}
			<button
				onclick={() => setParam({ category: c })}
				class="text-[0.78rem] px-3 py-[0.3rem] rounded-[2px] border cursor-pointer capitalize"
				style={pill(category === c)}
			>
				{c}
			</button>
		{/each}
	</div>

	<div class="flex flex-wrap gap-3 items-center">
		<select
			value={extract}
			onchange={(e) => setParam({ extract: e.currentTarget.value })}
			class="text-[0.78rem] px-3 py-[0.35rem] rounded-[2px] text-cream2 cursor-pointer"
			style={selectStyle}
		>
			{#each EXTRACTS as ex (ex.key)}
				<option value={ex.key}>{ex.label}</option>
			{/each}
		</select>

		<select
			value={sort}
			onchange={(e) => setParam({ sort: e.currentTarget.value })}
			class="text-[0.78rem] px-3 py-[0.35rem] rounded-[2px] text-cream2 cursor-pointer"
			style={selectStyle}
		>
			{#each SORTS as s (s.key)}
				<option value={s.key}>{s.label}</option>
			{/each}
		</select>

		<select
			value={basis}
			onchange={(e) => setParam({ basis: e.currentTarget.value })}
			class="text-[0.78rem] px-3 py-[0.35rem] rounded-[2px] text-cream2 cursor-pointer"
			style={selectStyle}
			title="Compare price per mg of a single cannabinoid"
		>
			{#each BASES as b (b.key)}
				<option value={b.key}>{b.label}</option>
			{/each}
		</select>

		<label class="flex items-center gap-2 text-[0.78rem] text-cream2 cursor-pointer">
			<input
				type="checkbox"
				checked={verifiedOnly}
				onchange={(e) => setParam({ verified: e.currentTarget.checked ? 'true' : null })}
			/>
			Verified brands only
		</label>

		{#if brand}
			<button
				onclick={() => setParam({ brand: null })}
				class="text-[0.72rem] px-2 py-1 rounded-[2px] text-gold cursor-pointer"
				style="border: 1px solid var(--gold);"
			>
				brand: {brand} ✕
			</button>
		{/if}

		<span class="text-[0.75rem] text-cream2 ml-auto">
			{remote.loading ? 'Loading…' : `${total} product${total === 1 ? '' : 's'}`}
		</span>
	</div>
</div>

<!-- Grid / states -->
{#if remote.error}
	<p class="text-cream2 py-12 text-center">
		Could not load the catalog. The data service may be offline.{' '}
		<span class="opacity-60">({remote.error.message})</span>
	</p>
{:else if remote.loading && products.length === 0}
	<div class="grid gap-[1.1rem]" style="grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));">
		{#each Array(8) as _, i (i)}
			<div
				class="rounded-[5px] aspect-[3/4] animate-pulse"
				style="background: var(--bg2); border: 1px solid rgba(58,122,47,0.15);"
			></div>
		{/each}
	</div>
{:else if products.length === 0}
	<p class="text-cream2 py-12 text-center">
		No products match these filters.{' '}
		<button
			onclick={() => setParam({ category: null, extract: null, brand: null })}
			class="text-gold cursor-pointer"
		>
			Clear filters
		</button>
	</p>
{:else}
	<div class="grid gap-[1.1rem]" style="grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));">
		{#each products as p, i (p.id)}
			<ProductCard {p} index={i} />
		{/each}
	</div>
{/if}

<!-- Pagination -->
{#if pages > 1}
	<div class="flex items-center justify-center gap-4 mt-10 text-[0.85rem] text-cream2">
		<button
			disabled={pageNum <= 1}
			onclick={() => setParam({ page: String(pageNum - 1) })}
			class="px-3 py-1.5 rounded-[2px] cursor-pointer disabled:opacity-40"
			style="border: 1px solid rgba(58,122,47,0.3);"
		>
			← Prev
		</button>
		<span>
			Page {pageNum} of {pages}
		</span>
		<button
			disabled={pageNum >= pages}
			onclick={() => setParam({ page: String(pageNum + 1) })}
			class="px-3 py-1.5 rounded-[2px] cursor-pointer disabled:opacity-40"
			style="border: 1px solid rgba(58,122,47,0.3);"
		>
			Next →
		</button>
	</div>
{/if}
