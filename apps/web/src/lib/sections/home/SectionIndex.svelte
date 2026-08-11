<!-- The front-page directory: erowid's link index, laid out as four Wii
     channels. Rows are set like a record's tracklisting — label, dot leader,
     value — so the whole thing stays rigidly organised under the gloss.
     Sub-entries are derived from the real section content, so this index can
     never drift out of sync with the pages it points at. -->
<script lang="ts">
	import AquaIcon from './AquaIcon.svelte';
	import { eras } from '$sections/history/content';
	import { topics } from '$sections/science/content';
	import { lawBlocks } from '$sections/legality/ndps';
	import { archives } from '$sections/legality/extras';

	interface Entry {
		href: string;
		label: string;
		meta?: string;
	}
	interface Channel {
		href: string;
		word: string;
		icon: 'history' | 'science' | 'legality' | 'products';
		color: string;
		blurb: string;
		count: string;
		entries: Entry[];
	}

	// One accent per channel, carried by its icon disc, ring and bullet orbs.
	const channels: Channel[] = [
		{
			href: '/history',
			word: 'HISTORY',
			icon: 'history',
			color: 'var(--mood-purple-deep)',
			blurb: 'How it started here, and why it was banned anyway.',
			count: `${eras.length} eras`,
			entries: eras.map((e) => ({ href: `/history#${e.id}`, label: e.title, meta: e.years }))
		},
		{
			href: '/science',
			word: 'SCIENCE',
			icon: 'science',
			color: 'var(--mood-green)',
			blurb: 'The plant, the extracts, the doses — and how to read a label.',
			count: `${topics.length} topics`,
			entries: topics.map((t) => ({ href: `/science#${t.id}`, label: t.title }))
		},
		{
			href: '/legality',
			word: 'LEGALITY',
			icon: 'legality',
			color: 'var(--mood-rose)',
			blurb: 'What the NDPS Act says, and what your state does with it.',
			count: `${lawBlocks.length} statutes`,
			entries: [
				{ href: '/legality#ndps', label: 'The NDPS Act' },
				{ href: '/legality#bhang', label: 'The bhang exemption' },
				{ href: '/legality#hemp', label: 'Hemp & AYUSH' },
				{ href: '/legality#penalties', label: 'Penalties', meta: 'by quantity' },
				{ href: '/legality#states', label: 'State by state' },
				{ href: '/legality#archives', label: 'Archives', meta: `${archives.length} sources` }
			]
		},
		{
			href: '/products',
			word: 'PRODUCTS',
			icon: 'products',
			color: 'var(--mood-peach)',
			blurb: 'Every product on the Indian market, compared by ₹ per mg.',
			count: 'live',
			entries: [
				{ href: '/products', label: 'The catalog', meta: 'by ₹/mg' },
				{ href: '/brands', label: 'Brands' },
				{ href: '/compare', label: 'Compare side by side' },
				{ href: '/forum', label: 'Forum reports' },
				{ href: '/survey/results', label: 'Survey results' }
			]
		}
	];
</script>

<div class="index">
	{#each channels as c (c.href)}
		<section class="chan" style="--c: {c.color}">
			<header>
				<AquaIcon name={c.icon} color={c.color} />
				<a class="word" href={c.href}>{c.word}</a>
				<span class="count">{c.count}</span>
			</header>
			<div class="body">
				<p class="blurb">{c.blurb}</p>
				<ul>
					{#each c.entries as e (e.href + e.label)}
						<li>
							<a href={e.href}>
								<span class="label">{e.label}</span>
								<span class="lead" aria-hidden="true"></span>
								{#if e.meta}<span class="meta">{e.meta}</span>{/if}
							</a>
						</li>
					{/each}
				</ul>
			</div>
		</section>
	{/each}
</div>

<style>
	.index {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 1.1rem;
	}

	/* ── The channel card: white lacquer, ringed in its own colour ── */
	.chan {
		display: flex;
		flex-direction: column;
		border-radius: 13px;
		overflow: hidden;
		background: #fffdfd;
		box-shadow:
			0 0 0 2px color-mix(in srgb, var(--c) 52%, #fff),
			0 0 0 3.5px rgba(255, 255, 255, 0.9),
			var(--aqua-drop);
		transition:
			transform 0.2s cubic-bezier(0.22, 1, 0.36, 1),
			box-shadow 0.2s ease;
	}
	.chan:hover {
		transform: translateY(-3px);
		box-shadow:
			0 0 0 2px color-mix(in srgb, var(--c) 78%, #fff),
			0 0 0 3.5px rgba(255, 255, 255, 0.95),
			var(--aqua-drop-hover);
	}

	/* the glossy channel header strip */
	header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.55rem 0.7rem;
		background-image: var(--wii-face);
		border-bottom: 1px solid color-mix(in srgb, var(--c) 30%, #fff);
	}
	.word {
		font-family: var(--font-pixel);
		font-size: clamp(1.25rem, 1.75vw, 1.55rem);
		line-height: 1;
		letter-spacing: 0.02em;
		color: color-mix(in srgb, var(--c) 88%, #000);
		text-shadow: 0 1px 0 rgba(255, 255, 255, 0.95);
	}
	.word:hover,
	.word:focus-visible {
		text-decoration: underline;
		text-underline-offset: 0.16em;
	}
	/* a small recessed capsule, Aqua's status pill */
	.count {
		margin-left: auto;
		font-family: var(--font-pixel);
		font-size: 0.68rem;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		white-space: nowrap;
		color: var(--mute);
		padding: 0.08rem 0.42rem;
		border-radius: 999px;
		background: rgba(255, 255, 255, 0.75);
		box-shadow: var(--aqua-well);
	}

	.body {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		padding: 0.6rem 0.7rem 0.75rem;
		/* the faintest silver wash pooling at the foot of the card */
		background: linear-gradient(to bottom, #fffdfd 60%, #fbf6f7 100%);
	}
	.blurb {
		font-family: var(--font-ubuntu);
		font-size: 0.86rem;
		line-height: 1.45;
		color: var(--mute);
	}

	ul {
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: 0.12rem;
	}
	li {
		display: flex;
		align-items: baseline;
		gap: 0.42rem;
	}
	/* The shiny bullet: a lit sphere in the channel's colour. */
	li::before {
		content: '';
		flex-shrink: 0;
		width: 0.58rem;
		height: 0.58rem;
		border-radius: 50%;
		background-image:
			radial-gradient(
				circle at 34% 26%,
				rgba(255, 255, 255, 0.98) 0 12%,
				rgba(255, 255, 255, 0.5) 24%,
				rgba(255, 255, 255, 0) 52%
			),
			linear-gradient(
				to bottom,
				color-mix(in srgb, var(--c) 74%, #fff) 0%,
				var(--c) 55%,
				color-mix(in srgb, var(--c) 70%, #000) 100%
			);
		box-shadow:
			0 1px 1px rgba(120, 60, 140, 0.3),
			inset 0 -1px 1px rgba(0, 0, 0, 0.2);
		transform: translateY(-0.05rem);
	}
	li a {
		flex: 1;
		min-width: 0;
		display: flex;
		align-items: baseline;
		gap: 0.3rem;
		padding: 0.14rem 0;
		font-family: var(--font-ubuntu);
		font-size: 0.9rem;
		color: var(--ink);
	}
	/* the tracklisting dot leader */
	.lead {
		flex: 1 1 0.8rem;
		min-width: 0.5rem;
		align-self: center;
		border-bottom: 1px dotted color-mix(in srgb, var(--ink) 34%, transparent);
		transform: translateY(0.12em);
	}
	.label {
		flex-shrink: 0;
		max-width: 100%;
	}
	li a:hover .label,
	li a:focus-visible .label {
		color: color-mix(in srgb, var(--c) 82%, #000);
		text-decoration: underline;
		text-underline-offset: 0.18em;
	}
	.meta {
		flex-shrink: 0;
		font-family: var(--font-pixel);
		font-size: 0.7rem;
		letter-spacing: 0.03em;
		color: var(--mute);
		white-space: nowrap;
	}

	@media (max-width: 900px) {
		.index {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
	@media (max-width: 560px) {
		.index {
			grid-template-columns: 1fr;
		}
	}
</style>
