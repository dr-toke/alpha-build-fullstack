<script lang="ts">
	// Exact copy of ../dr-toke/apps/web/app/product/page.tsx.
	import { page } from '$app/state';
	import CatalogShell from '$sections/products/CatalogShell.svelte';
	import BuyButton from '$sections/products/BuyButton.svelte';
	import CommentThread from '$sections/community/CommentThread.svelte';
	import { Remote } from '$lib/api/remote.svelte';
	import {
		type ApiProduct,
		categoryStyle,
		formatINR,
		hasSmokable,
		pmgColor,
		prettyExtract
	} from '$lib/api/catalog';

	const id = $derived(page.url.searchParams.get('id'));

	const remote = new Remote<{ product: ApiProduct }>();
	$effect(() => remote.load(id ? `/api/products/${encodeURIComponent(id)}` : null));
</script>

<svelte:head>
	<title>Product — Dr Toke</title>
</svelte:head>

<CatalogShell>
	<main class="max-w-[1100px] mx-auto px-8 py-16">
		{#if !id}
			<p class="text-cream2 py-20 text-center">No product specified.</p>
		{:else if remote.loading}
			<div class="py-20">
				<div class="h-8 w-1/2 rounded animate-pulse mb-4" style="background: var(--bg2);"></div>
				<div class="h-64 rounded animate-pulse" style="background: var(--bg2);"></div>
			</div>
		{:else if remote.error || !remote.data}
			<p class="text-cream2 py-20 text-center">
				Product not found.
				<a href="/products" class="text-gold hover:underline">Back to catalog</a>
			</p>
		{:else}
			{@const p = remote.data.product}
			{@const pmg = p.best_price_per_mg}
			{@const pmgC = pmgColor(pmg)}
			{@const cat = categoryStyle(p.category)}
			{@const secondary = p.categories.filter((c) => c !== p.category)}

			<a href="/products" class="text-[0.78rem] text-gold hover:underline">← Back to catalog</a>

			<div class="grid gap-8 mt-4" style="grid-template-columns: minmax(0,1fr) minmax(0,1.3fr);">
				<!-- Image -->
				<div
					class="rounded-[6px] aspect-square flex items-center justify-center overflow-hidden"
					style="background: var(--bg3); border: 1px solid rgba(58,122,47,0.25);"
				>
					{#if p.image_url}
						<img src={p.image_url} alt={p.name} class="w-full h-full object-cover" />
					{:else}
						<span class="text-cream2 opacity-50">No image</span>
					{/if}
				</div>

				<!-- Info -->
				<div>
					<div class="flex items-center gap-2 mb-2 flex-wrap">
						<span
							class="text-[0.62rem] px-2 py-[3px] rounded-[2px] font-semibold uppercase tracking-[0.05em]"
							style={`background: ${cat.bg}; border: 1px solid ${cat.border}; color: ${cat.text};`}
						>
							{p.category}
						</span>
						{#each secondary as c (c)}
							<span
								class="text-[0.58rem] px-[6px] py-[2px] rounded-[2px] text-cream2"
								style="border: 1px solid rgba(176,160,128,0.3);"
							>
								{c}
							</span>
						{/each}
						{#if prettyExtract(p.extract_type)}
							<span class="text-[0.65rem] text-cream2">{prettyExtract(p.extract_type)}</span>
						{/if}
					</div>

					<h1 class="font-display font-bold text-cream text-[1.9rem] leading-tight mb-1">{p.name}</h1>
					<p class="text-cream2 text-[0.9rem] mb-4">
						by
						<a href={`/products?brand=${encodeURIComponent(p.brand.slug)}`} class="text-gold hover:underline">
							{p.brand.name}
						</a>
						{#if p.brand.verified}
							<span class="ml-2 text-[0.65rem] text-green2">✓ Verified by Dr Toke</span>
						{:else}
							<span class="ml-2 text-[0.65rem]" style="color: var(--gold3);">Pending verification</span>
						{/if}
					</p>

					{#if !p.brand.verified}
						<p
							class="text-[0.78rem] leading-snug rounded-[3px] px-3 py-2 mb-4"
							style="background: rgba(200,168,75,0.06); border: 1px solid rgba(200,168,75,0.35); color: var(--gold2);"
						>
							Compliance not yet confirmed by the Dr Toke editorial team. We show it for
							transparency — make your own informed choice.
						</p>
					{/if}

					{#if hasSmokable(p)}
						<p
							class="text-[0.8rem] leading-snug rounded-[3px] px-3 py-2 mb-4"
							style="background: rgba(200,168,75,0.06); border: 1px solid rgba(200,168,75,0.4); color: var(--gold2);"
						>
							⚠ Smokable hemp products occupy a legal grey zone in India. Ayush-licensed
							extracts are protected; raw flower is not. Know your state laws before purchasing.
						</p>
					{/if}

					{#if p.prescription_required}
						<p
							class="text-[0.82rem] leading-snug rounded-[3px] px-3 py-2 mb-4"
							style="background: rgba(224,122,90,0.1); border: 1px solid var(--red2); color: var(--red2);"
						>
							<strong>Rx Required</strong> — available via licensed Ayush practitioners only.
						</p>
					{/if}

					<!-- Price block -->
					<div
						class="rounded-[5px] p-4 mb-4 flex items-end justify-between gap-2"
						style="background: var(--bg2); border: 1px solid rgba(58,122,47,0.25);"
					>
						<div class="min-w-0 flex-1">
							<div class="text-cream font-bold text-[1.6rem]">{formatINR(p.best_price_inr)}</div>
							<div class="text-[0.7rem] text-cream2">
								{[p.volume_ml ? `${p.volume_ml}ml` : '', p.weight_g ? `${p.weight_g}g` : ''].filter(Boolean).join(' · ')}
								{p.best_listing ? ` · cheapest at ${p.best_listing.source}` : ''}
							</div>
						</div>
						{#if pmg != null && pmgC}
							<div class="text-right shrink-0">
								<div class="font-bold text-[1.2rem]" style={`color: ${pmgC};`}>
									₹{pmg.toFixed(2)}/mg
								</div>
								<div class="text-[0.6rem] text-cream2">{p.price_per_mg_basis}</div>
							</div>
						{:else}
							<span class="text-[0.7rem] text-cream2 opacity-60">₹/mg not applicable</span>
						{/if}
					</div>

					<!-- Cannabinoid detail -->
					{#if p.cannabinoids.cbd_mg || p.cannabinoids.thc_mg}
						<div class="flex gap-4 mb-4 text-[0.8rem] text-cream2">
							{#if p.cannabinoids.cbd_mg}<span>CBD: {p.cannabinoids.cbd_mg}mg</span>{/if}
							{#if p.cannabinoids.thc_mg}<span>THC: {p.cannabinoids.thc_mg}mg</span>{/if}
							{#if p.coa_available}<span class="text-green2">COA available</span>{/if}
						</div>
					{/if}

					<!-- Buy + listings -->
					{#if p.best_listing}
						<div class="flex items-center gap-3 mb-3">
							<BuyButton
								listingId={p.best_listing.listing_id}
								sourceUrl={p.best_listing.source_url}
								affiliateUrl={p.best_listing.affiliate_url}
								label={`Buy at ${p.best_listing.source}`}
							/>
							<span class="text-[0.7rem] text-cream2">Opens in a new tab. We may earn a commission.</span>
						</div>
					{:else}
						<p class="text-cream2 text-[0.85rem] mb-3">Currently out of stock across tracked sources.</p>
					{/if}

					{#if p.other_listings.length > 0}
						<div class="mt-2">
							<p class="text-[0.7rem] uppercase tracking-[0.1em] text-cream2 mb-2">Also available at</p>
							<ul class="flex flex-col gap-2">
								{#each p.other_listings as l, i (i)}
									<li
										class="flex items-center justify-between text-[0.82rem] rounded-[3px] px-3 py-2"
										style="background: var(--bg2); border: 1px solid rgba(58,122,47,0.2);"
									>
										<span class="text-cream2">
											{l.source} · {formatINR(l.price_inr)} {l.in_stock ? '' : '(out of stock)'}
										</span>
										{#if l.in_stock}
											<BuyButton listingId={l.listing_id} sourceUrl={l.source_url} affiliateUrl={l.affiliate_url} label="Buy" />
										{/if}
									</li>
								{/each}
							</ul>
						</div>
					{/if}
				</div>
			</div>

			<CommentThread targetType="product" targetId={p.id} />
		{/if}
	</main>
</CatalogShell>
