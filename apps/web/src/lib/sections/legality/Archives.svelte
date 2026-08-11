<script lang="ts">
	import { archives, type ArchiveKind } from './extras';

	const KINDS: { key: ArchiveKind; label: string }[] = [
		{ key: 'statute', label: 'statutes' },
		{ key: 'report', label: 'reports' },
		{ key: 'judgment', label: 'judgments' },
		{ key: 'international', label: 'international' },
		{ key: 'news', label: 'press' }
	];

	let search = $state('');
	let kind = $state<'' | ArchiveKind>('');

	const filtered = $derived.by(() => {
		const q = search.trim().toLowerCase();
		return archives.filter((a) => {
			if (kind && a.kind !== kind) return false;
			if (q && !(a.title + ' ' + a.note + ' ' + a.source + ' ' + a.year).toLowerCase().includes(q))
				return false;
			return true;
		});
	});
</script>

<section class="archives" id="archives" aria-label="The archives">
	<h2>The archives</h2>
	<p class="lede">
		Everything the law says, said, and got told — statutes, commissions, judgments, treaties, and
		the clippings file. Sourced where it exists, marked where it's still being collected.
	</p>

	<div class="controls">
		<input type="search" placeholder="search the archive…" bind:value={search} />
		<div class="pills" role="group" aria-label="Filter by kind">
			<button class:on={kind === ''} onclick={() => (kind = '')}>all</button>
			{#each KINDS as k (k.key)}
				<button class:on={kind === k.key} onclick={() => (kind = k.key)}>{k.label}</button>
			{/each}
		</div>
		<span class="count">{filtered.length} of {archives.length} entries</span>
	</div>

	<ol class="stack">
		{#each filtered as a (a.title)}
			<li class="entry">
				<div class="meta">
					<span class="year">{a.year}</span>
					<span class="kind">{a.kind}</span>
				</div>
				<div class="body">
					<h3>
						{#if a.url}
							<a href={a.url} target="_blank" rel="noopener noreferrer">{a.title} ↗</a>
						{:else}
							{a.title} <span class="unsourced">· to source</span>
						{/if}
					</h3>
					<p class="note">{a.note}</p>
					<p class="source">{a.source}</p>
				</div>
			</li>
		{:else}
			<li class="empty">
				nothing matches.
				<button
					class="reset"
					onclick={() => {
						search = '';
						kind = '';
					}}>reset</button
				>
			</li>
		{/each}
	</ol>
</section>

<style>
	.archives {
		padding: 2.1rem 0;
		border-top: 1px solid var(--line);
		scroll-margin-top: 3.2rem;
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
		max-width: 44rem;
		margin-bottom: 1.2rem;
	}
	.controls {
		display: flex;
		align-items: center;
		gap: 0.9rem;
		flex-wrap: wrap;
		margin-bottom: 1.1rem;
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
	.count {
		font-family: var(--font-pixel);
		font-size: 0.85rem;
		color: var(--mute);
		margin-left: auto;
	}
	.stack {
		list-style: none;
		display: flex;
		flex-direction: column;
	}
	.entry {
		display: flex;
		gap: 1.1rem;
		padding: 0.85rem 0;
		border-bottom: 1px dashed var(--line);
	}
	.meta {
		min-width: 6.2rem;
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}
	.year {
		font-family: var(--font-pixel);
		font-size: 1rem;
	}
	.kind {
		font-family: var(--font-pixel);
		font-size: 0.72rem;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: var(--mute);
	}
	.body h3 {
		font-family: var(--font-pixel);
		font-size: 1.15rem;
		line-height: 1.2;
	}
	.body h3 a:hover {
		background: var(--ink);
		color: var(--paper);
	}
	.unsourced {
		font-size: 0.75rem;
		color: var(--mute);
		letter-spacing: 0.06em;
	}
	.note {
		font-size: 0.9rem;
		line-height: 1.6;
		max-width: 46rem;
		margin-top: 0.15rem;
	}
	.source {
		font-family: var(--font-pixel);
		font-size: 0.78rem;
		color: var(--mute);
		margin-top: 0.2rem;
	}
	.empty {
		font-family: var(--font-pixel);
		padding: 1.2rem 0;
		color: var(--mute);
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
