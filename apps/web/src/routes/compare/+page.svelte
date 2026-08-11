<script lang="ts">
	// Exact copy of ../dr-toke/apps/web/app/compare/page.tsx.
	import CatalogShell from '$sections/products/CatalogShell.svelte';
	import CompareTable from '$sections/products/CompareTable.svelte';
</script>

<svelte:head>
	<title>Compare ₹/mg — Every Cannabinoid Product Ranked by Value</title>
	<meta
		name="description"
		content="India's cannabinoid price comparison: every verified product ranked by price per mg of CBD and THC. Sort, filter, and find the best value."
	/>
</svelte:head>

<CatalogShell>
	<main class="max-w-[1380px] mx-auto px-6 py-16">
		<p class="reveal text-[0.68rem] tracking-[0.22em] uppercase text-gold font-semibold font-display mb-1">
			The Value Index
		</p>
		<h1
			class="reveal reveal-delay-1 font-display font-bold text-cream leading-[1.15] mb-[0.45rem]"
			style="font-size: clamp(1.9rem, 3.8vw, 2.9rem);"
		>
			Compare ₹ per mg
		</h1>
		<p class="reveal reveal-delay-2 text-cream2 text-base max-w-[680px] leading-[1.7] mb-8">
			The sticker price means nothing — what matters is what you pay per milligram of actual
			cannabinoid. Every in-stock product, one table, normalised. Sort any column. Switch the basis
			to compare per mg of CBD or per mg of THC, like-for-like.
		</p>

		<CompareTable />
	</main>
</CatalogShell>
