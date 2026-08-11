<script lang="ts">
	// Migrated from ../dr-toke/apps/web/app/survey/results/page.tsx.
	// Aggregate results — running totals from the API, never individual responses.
	import CatalogShell from '$sections/products/CatalogShell.svelte';
	import { Remote } from '$lib/api/remote.svelte';
	import { SURVEY_STEPS } from '$sections/community/survey';

	interface OptionCount {
		option: string;
		count: number;
	}

	interface SurveyResults {
		results: Record<string, OptionCount[]>;
		total_responses: number;
	}

	const remote = new Remote<SurveyResults>();
	$effect(() => remote.load('/api/survey/results'));

	const hasData = $derived(remote.data != null && remote.data.total_responses > 0);

	function stepTotal(counts: OptionCount[]): number {
		return counts.reduce((a, b) => a + b.count, 0);
	}

	function barColor(pct: number): string {
		return pct > 40 ? 'var(--gold)' : pct > 20 ? 'var(--green2)' : 'var(--green3)';
	}
</script>

<svelte:head>
	<title>Survey Results — Dr Toke</title>
	<meta
		name="description"
		content="Aggregate responses from the Dr Toke product survey. Individual responses are never stored."
	/>
</svelte:head>

<CatalogShell>
	<main class="max-w-[900px] mx-auto px-8 py-20">
		<p class="text-[0.68rem] tracking-[0.22em] uppercase text-gold font-semibold font-display mb-1">
			Community Data
		</p>
		<h1
			class="font-display font-bold text-cream leading-[1.15] mb-[0.45rem]"
			style="font-size: clamp(1.9rem, 3.8vw, 2.9rem);"
		>
			Survey Results
		</h1>
		<p class="text-cream2 text-base max-w-[580px] leading-[1.7] mb-12">
			Aggregate responses from the Dr Toke product survey. These directly shape our formulation.
			Individual responses are never stored.
		</p>

		{#if remote.loading}
			<p class="text-cream2 py-16 text-center opacity-60">Loading results…</p>
		{:else if remote.error}
			<p class="text-cream2 py-16 text-center opacity-60">
				Could not load results. The data service may be offline.
			</p>
		{:else if !hasData}
			<div class="text-cream2 text-base py-16 text-center opacity-60">
				<p class="text-4xl mb-4">📊</p>
				<p>No survey responses recorded yet.</p>
				<a href="/survey" class="block mt-4 text-gold hover:underline text-sm">Take the survey →</a>
			</div>
		{:else if remote.data}
			<p class="text-cream2 text-[0.85rem] mb-10">
				{remote.data.total_responses} total response{remote.data.total_responses === 1 ? '' : 's'}
			</p>
			<div class="flex flex-col gap-12">
				{#each SURVEY_STEPS as step (step.key)}
					{@const counts = remote.data.results[step.key] ?? []}
					{@const total = stepTotal(counts)}
					<section>
						<h2 class="font-display font-bold text-cream text-[1.2rem] mb-4">{step.question}</h2>
						<div class="flex flex-col gap-3">
							{#each counts as c (c.option)}
								{@const pct = total > 0 ? Math.round((c.count / total) * 100) : 0}
								<div>
									<div class="flex justify-between items-center mb-1">
										<span class="text-[0.86rem] text-cream2">{c.option}</span>
										<span class="text-[0.78rem] text-gold font-semibold">
											{pct}%{c.count > 0 ? ` (${c.count})` : ''}
										</span>
									</div>
									<div class="h-[6px] rounded-full bg-[rgba(58,122,47,0.2)]">
										<div
											class="h-full rounded-full transition-all duration-700"
											style={`width: ${pct}%; background: ${barColor(pct)};`}
										></div>
									</div>
								</div>
							{/each}
						</div>
					</section>
				{/each}
			</div>
		{/if}
	</main>
</CatalogShell>
