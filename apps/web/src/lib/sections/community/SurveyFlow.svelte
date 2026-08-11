<!-- Exact copy of ../dr-toke/apps/web/components/survey/SurveyFlow.tsx.
     PHILOSOPHY.md §4.6: no email ask, no identity, no tracking; thank-you
     screen reflects choices back; progress bar fills; steps slide from right.
     (The React original wired hover styles via JS — here it's the .opt CSS.) -->
<script lang="ts">
	import { apiFetch } from '$lib/api/client';
	import { SURVEY_STEPS, type SurveyKey } from './survey';

	type SurveyAnswers = Partial<Record<SurveyKey, string>>;
	type SurveyIndices = Partial<Record<SurveyKey, number>>;

	let step = $state(0);
	let answers = $state<SurveyAnswers>({});
	let indices = $state<SurveyIndices>({});
	let done = $state(false);
	let stepKey = $state(0); // forces re-mount for animation

	const totalSteps = SURVEY_STEPS.length;
	const currentStep = $derived(SURVEY_STEPS[step]);

	function answer(key: SurveyKey, value: string, index: number): void {
		answers = { ...answers, [key]: value };
		indices = { ...indices, [key]: index };

		if (step < totalSteps - 1) {
			stepKey += 1;
			step += 1;
		} else {
			// Submit aggregate counters to the API — no PII, no identity, no session.
			// Each submission is four counter increments server-side.
			void apiFetch('/api/survey/response', {
				method: 'POST',
				body: JSON.stringify({
					extractType: indices.extractType ?? 0,
					useCase: indices.useCase ?? 0,
					priceRange: indices.priceRange ?? 0,
					carrierOil: indices.carrierOil ?? 0
				})
			}).catch(() => {
				// Submission failed (offline/API down). The user still sees their
				// selections; we simply don't record this response.
			});
			done = true;
		}
	}

	function back(): void {
		if (step > 0) {
			stepKey += 1;
			step -= 1;
		}
	}

	function reset(): void {
		step = 0;
		answers = {};
		done = false;
		stepKey += 1;
	}
</script>

<div
	class="max-w-[660px] rounded-[6px] p-[clamp(1.5rem,4vw,2.5rem)] relative overflow-hidden border border-[rgba(200,168,75,0.28)]"
	style="background: linear-gradient(135deg, var(--bg2) 0%, var(--bg3) 100%);"
>
	<!-- Decorative circle -->
	<div
		class="absolute right-[-50px] top-[-50px] w-[180px] h-[180px] rounded-full pointer-events-none"
		style="background: rgba(200,168,75,0.04); border: 1px solid rgba(200,168,75,0.12);"
		aria-hidden="true"
	></div>

	{#if !done}
		{#key stepKey}
			<div class="step-in relative">
				<!-- Progress bar -->
				<div class="flex gap-[4px] mb-6" aria-label="Survey progress">
					{#each SURVEY_STEPS as _, i (i)}
						<div
							class="h-[3px] rounded-[2px] flex-1 transition-colors duration-300"
							style={`background: ${i <= step ? 'var(--gold)' : 'rgba(58,122,47,0.35)'};`}
						></div>
					{/each}
				</div>

				<!-- Step counter -->
				<p class="text-[0.68rem] text-cream2 tracking-[0.12em] uppercase mb-[0.6rem]">
					Question {step + 1} of {totalSteps}
				</p>

				<!-- Question -->
				<h3 class="font-display font-bold text-cream mb-6 leading-[1.3]" style="font-size: 1.4rem;">
					{currentStep?.question}
				</h3>

				<!-- Options -->
				<div class="flex flex-col gap-[0.6rem]">
					{#each currentStep?.options ?? [] as opt, i (i)}
						<button
							onclick={() => currentStep && answer(currentStep.key, opt, i)}
							class="opt text-left w-full px-5 py-[0.85rem] rounded-[4px] text-[0.9rem] text-cream cursor-pointer"
							style="font-family: var(--font-body), sans-serif;"
						>
							{opt}
						</button>
					{/each}
				</div>

				<!-- Back button -->
				{#if step > 0}
					<button
						onclick={back}
						class="mt-5 bg-transparent border-none text-cream2 text-[0.85rem] cursor-pointer"
						style="font-family: var(--font-body), sans-serif;"
					>
						← Back
					</button>
				{/if}
			</div>
		{/key}
	{:else}
		<!-- Thank you — PHILOSOPHY.md §4.6: reflect choices back, no email ask -->
		<div class="step-in text-center py-6 relative">
			<div class="text-[2.5rem] mb-3" aria-hidden="true">✓</div>
			<h3 class="font-display font-bold mb-3" style="font-size: 1.6rem; color: var(--green2);">
				Thank you.
			</h3>
			<p class="text-cream2 text-[0.9rem] leading-[1.65] max-w-[460px] mx-auto mb-4">
				<strong class="text-cream">{answers.extractType?.split(' (')[0]}</strong>
				· <strong class="text-cream">{answers.useCase}</strong>
				· <strong class="text-cream">{answers.priceRange}</strong>
				· <strong class="text-cream">{answers.carrierOil}</strong>
			</p>
			<p class="text-cream2 text-[0.8rem] mb-2">This shapes our formulation.</p>
			<p class="text-cream2 text-[0.75rem] opacity-70 mb-6">
				No email captured — your privacy is respected.
				<a href="/survey/results" class="text-gold hover:underline">See aggregate results →</a>
			</p>
			<button
				onclick={reset}
				class="border border-[rgba(58,122,47,0.5)] text-cream2 bg-transparent px-5 py-2 text-[0.85rem] cursor-pointer rounded-[2px] transition-colors duration-200 hover:border-gold hover:text-gold"
				style="font-family: var(--font-body), sans-serif;"
			>
				Take again
			</button>
		</div>
	{/if}
</div>

<style>
	.opt {
		background: var(--bg);
		border: 1px solid rgba(58, 122, 47, 0.5);
		transition:
			border-color 0.2s ease,
			background 0.2s ease,
			transform 0.2s ease;
	}
	.opt:hover {
		border-color: var(--gold);
		background: rgba(200, 168, 75, 0.07);
		transform: translateX(4px);
	}
</style>
