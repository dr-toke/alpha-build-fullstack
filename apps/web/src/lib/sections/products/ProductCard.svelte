<script lang="ts">
	// Exact port of ../dr-toke/apps/web/components/products/ProductCard.tsx.
	import {
		type ApiProduct,
		categoryStyle,
		formatINR,
		hasSmokable,
		pmgColor,
		prettyExtract
	} from '$lib/api/catalog';

	let { p, index }: { p: ApiProduct; index: number } = $props();

	const secondary = $derived(p.categories.filter((c) => c !== p.category).slice(0, 2));
	const pmg = $derived(p.best_price_per_mg);
	const pmgC = $derived(pmgColor(pmg));
	const cbdPmg = $derived(p.cbd_price_per_mg);
	const thcPmg = $derived(p.thc_price_per_mg);
	const size = $derived(
		[
			p.cannabinoids.concentration_type !== 'hemp_seed' && p.cannabinoids.cbd_mg
				? `${p.cannabinoids.cbd_mg}mg`
				: '',
			p.volume_ml ? `${p.volume_ml}ml` : p.weight_g ? `${p.weight_g}g` : ''
		]
			.filter(Boolean)
			.join(' · ')
	);
</script>

{#snippet badge(cat: string, primary: boolean = false)}
	{@const s = categoryStyle(cat)}
	<span
		class={`rounded-[2px] tracking-[0.05em] ${primary ? 'text-[0.62rem] px-2 py-[3px] font-semibold uppercase' : 'text-[0.58rem] px-[6px] py-[2px]'}`}
		style={`background: ${primary ? s.bg : 'transparent'}; border: 1px solid ${primary ? s.border : 'rgba(176,160,128,0.3)'}; color: ${primary ? s.text : 'var(--cream2)'};`}
	>
		{cat}
	</span>
{/snippet}

<a
	href={`/product?id=${encodeURIComponent(p.id)}`}
	class={`card-hover reveal reveal-delay-${Math.min((index % 5) + 1, 5)} rounded-[5px] overflow-hidden flex flex-col`}
	style="background: var(--bg2); border: 1px solid rgba(58,122,47,0.25);"
>
	<!-- Image -->
	<div class="relative aspect-[4/3] flex items-center justify-center" style="background: var(--bg3);">
		{#if p.image_url}
			<img src={p.image_url} alt={p.name} loading="lazy" class="w-full h-full object-cover" />
		{:else}
			<span class="text-cream2 text-[0.7rem] opacity-50">No image</span>
		{/if}
		{#if !p.in_stock}
			<span
				class="absolute top-2 right-2 text-[0.6rem] px-2 py-[2px] rounded-[2px]"
				style="background: rgba(8,15,8,0.85); color: var(--red2); border: 1px solid var(--red2);"
			>
				Out of stock
			</span>
		{/if}
	</div>

	<div class="p-4 flex flex-col gap-2 flex-1">
		<div class="flex items-start justify-between gap-2">
			<span class="text-[0.7rem] text-cream2">{p.brand.name}</span>
			{#if p.brand.verified}
				<span class="text-[0.58rem] text-green2" title="Brand verified by Dr Toke">✓ Verified</span>
			{:else}
				<span class="text-[0.58rem]" style="color: var(--gold3);" title="Compliance not yet confirmed">Pending</span>
			{/if}
		</div>

		<h3 class="font-display font-bold text-cream text-[1.05rem] leading-tight line-clamp-2">
			{p.name}
		</h3>

		<div class="flex flex-wrap gap-1 items-center">
			{@render badge(p.category, true)}
			{#each secondary as c (c)}
				{@render badge(c)}
			{/each}
			{#if prettyExtract(p.extract_type)}
				<span class="text-[0.58rem] text-cream2">{prettyExtract(p.extract_type)}</span>
			{/if}
		</div>

		{#if p.prescription_required}
			<span
				class="self-start text-[0.6rem] px-2 py-[2px] rounded-[2px]"
				style="background: rgba(224,122,90,0.12); border: 1px solid var(--red2); color: var(--red2);"
			>
				Rx Required
			</span>
		{/if}

		{#if hasSmokable(p)}
			<p
				class="text-[0.62rem] leading-snug rounded-[3px] px-2 py-1"
				style="background: rgba(200,168,75,0.06); border: 1px solid rgba(200,168,75,0.4); color: var(--gold2);"
			>
				⚠ Smokable — legal grey zone. Know your state laws.
			</p>
		{/if}

		<div class="mt-auto pt-2 flex items-end justify-between gap-2">
			<div class="min-w-0 flex-1">
				<div class="text-cream font-semibold text-[1.05rem]">{formatINR(p.best_price_inr)}</div>
				{#if size}
					<div class="text-[0.65rem] text-cream2">{size}</div>
				{/if}
			</div>
			{#if cbdPmg != null && thcPmg != null}
				<!-- Full-spectrum / vijaya: show both bases, never one ambiguous number. -->
				<div class="text-right leading-tight shrink-0">
					<div class="font-bold text-[0.9rem]" style={`color: ${pmgColor(cbdPmg) ?? 'var(--cream2)'};`}>
						₹{cbdPmg.toFixed(2)}/mg <span class="text-[0.58rem] font-normal text-cream2">CBD</span>
					</div>
					<div class="font-bold text-[0.9rem]" style={`color: ${pmgColor(thcPmg) ?? 'var(--cream2)'};`}>
						₹{thcPmg.toFixed(2)}/mg <span class="text-[0.58rem] font-normal text-cream2">THC</span>
					</div>
				</div>
			{:else if pmg != null && pmgC}
				<div class="text-right shrink-0">
					<div class="font-bold text-[0.95rem]" style={`color: ${pmgC};`}>
						₹{pmg.toFixed(2)}/mg
					</div>
					<div class="text-[0.58rem] text-cream2">{p.price_per_mg_basis}</div>
				</div>
			{:else}
				<span class="text-[0.6rem] text-cream2 opacity-60" title="₹/mg not applicable for this product type">
					₹/mg n/a
				</span>
			{/if}
		</div>
	</div>
</a>
