<script lang="ts">
	// Migrated from ../dr-toke/apps/web/app/survey/page.tsx (community tier,
	// so it keeps the exact dr-toke look).
	import CatalogShell from '$sections/products/CatalogShell.svelte';
	import SurveyFlow from '$sections/community/SurveyFlow.svelte';
</script>

<svelte:head>
	<title>Survey — Shape Our Oil — Dr Toke</title>
	<meta
		name="description"
		content="Help us build the right hemp extract for the Indian market. Tell us what you want — no email required."
	/>
</svelte:head>

<CatalogShell>
	<main class="max-w-[1200px] mx-auto px-8 py-20">
		<p class="reveal text-[0.68rem] tracking-[0.22em] uppercase text-gold font-semibold font-display mb-1">
			Shape What Comes Next
		</p>
		<h1
			class="reveal reveal-delay-1 font-display font-bold text-cream leading-[1.15] mb-[0.45rem]"
			style="font-size: clamp(1.9rem, 3.8vw, 2.9rem);"
		>
			Help Us Build the Right Product
		</h1>
		<p class="reveal reveal-delay-2 text-cream2 text-base max-w-[620px] leading-[1.7] mb-10">
			We're developing a COA-verified, Ayush-compliant hemp extract for the Indian market. Your
			answers directly shape the formulation. <strong class="text-cream">No email required.</strong>
		</p>

		<div class="reveal reveal-delay-3">
			<SurveyFlow />
		</div>

		<div class="reveal mt-10 max-w-[660px]">
			<p class="text-cream2 text-[0.85rem] leading-relaxed">
				Aggregate results are published publicly at
				<a href="/survey/results" class="text-gold hover:underline">/survey/results</a>. Individual
				responses are never stored — only running totals. No account, no email, no tracking.
			</p>
		</div>
	</main>
</CatalogShell>
