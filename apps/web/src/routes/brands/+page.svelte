<script lang="ts">
	// Migrated from ../dr-toke/apps/web/app/brands/page.tsx (part of the
	// products world, so it keeps the exact dr-toke look).
	import CatalogShell from '$sections/products/CatalogShell.svelte';
	import BrandsGrid from '$sections/products/BrandsGrid.svelte';
	import Aggregators from '$sections/products/Aggregators.svelte';
</script>

<svelte:head>
	<title>Verified Brand Directory — Dr Toke</title>
	<meta
		name="description"
		content="Ayush-registered and FSSAI-compliant hemp and cannabis brands in India. Verified, updated, consumer-perspective reviews."
	/>
</svelte:head>

<CatalogShell>
	<main class="max-w-[1200px] mx-auto px-8 py-20">
		<p class="reveal text-[0.68rem] tracking-[0.22em] uppercase text-gold font-semibold font-display mb-1">
			Trusted Sources
		</p>
		<h1
			class="reveal reveal-delay-1 font-display font-bold text-cream leading-[1.15] mb-[0.45rem]"
			style="font-size: clamp(1.9rem, 3.8vw, 2.9rem);"
		>
			Verified Brand Directory
		</h1>
		<p class="reveal reveal-delay-2 text-cream2 text-base max-w-[620px] leading-[1.7] mb-10">
			Every listing carries verifiable Ayush registration and/or FSSAI compliance. No grey-market
			products. Descriptions written from consumer perspective, not PR copy.
		</p>

		<BrandsGrid />

		<!-- Aggregators -->
		<div class="mt-12 pt-8 border-t border-[rgba(58,122,47,0.2)]">
			<p class="reveal text-[0.68rem] tracking-[0.18em] uppercase text-gold font-semibold mb-4">
				Shop Across Platforms
			</p>
			<Aggregators cardBg="bg2" />
		</div>
	</main>
</CatalogShell>
