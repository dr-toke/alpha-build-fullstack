<script lang="ts">
	import type { ApiState } from '$lib/api/reference';

	let { state }: { state: ApiState } = $props();

	const STATUS_COLORS: Record<string, string> = {
		legal: '#3a7a2f',
		tolerated: '#a07828',
		grey: '#a07828',
		limited: '#a07828',
		illegal: '#b03a2e'
	};

	const color = $derived(STATUS_COLORS[state.status] ?? 'var(--mute)');
</script>

<article class="state">
	<header>
		<h3>{state.name}</h3>
		<span class="status" style="border-color: {color}; color: {color};">{state.status}</span>
	</header>

	{#if state.featured}
		<p class="region">★ your region</p>
	{/if}

	<p class="shops">bhang shops: {state.bhang_shops}</p>
	<p class="detail">{state.detail}</p>

	{#if state.notes}
		<p class="notes">{state.notes}</p>
	{/if}

	<footer>
		{#if state.excise_url}
			<a href={state.excise_url} target="_blank" rel="noopener noreferrer">state excise ↗</a>
		{/if}
		<span class="verified">
			verified {state.last_verified}{#if state.stale}
				· <em>re-check due</em>{/if}
		</span>
	</footer>
</article>

<style>
	.state {
		border: 1px solid var(--ink);
		padding: 0.95rem 1rem 0.85rem;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}
	header {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 0.6rem;
	}
	h3 {
		font-family: var(--font-pixel);
		font-size: 1.35rem;
		line-height: 1.1;
	}
	.status {
		font-family: var(--font-pixel);
		font-size: 0.78rem;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		border: 1px solid;
		padding: 0.08rem 0.4rem;
		white-space: nowrap;
	}
	.region {
		font-family: var(--font-pixel);
		font-size: 0.8rem;
		color: var(--leaf);
	}
	.shops {
		font-family: var(--font-pixel);
		font-size: 0.85rem;
		color: var(--mute);
	}
	.detail {
		font-size: 0.9rem;
		line-height: 1.6;
	}
	.notes {
		font-size: 0.8rem;
		color: var(--mute);
		line-height: 1.5;
	}
	footer {
		margin-top: auto;
		padding-top: 0.5rem;
		display: flex;
		justify-content: space-between;
		gap: 0.6rem;
		flex-wrap: wrap;
		font-family: var(--font-pixel);
		font-size: 0.78rem;
	}
	footer a {
		text-decoration: underline;
	}
	footer a:hover {
		color: var(--leaf);
	}
	.verified {
		color: var(--mute);
	}
	.verified em {
		color: #b03a2e;
		font-style: normal;
	}
</style>
