<script lang="ts">
	import PageHead from '$lib/components/ui/PageHead.svelte';
	import LegalSubnav from '$sections/legality/LegalSubnav.svelte';
	import StateGrid from '$sections/legality/StateGrid.svelte';
	import Archives from '$sections/legality/Archives.svelte';
	import { lawBlocks } from '$sections/legality/ndps';
	import { timeline, penalties, faqs, glossary } from '$sections/legality/extras';
</script>

<svelte:head>
	<title>Legality — Dr Toke</title>
	<meta
		name="description"
		content="Cannabis law in India, in plain language: the NDPS Act, the bhang carve-out, penalties, a legal timeline, FAQ, glossary, and the live state-by-state status."
	/>
</svelte:head>

<main class="wrap">
	<PageHead
		word="LEGALITY"
		lede="The NDPS Act in plain language, the bhang carve-out, penalties, and the live verified status of every state."
	/>

	<LegalSubnav />

	<!-- ── Overview ── -->
	<section class="block" id="overview">
		<h2>The one-minute version</h2>
		<p class="body">
			One plant, three legal universes. The <strong>flower and resin</strong> are banned nationally
			since 1985. The <strong>leaf (bhang)</strong> belongs to each state's excise department — legal
			and licensed in some states, banned in others. And <strong>licensed hemp & Ayush products</strong>
			are an open, growing legal market. Everything on this page exists to keep you on the right side
			of those three lines.
		</p>
		<div class="chips">
			<span class="chip c-red">flower & resin — banned (NDPS)</span>
			<span class="chip c-amber">bhang — state by state</span>
			<span class="chip c-green">licensed hemp/ayush — legal</span>
		</div>
	</section>

	{#each lawBlocks as block (block.id)}
		<section class="block" id={block.id}>
			<h2>{block.title}</h2>
			<p class="body">{block.body}</p>
			{#if block.todo}
				<p class="todo">✎ to write: {block.todo}</p>
			{/if}
		</section>
	{/each}

	<!-- ── Timeline ── -->
	<section class="block" id="timeline">
		<h2>The timeline</h2>
		<ol class="timeline">
			{#each timeline as t (t.year + t.text)}
				<li>
					<span class="year">{t.year}</span>
					<span class="event">{t.text}</span>
				</li>
			{/each}
		</ol>
	</section>

	<!-- ── Penalties ── -->
	<section class="block" id="penalties">
		<h2>Penalties, plainly</h2>
		<p class="body">
			The NDPS Act punishes by quantity, not intent. These are the commonly cited ganja thresholds —
			shown for awareness, and due for counsel review before launch.
		</p>
		<div class="table-scroll">
			<table>
				<thead>
					<tr><th>offence</th><th>quantity</th><th>punishment</th></tr>
				</thead>
				<tbody>
					{#each penalties as row (row.offence)}
						<tr>
							<td>{row.offence}</td>
							<td>{row.quantity}</td>
							<td>{row.punishment}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<p class="todo">✎ verify: thresholds & sections with counsel; add bail/procedure notes</p>
	</section>

	<StateGrid />

	<Archives />

	<!-- ── FAQ ── -->
	<section class="block" id="faq">
		<h2>Questions people actually ask</h2>
		<div class="faqs">
			{#each faqs as f (f.q)}
				<details>
					<summary>{f.q}</summary>
					<p>{f.a}</p>
				</details>
			{/each}
		</div>
	</section>

	<!-- ── Glossary ── -->
	<section class="block" id="glossary">
		<h2>Glossary</h2>
		<dl class="glossary">
			{#each glossary as g (g.term)}
				<div class="entry">
					<dt>{g.term}</dt>
					<dd>{g.def}</dd>
				</div>
			{/each}
		</dl>
	</section>

	<p class="disclaimer-note">
		None of this is legal advice. It is legal <em>awareness</em> — the starting point, not the
		lawyer.
	</p>
</main>

<style>
	.wrap {
		max-width: 1080px;
		margin: 0 auto;
		padding: 0 2rem 4rem;
	}
	.block {
		padding: 2.1rem 0;
		border-top: 1px solid var(--line);
		scroll-margin-top: 3.2rem;
	}
	.block h2 {
		font-family: var(--font-pixel);
		font-size: clamp(1.7rem, 3.4vw, 2.4rem);
		line-height: 1.08;
		margin-bottom: 0.7rem;
	}
	.block .body {
		max-width: 46rem;
		font-size: 1rem;
		line-height: 1.75;
	}
	.todo {
		display: inline-block;
		margin-top: 1rem;
		font-family: var(--font-pixel);
		font-size: 0.85rem;
		color: var(--mute);
		border: 1px dashed var(--mute);
		padding: 0.2rem 0.55rem;
	}

	.chips {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
		margin-top: 1rem;
	}
	.chip {
		font-family: var(--font-pixel);
		font-size: 0.85rem;
		padding: 0.25rem 0.6rem;
		border: 1px solid;
	}
	.c-red { color: #b03a2e; border-color: #b03a2e; }
	.c-amber { color: #a07828; border-color: #a07828; }
	.c-green { color: #3a7a2f; border-color: #3a7a2f; }

	.timeline {
		list-style: none;
		max-width: 46rem;
	}
	.timeline li {
		display: flex;
		gap: 1rem;
		padding: 0.55rem 0;
		border-bottom: 1px dashed var(--line);
	}
	.year {
		font-family: var(--font-pixel);
		font-size: 0.95rem;
		min-width: 6.5rem;
		color: var(--ink);
	}
	.event {
		font-size: 0.93rem;
		line-height: 1.6;
	}

	.table-scroll {
		overflow-x: auto;
		margin-top: 1.1rem;
		border: 1px solid var(--ink);
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.9rem;
	}
	th {
		font-family: var(--font-pixel);
		font-size: 0.85rem;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		text-align: left;
		padding: 0.55rem 0.8rem;
		background: var(--ink);
		color: var(--paper);
		white-space: nowrap;
	}
	td {
		padding: 0.55rem 0.8rem;
		border-top: 1px solid var(--line);
		vertical-align: top;
	}
	td:first-child {
		font-weight: 700;
		white-space: nowrap;
	}

	.faqs {
		max-width: 46rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	details {
		border: 1px solid var(--ink);
		padding: 0.65rem 0.85rem;
	}
	summary {
		font-family: var(--font-pixel);
		font-size: 1rem;
		cursor: pointer;
	}
	details[open] summary {
		margin-bottom: 0.45rem;
	}
	details p {
		font-size: 0.92rem;
		line-height: 1.65;
	}

	.glossary {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
		gap: 0.7rem;
		max-width: 60rem;
	}
	.entry {
		border: 1px dashed var(--mute);
		padding: 0.6rem 0.75rem;
	}
	dt {
		font-family: var(--font-pixel);
		font-size: 1rem;
	}
	dd {
		font-size: 0.85rem;
		color: var(--mute);
		line-height: 1.55;
	}

	.disclaimer-note {
		margin-top: 2.2rem;
		font-family: var(--font-pixel);
		font-size: 0.95rem;
		color: var(--mute);
	}
</style>
