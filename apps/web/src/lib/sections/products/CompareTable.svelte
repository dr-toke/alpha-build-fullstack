<script lang="ts">
	// Exact port of ../dr-toke/apps/web/components/compare/CompareTable.tsx —
	// one dense sortable table, ₹-per-mg-of-cannabinoid as "the only number
	// that matters", everything filtered/sorted client-side in place.
	import { Remote } from '$lib/api/remote.svelte';
	import BuyButton from './BuyButton.svelte';
	import { pmgColor, formatINR } from '$lib/api/catalog';

	interface CompareItem {
		id: string;
		name: string;
		brand: string;
		brand_slug: string;
		verified: boolean;
		category: string;
		extract_type: string;
		cbd_mg: number | null;
		thc_mg: number | null;
		total_mg: number | null;
		basis: string;
		volume_ml: number | null;
		weight_g: number | null;
		price_inr: number;
		cbd_pmg: number | null;
		thc_pmg: number | null;
		best_pmg: number;
		rx: boolean;
		listing_id: string | null;
		source: string | null;
		source_url: string | null;
		affiliate_url: string | null;
	}

	interface CompareResponse {
		items: CompareItem[];
		total: number;
		last_updated: string;
	}

	type Basis = 'best' | 'cbd' | 'thc';
	type SortKey = 'pmg' | 'price' | 'mg' | 'brand' | 'name' | 'category';

	const CATEGORIES = ['tincture', 'edible', 'topical', 'smokable', 'vapeable', 'extract', 'beverage'];
	const EXTRACTS = ['full_spectrum', 'broad_spectrum', 'isolate', 'vijaya'];

	function pmgFor(it: CompareItem, b: Basis): number | null {
		if (b === 'cbd') return it.cbd_pmg;
		if (b === 'thc') return it.thc_pmg;
		return it.best_pmg;
	}

	function mgFor(it: CompareItem, b: Basis): number | null {
		if (b === 'cbd') return it.cbd_mg;
		if (b === 'thc') return it.thc_mg;
		return it.total_mg;
	}

	const remote = new Remote<CompareResponse>();
	$effect(() => remote.load('/api/compare'));

	let basis = $state<Basis>('best');
	let search = $state('');
	let category = $state('');
	let extract = $state('');
	let brand = $state('');
	let verifiedOnly = $state(false);
	let noRx = $state(false);
	let maxBudget = $state('');
	let sortKey = $state<SortKey>('pmg');
	let sortAsc = $state(true);

	const items = $derived(remote.data?.items ?? []);

	const brands = $derived(Array.from(new Set(items.map((i) => i.brand))).sort());

	const filtered = $derived.by(() => {
		const q = search.trim().toLowerCase();
		const budget = parseFloat(maxBudget) || Infinity;
		let out = items.filter((it) => {
			if (pmgFor(it, basis) == null) return false;
			if (q && !(it.name + ' ' + it.brand).toLowerCase().includes(q)) return false;
			if (category && it.category !== category) return false;
			if (extract && it.extract_type !== extract) return false;
			if (brand && it.brand !== brand) return false;
			if (verifiedOnly && !it.verified) return false;
			if (noRx && it.rx) return false;
			if (it.price_inr > budget) return false;
			return true;
		});
		const dir = sortAsc ? 1 : -1;
		out = [...out].sort((a, b) => {
			switch (sortKey) {
				case 'pmg':
					return ((pmgFor(a, basis) ?? Infinity) - (pmgFor(b, basis) ?? Infinity)) * dir;
				case 'price':
					return (a.price_inr - b.price_inr) * dir;
				case 'mg':
					return ((mgFor(a, basis) ?? 0) - (mgFor(b, basis) ?? 0)) * dir;
				case 'brand':
					return a.brand.localeCompare(b.brand) * dir;
				case 'name':
					return a.name.localeCompare(b.name) * dir;
				case 'category':
					return a.category.localeCompare(b.category) * dir;
			}
		});
		return out;
	});

	function clickSort(k: SortKey) {
		if (k === sortKey) sortAsc = !sortAsc;
		else {
			sortKey = k;
			sortAsc = true;
		}
	}

	function reset() {
		search = '';
		category = '';
		extract = '';
		brand = '';
		verifiedOnly = false;
		noRx = false;
		maxBudget = '';
		basis = 'best';
		sortKey = 'pmg';
		sortAsc = true;
	}

	const inputStyle = 'background: var(--bg2); border: 1px solid rgba(58,122,47,0.4); color: var(--cream);';

	const arrow = (k: SortKey) => (sortKey === k ? (sortAsc ? ' ↑' : ' ↓') : '');
</script>

{#snippet th(label: string, k: SortKey, title: string = 'Click to sort')}
	<th class="px-3 py-2 font-semibold cursor-pointer select-none hover:text-gold2" onclick={() => clickSort(k)} {title}>
		{label}
	</th>
{/snippet}

{#if remote.error}
	<p class="text-cream2 py-12 text-center">
		Could not load comparison data. <span class="opacity-60">({remote.error.message})</span>
	</p>
{:else}
	<div>
		<!-- ── Filter bar ── -->
		<div class="flex flex-wrap gap-2 items-center mb-3">
			<input
				bind:value={search}
				placeholder="Search product or brand…"
				class="text-[0.8rem] px-3 py-[0.4rem] rounded-[3px] w-[220px]"
				style={inputStyle}
			/>
			<select
				value={basis}
				onchange={(e) => (basis = e.currentTarget.value as Basis)}
				class="text-[0.8rem] px-2 py-[0.4rem] rounded-[3px] cursor-pointer"
				style={inputStyle}
				title="Which cannabinoid to normalise price against"
			>
				<option value="best">₹/mg — dominant</option>
				<option value="cbd">₹/mg of CBD</option>
				<option value="thc">₹/mg of THC</option>
			</select>
			<select
				bind:value={category}
				class="text-[0.8rem] px-2 py-[0.4rem] rounded-[3px] cursor-pointer capitalize"
				style={inputStyle}
			>
				<option value="">All categories</option>
				{#each CATEGORIES as c (c)}
					<option value={c}>{c}</option>
				{/each}
			</select>
			<select
				bind:value={extract}
				class="text-[0.8rem] px-2 py-[0.4rem] rounded-[3px] cursor-pointer"
				style={inputStyle}
			>
				<option value="">All extracts</option>
				{#each EXTRACTS as x (x)}
					<option value={x}>{x.replace('_', ' ')}</option>
				{/each}
			</select>
			<select
				bind:value={brand}
				class="text-[0.8rem] px-2 py-[0.4rem] rounded-[3px] cursor-pointer max-w-[170px]"
				style={inputStyle}
			>
				<option value="">All brands</option>
				{#each brands as b (b)}
					<option value={b}>{b}</option>
				{/each}
			</select>
			<input
				value={maxBudget}
				oninput={(e) => (maxBudget = e.currentTarget.value.replace(/[^\d]/g, ''))}
				placeholder="Max ₹"
				inputmode="numeric"
				class="text-[0.8rem] px-3 py-[0.4rem] rounded-[3px] w-[90px]"
				style={inputStyle}
			/>
			<label class="flex items-center gap-1 text-[0.78rem] text-cream2 cursor-pointer">
				<input type="checkbox" bind:checked={verifiedOnly} />
				Verified
			</label>
			<label class="flex items-center gap-1 text-[0.78rem] text-cream2 cursor-pointer">
				<input type="checkbox" bind:checked={noRx} />
				No Rx
			</label>
			<button
				onclick={reset}
				class="text-[0.74rem] px-2 py-[0.35rem] rounded-[3px] text-gold cursor-pointer"
				style="border: 1px solid var(--gold);"
			>
				Reset
			</button>
			<span class="text-[0.76rem] text-cream2 ml-auto">
				{remote.loading ? 'Loading…' : `${filtered.length} of ${items.length} products`}
				{#if remote.data?.last_updated}
					<span class="opacity-50"> · updated {remote.data.last_updated.slice(0, 10)}</span>
				{/if}
			</span>
		</div>

		<!-- ── Table ── -->
		<div class="overflow-x-auto rounded-[4px]" style="border: 1px solid rgba(58,122,47,0.3);">
			<table class="w-full text-[0.82rem]" style="border-collapse: collapse;">
				<thead>
					<tr
						class="text-left text-[0.66rem] uppercase tracking-[0.08em] text-gold sticky top-0"
						style="background: var(--bg3);"
					>
						{@render th(`₹/mg${arrow('pmg')}`, 'pmg', 'Price per mg of cannabinoid — the only number that matters')}
						{@render th(`Price${arrow('price')}`, 'price')}
						{@render th(`Content${arrow('mg')}`, 'mg')}
						<th class="px-3 py-2 font-semibold">Size</th>
						{@render th(`Brand${arrow('brand')}`, 'brand')}
						{@render th(`Product${arrow('name')}`, 'name')}
						{@render th(`Type${arrow('category')}`, 'category')}
						<th class="px-3 py-2 font-semibold">Buy</th>
					</tr>
				</thead>
				<tbody>
					{#if remote.loading && items.length === 0}
						<tr><td colspan="8" class="px-3 py-10 text-center text-cream2">Loading the catalogue…</td></tr>
					{:else if filtered.length === 0}
						<tr>
							<td colspan="8" class="px-3 py-10 text-center text-cream2">
								No products match. <button onclick={reset} class="text-gold cursor-pointer">Reset filters</button>
							</td>
						</tr>
					{:else}
						{#each filtered as it, i (it.id)}
							{@const pmg = pmgFor(it, basis) ?? Infinity}
							{@const mg = mgFor(it, basis)}
							{@const topPick = sortKey === 'pmg' && sortAsc && i < 3}
							<tr
								class="border-t"
								style={`border-color: rgba(58,122,47,0.18); background: ${topPick ? 'rgba(90,170,74,0.07)' : i % 2 ? 'rgba(255,255,255,0.012)' : 'transparent'};`}
							>
								<td
									class="px-3 py-[0.55rem] whitespace-nowrap font-bold"
									style={`color: ${pmgColor(pmg) ?? 'var(--cream2)'};`}
								>
									₹{pmg.toFixed(2)}
									{#if topPick}<span class="ml-1 text-[0.6rem]" title="Best value right now">★</span>{/if}
									{#if basis === 'best'}
										<span class="ml-1 text-[0.6rem] font-normal text-cream2 uppercase">{it.basis}</span>
									{/if}
								</td>
								<td class="px-3 py-[0.55rem] whitespace-nowrap text-cream">{formatINR(it.price_inr)}</td>
								<td class="px-3 py-[0.55rem] whitespace-nowrap text-cream2">
									{mg ? `${Math.round(mg)}mg` : '—'}
									{#if it.cbd_mg && it.thc_mg}
										<span class="text-[0.66rem] opacity-70"> ({Math.round(it.cbd_mg)}C/{Math.round(it.thc_mg)}T)</span>
									{/if}
								</td>
								<td class="px-3 py-[0.55rem] whitespace-nowrap text-cream2">
									{it.volume_ml ? `${it.volume_ml}ml` : it.weight_g ? `${it.weight_g}g` : '—'}
								</td>
								<td class="px-3 py-[0.55rem] whitespace-nowrap">
									<span class="text-cream">{it.brand}</span>
									{#if it.verified}<span class="ml-1 text-green2 text-[0.7rem]" title="Verified by Dr Toke">✓</span>{/if}
								</td>
								<td class="px-3 py-[0.55rem] max-w-[340px]">
									<a href={`/product?id=${encodeURIComponent(it.id)}`} class="text-cream hover:text-gold line-clamp-1">
										{it.name}
									</a>
									{#if it.rx}<span class="text-[0.6rem]" style="color: var(--red2);"> Rx</span>{/if}
								</td>
								<td class="px-3 py-[0.55rem] whitespace-nowrap text-cream2 capitalize">{it.category}</td>
								<td class="px-3 py-[0.55rem] whitespace-nowrap">
									{#if it.source_url}
										<BuyButton
											compact
											listingId={it.listing_id ?? undefined}
											sourceUrl={it.source_url}
											affiliateUrl={it.affiliate_url}
											label={it.source ?? 'store'}
											pagePath="/compare"
										/>
									{:else}
										<span class="text-cream2 opacity-50 text-[0.72rem]">—</span>
									{/if}
								</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>
		</div>

		<p class="text-[0.68rem] text-cream2 opacity-60 mt-3">
			₹/mg = listed price ÷ stated cannabinoid content. Always verify the live price and COA with the seller
			before purchase. Products marked Rx require a valid prescription.
		</p>
	</div>
{/if}
