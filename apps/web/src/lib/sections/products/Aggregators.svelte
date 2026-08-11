<script lang="ts">
	// Port of ../dr-toke/apps/web/components/brands/Aggregators.tsx.
	// Live marketplace aggregators (GET /api/aggregators). Dead or disabled
	// marketplaces drop off automatically — the API withholds broken ones.
	import { Remote } from '$lib/api/remote.svelte';
	import type { AggregatorListResponse } from '$lib/api/reference';

	let { cardBg = 'bg2' }: { cardBg?: 'bg2' | 'bg3' } = $props();

	const remote = new Remote<AggregatorListResponse>();
	$effect(() => remote.load('/api/aggregators'));

	const aggregators = $derived(remote.data?.aggregators ?? []);
</script>

{#if remote.error}
	<!-- degrade gracefully — section just stays empty -->
{:else if remote.loading}
	<div class="grid gap-4" style="grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));">
		{#each Array(3) as _, i (i)}
			<div
				class="rounded-[4px] h-[110px] animate-pulse"
				style={`background: var(--${cardBg}); border: 1px solid rgba(58,122,47,0.2);`}
			></div>
		{/each}
	</div>
{:else if aggregators.length > 0}
	<div class="grid gap-4" style="grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));">
		{#each aggregators as a, i (a.slug)}
			{@const meta = [
				a.brand_count ? `${a.brand_count} brands` : null,
				a.product_count ? `${a.product_count} products` : null
			]
				.filter(Boolean)
				.join(' · ')}
			<a
				href={a.url}
				target="_blank"
				rel="noopener noreferrer"
				class={`card-hover reveal reveal-delay-${Math.min(i + 1, 5)} rounded-[4px] p-5 border border-[rgba(58,122,47,0.28)] block`}
				style={`background: var(--${cardBg});`}
			>
				<h3 class="font-display font-bold text-cream text-[1.1rem] mb-1">{a.name} ↗</h3>
				{#if meta}
					<p class="text-[0.68rem] tracking-[0.08em] uppercase text-gold mb-1">{meta}</p>
				{/if}
				<p class="text-[0.82rem] text-cream2 leading-relaxed">{a.description}</p>
			</a>
		{/each}
	</div>
{/if}
