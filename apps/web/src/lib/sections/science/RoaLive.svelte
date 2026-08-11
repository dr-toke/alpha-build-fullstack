<script lang="ts">
	// Migrated from apps/web/components/education/RoaGuide.tsx — same live data
	// (GET /api/roa), re-skinned into the site's pixel theme.
	import { Remote } from '$lib/api/remote.svelte';
	import type { ROAListResponse } from '$lib/api/reference';

	const remote = new Remote<ROAListResponse>();
	$effect(() => remote.load('/api/roa'));

	const methods = $derived(remote.data?.methods ?? []);
</script>

<section class="roa-live" aria-label="Routes of administration — live guide">
	<h2>The live guide</h2>
	<p class="lede">
		Onset, duration, and bioavailability per method — served from the backend catalogue and kept
		verified.
	</p>

	{#if remote.error}
		<p class="offline">
			The ROA guide lives in the backend (<code>/api/roa</code>) and it isn't reachable right now
			({remote.error.message}). The written primer above still applies.
		</p>
	{:else if remote.loading}
		<div class="list">
			{#each Array(4) as _, i (i)}
				<div class="ghost"></div>
			{/each}
		</div>
	{:else if methods.length > 0}
		<div class="list">
			{#each methods as m (m.slug)}
				<article class="method" id={m.slug}>
					<header>
						<h3>{m.method}</h3>
						<div class="stats">
							<span>onset: {m.onset}</span>
							<span>duration: {m.duration}</span>
							<span>bioavail: {m.bioavailability}</span>
						</div>
					</header>
					<div class="cols">
						<div>
							<h4 class="pro">pros</h4>
							<ul>
								{#each m.pros as p (p)}<li>{p}</li>{/each}
							</ul>
						</div>
						<div>
							<h4 class="con">cons</h4>
							<ul>
								{#each m.cons as c (c)}<li>{c}</li>{/each}
							</ul>
						</div>
						<div>
							<h4 class="best">best for</h4>
							<ul>
								{#each m.best_for as b (b)}<li>{b}</li>{/each}
							</ul>
						</div>
					</div>
					{#if m.warning_note}
						<p class="warning">⚠ {m.warning_note}</p>
					{/if}
				</article>
			{/each}
		</div>
	{/if}
</section>

<style>
	.roa-live {
		padding: 2.1rem 0;
		border-top: 1px solid var(--line);
	}
	h2 {
		font-family: var(--font-pixel);
		font-size: clamp(1.9rem, 4vw, 2.8rem);
		line-height: 1.05;
		margin-bottom: 0.4rem;
	}
	.lede {
		font-size: 0.95rem;
		color: var(--mute);
		max-width: 42rem;
		margin-bottom: 1.4rem;
	}
	.list {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}
	.method {
		border: 1px solid var(--ink);
		padding: 1rem 1.1rem;
	}
	.method header {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 0.8rem;
		flex-wrap: wrap;
		margin-bottom: 0.8rem;
	}
	.method h3 {
		font-family: var(--font-pixel);
		font-size: 1.5rem;
	}
	.stats {
		display: flex;
		gap: 0.4rem;
		flex-wrap: wrap;
	}
	.stats span {
		font-family: var(--font-pixel);
		font-size: 0.8rem;
		border: 1px solid var(--mute);
		color: var(--mute);
		padding: 0.12rem 0.45rem;
	}
	.cols {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
		gap: 1rem;
	}
	.cols h4 {
		font-family: var(--font-pixel);
		font-size: 0.85rem;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		margin-bottom: 0.3rem;
	}
	.cols .pro { color: #3a7a2f; }
	.cols .con { color: #b03a2e; }
	.cols .best { color: #a07828; }
	.cols ul {
		list-style: none;
		font-size: 0.88rem;
		line-height: 1.6;
		color: var(--ink);
	}
	.cols li::before {
		content: '· ';
		color: var(--mute);
	}
	.warning {
		margin-top: 0.9rem;
		font-size: 0.88rem;
		line-height: 1.6;
		border-left: 3px solid #a07828;
		padding: 0.4rem 0.7rem;
		background: rgba(160, 120, 40, 0.06);
	}
	.ghost {
		border: 1px dashed var(--line);
		min-height: 150px;
		animation: ghostPulse 1.4s ease-in-out infinite;
	}
	@keyframes ghostPulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.45; }
	}
	.offline {
		font-family: var(--font-pixel);
		font-size: 0.95rem;
		border: 1px dashed var(--mute);
		padding: 1rem 1.1rem;
		max-width: 46rem;
		color: var(--mute);
	}
	.offline code {
		font-family: inherit;
		color: var(--ink);
	}
</style>
