<script lang="ts">
	import { Remote } from '$lib/api/remote.svelte';
	import type { StateListResponse } from '$lib/api/reference';
	import StateCard from './StateCard.svelte';

	const remote = new Remote<StateListResponse>();
	$effect(() => remote.load('/api/states'));

	const states = $derived(remote.data?.states ?? []);

	// ── Search + status filter (migrated from apps/web's states page idea:
	//    filterable, shareable legal grid — here fully client-side). ──
	const STATUSES = ['legal', 'tolerated', 'grey', 'limited', 'illegal'] as const;

	let search = $state('');
	let status = $state(''); // '' = all

	const filtered = $derived.by(() => {
		const q = search.trim().toLowerCase();
		return states.filter((s) => {
			if (status && s.status !== status) return false;
			if (q && !(s.name + ' ' + s.detail + ' ' + (s.notes ?? '')).toLowerCase().includes(q))
				return false;
			return true;
		});
	});
</script>

<section class="grid-wrap" id="states" aria-label="State-by-state legal status">
	<h2>State by state</h2>
	<p class="lede">
		Live from the Dr Toke catalogue — every entry carries the date it was last verified, because
		laws change.
	</p>

	<div class="controls">
		<input type="search" placeholder="search a state… (name, detail)" bind:value={search} />
		<div class="pills" role="group" aria-label="Filter by status">
			<button class:on={status === ''} onclick={() => (status = '')}>all</button>
			{#each STATUSES as st (st)}
				<button class:on={status === st} class={`st-${st}`} onclick={() => (status = st)}>
					{st}
				</button>
			{/each}
		</div>
		<span class="count">
			{remote.loading ? 'loading…' : `${filtered.length} of ${states.length} states`}
		</span>
	</div>

	{#if remote.error}
		<p class="offline">
			The state data lives in the backend (<code>/api/states</code>) and it isn't reachable right
			now ({remote.error.message}). Start it with <code>../dr-toke/scripts/demo-up.sh</code>.
		</p>
	{:else if remote.loading}
		<div class="grid">
			{#each Array(6) as _, i (i)}
				<div class="ghost"></div>
			{/each}
		</div>
	{:else if filtered.length === 0}
		<p class="offline">
			No states match.
			<button
				class="reset"
				onclick={() => {
					search = '';
					status = '';
				}}>reset</button
			>
		</p>
	{:else}
		<div class="grid">
			{#each filtered as state (state.slug)}
				<StateCard {state} />
			{/each}
		</div>
	{/if}
</section>

<style>
	.grid-wrap {
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
		margin-bottom: 1.2rem;
	}
	.controls {
		display: flex;
		align-items: center;
		gap: 0.9rem;
		flex-wrap: wrap;
		margin-bottom: 1.2rem;
	}
	.controls input {
		font-family: var(--font-pixel);
		font-size: 0.95rem;
		padding: 0.4rem 0.6rem;
		border: 1px solid var(--ink);
		background: var(--paper);
		color: var(--ink);
		width: min(20rem, 100%);
	}
	.controls input:focus {
		outline: 2px solid var(--ink);
		outline-offset: 1px;
	}
	.pills {
		display: flex;
		gap: 0.4rem;
		flex-wrap: wrap;
	}
	.pills button {
		font-family: var(--font-pixel);
		font-size: 0.85rem;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		padding: 0.25rem 0.6rem;
		border: 1px solid var(--mute);
		background: transparent;
		color: var(--mute);
		cursor: pointer;
	}
	.pills button.on {
		background: var(--ink);
		border-color: var(--ink);
		color: var(--paper);
	}
	.pills button.st-legal:not(.on) { border-color: #3a7a2f; color: #3a7a2f; }
	.pills button.st-tolerated:not(.on),
	.pills button.st-grey:not(.on),
	.pills button.st-limited:not(.on) { border-color: #a07828; color: #a07828; }
	.pills button.st-illegal:not(.on) { border-color: #b03a2e; color: #b03a2e; }
	.count {
		font-family: var(--font-pixel);
		font-size: 0.85rem;
		color: var(--mute);
		margin-left: auto;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
		gap: 1rem;
	}
	.ghost {
		border: 1px dashed var(--line);
		min-height: 170px;
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
	.reset {
		font-family: inherit;
		font-size: inherit;
		border: none;
		background: none;
		color: var(--ink);
		text-decoration: underline;
		cursor: pointer;
	}
</style>
