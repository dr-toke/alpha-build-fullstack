<script lang="ts">
	// Port of ../dr-toke/apps/web/components/brands/BrandCard.tsx.
	// Renders a brand from live API data (GET /api/brands).
	import Tag from './Tag.svelte';
	import type { ApiBrand } from '$lib/api/catalog';

	let { brand, index = 0 }: { brand: ApiBrand; index?: number } = $props();

	const COMPLIANCE_LINKS = {
		ayush: { url: 'https://ayush.gov.in', tooltip: 'Verify this registration at ayush.gov.in' },
		fssai: { url: 'https://foscos.fssai.gov.in', tooltip: 'Verify this licence at foscos.fssai.gov.in' }
	} as const;

	const delayClass = $derived(`reveal-delay-${Math.min((index % 4) + 1, 5)}`);
	const subtitle = $derived([brand.full_name, brand.city].filter(Boolean).join(' · '));
</script>

<article
	class={`card-hover reveal ${delayClass} flex flex-col gap-[0.55rem] rounded-[4px] p-6 bg-bg2 border border-[rgba(58,122,47,0.28)]`}
>
	<!-- Header row -->
	<div class="flex justify-between items-start gap-2">
		<div>
			<h3 class="font-display font-bold text-[1.25rem] text-cream leading-tight">
				{brand.name}
			</h3>
			{#if subtitle}<p class="text-[0.72rem] text-cream2">{subtitle}</p>{/if}
		</div>

		<!-- Compliance badges -->
		<div class="flex gap-1 flex-wrap justify-end flex-shrink-0">
			{#if brand.ayush}
				<a
					href={COMPLIANCE_LINKS.ayush.url}
					target="_blank"
					rel="noopener noreferrer"
					title={COMPLIANCE_LINKS.ayush.tooltip}
					class="cursor-help"
				>
					<Tag variant="green">Ayush ↗</Tag>
				</a>
			{/if}
			{#if brand.fssai}
				<a
					href={COMPLIANCE_LINKS.fssai.url}
					target="_blank"
					rel="noopener noreferrer"
					title={COMPLIANCE_LINKS.fssai.tooltip}
					class="cursor-help"
				>
					<Tag variant="gold">FSSAI ↗</Tag>
				</a>
			{/if}
			{#if brand.coa_available}
				<span title="Certificate of Analysis available on request">
					<Tag variant="muted" class="cursor-default">COA</Tag>
				</span>
			{/if}
		</div>
	</div>

	<!-- Description — consumer voice -->
	{#if brand.description}
		<p class="text-[0.86rem] text-cream2 leading-[1.62] flex-grow">{brand.description}</p>
	{/if}

	<!-- Categories (from live catalog) -->
	{#if brand.categories.length > 0 || brand.highlight}
		<div class="flex flex-wrap gap-[0.3rem] pt-1 items-center">
			{#each brand.categories as c (c)}
				<Tag variant="green">{c}</Tag>
			{/each}
			{#if brand.highlight}
				<span class="text-[0.68rem] text-gold font-semibold ml-auto self-end flex-shrink-0">
					✦ {brand.highlight}
				</span>
			{/if}
		</div>
	{/if}

	<!-- Actions -->
	<div class="flex gap-3 items-center pt-1 mt-auto">
		{#if brand.website_url}
			<a
				href={brand.website_url}
				target="_blank"
				rel="noopener noreferrer"
				class="text-[0.8rem] text-gold font-medium opacity-70 hover:opacity-100 transition-opacity duration-200"
			>
				Official site →
			</a>
		{/if}
		{#if brand.product_count > 0}
			<a
				href={`/products?brand=${encodeURIComponent(brand.slug)}`}
				class="text-[0.78rem] text-cream2 opacity-70 hover:opacity-100 transition-opacity duration-200 ml-auto"
			>
				{brand.product_count} product{brand.product_count === 1 ? '' : 's'} →
			</a>
		{/if}
	</div>

	<!-- Last verified -->
	{#if brand.last_verified}
		<p class="text-[0.65rem] text-cream2 opacity-40 mt-1">Data verified: {brand.last_verified}</p>
	{/if}
</article>
