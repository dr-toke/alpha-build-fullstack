<script lang="ts">
	// Exact copy of ../dr-toke/apps/web/app/products/page.tsx.
	// (Only change: the ROA guide link points at this site's /science#roa.)
	import CatalogShell from '$sections/products/CatalogShell.svelte';
	import CatalogGrid from '$sections/products/CatalogGrid.svelte';
</script>

<svelte:head>
	<title>Product Catalog — Dr Toke</title>
	<meta
		name="description"
		content="Every verified hemp and cannabinoid product in India, aggregated and compared by ₹/mg. Tinctures, edibles, topicals, extracts, and more — with honest compliance labelling."
	/>
</svelte:head>

<CatalogShell>
	<main class="max-w-[1200px] mx-auto px-8 py-20">
		<p class="reveal text-[0.68rem] tracking-[0.22em] uppercase text-gold font-semibold font-display mb-1">
			The Catalog
		</p>
		<h1
			class="reveal reveal-delay-1 font-display font-bold text-cream leading-[1.15] mb-[0.45rem]"
			style="font-size: clamp(1.9rem, 3.8vw, 2.9rem);"
		>
			Every Verified Product, Compared
		</h1>
		<p class="reveal reveal-delay-2 text-cream2 text-base max-w-[640px] leading-[1.7] mb-4">
			Aggregated from across India's licensed cannabinoid market and ranked by value (₹ per mg of
			active cannabinoid). We label compliance honestly — verified brands, pending brands, and
			prescription-only products are all marked clearly.
		</p>
		<p class="reveal reveal-delay-2 text-cream2 text-[0.82rem] max-w-[640px] mb-10 opacity-80">
			New to cannabinoids? Read the
			<a href="/science#roa" class="text-gold hover:underline">Routes of Administration guide</a>
			before buying — how you take it matters as much as what you take.
		</p>

		<CatalogGrid />
	</main>
</CatalogShell>
