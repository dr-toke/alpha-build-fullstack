<script lang="ts">
	import Wordmark from '$sections/home/Wordmark.svelte';
	import Vision from '$sections/home/Vision.svelte';
	import PhotoBand from '$sections/home/PhotoBand.svelte';
	import SectionIndex from '$sections/home/SectionIndex.svelte';

	// erowid's link stack: everything that isn't a reading section.
	const utility: { href: string; label: string; cyber?: boolean }[] = [
		{ href: '/forum', label: 'forum' },
		{ href: '/survey', label: 'survey' },
		{ href: '/account', label: 'account' },
		{ href: '/parcha', label: 'parcha', cyber: true }
	];

	// Three doors for a first-time reader, in the order the questions get asked.
	const startHere: { href: string; label: string; note: string }[] = [
		{ href: '/legality#ndps', label: 'Is it legal?', note: 'what the NDPS Act says' },
		{ href: '/science#roa', label: 'How much, how long?', note: 'dose, onset, duration' },
		{ href: '/products', label: 'What should it cost?', note: '₹ per mg, across brands' }
	];
</script>

<svelte:head>
	<title>Dr Toke — know what you're doing</title>
	<meta
		name="description"
		content="An open source of knowledge on cannabis in India — the history, the science, the legality, and every product compared by ₹/mg."
	/>
</svelte:head>

<main class="home">
	<!-- film grain over the whole page, so the gloss reads as photographed
	     rather than rendered -->
	<div class="grain" aria-hidden="true"></div>

	<header class="masthead">
		<Wordmark />
		<p class="tagline">Documenting cannabis in India — the plant, the law, and the market</p>
		<nav class="utility" aria-label="Community and account">
			{#each utility as u (u.href)}
				<a href={u.href} class:cyber={u.cyber}>{u.label}</a>
			{/each}
		</nav>
	</header>

	<PhotoBand />

	<SectionIndex />

	<section class="colophon">
		<div class="about">
			<h2>About this site</h2>
			<Vision />
		</div>
		<div class="start">
			<h2>Start here</h2>
			<ul>
				{#each startHere as s (s.href)}
					<li>
						<a href={s.href}>
							<span class="q">{s.label}</span>
							<span class="lead" aria-hidden="true"></span>
							<span class="note">{s.note}</span>
						</a>
					</li>
				{/each}
			</ul>
		</div>
	</section>
</main>

<style>
	.home {
		position: relative;
		max-width: 1180px;
		margin: 0 auto;
		padding: 0.6rem 2rem 3rem;
		display: flex;
		flex-direction: column;
		gap: 1.4rem;
	}
	.grain {
		position: fixed;
		inset: 0;
		z-index: 40;
		background-image: var(--grain);
		opacity: 0.05;
		mix-blend-mode: multiply;
		pointer-events: none;
	}

	/* ── Masthead: a centred altar — logo in its sunburst, then the tagline,
	   then the pills ── */
	.masthead {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.7rem;
	}
	.tagline {
		font-family: var(--font-pixel);
		font-size: clamp(0.95rem, 1.7vw, 1.2rem);
		letter-spacing: 0.03em;
		line-height: 1.35;
		text-align: center;
		text-wrap: balance;
		color: var(--mood-green);
		text-shadow: 0 1px 0 rgba(255, 255, 255, 0.95);
		max-width: 36rem;
	}

	/* ── Utility: shiny pills ── */
	.utility {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 0.4rem;
	}
	.utility a {
		font-family: var(--font-pixel);
		font-size: 0.84rem;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: #fff;
		text-shadow: 0 1px 1px rgba(0, 0, 0, 0.35);
		padding: 0.16rem 0.9rem;
		border-radius: 999px;
		background-color: var(--mood-rose);
		background-image: var(--aqua-face-accent);
		box-shadow:
			0 0 0 1.5px color-mix(in srgb, var(--mood-rose) 62%, #fff),
			0 0 0 3px rgba(255, 255, 255, 0.9),
			var(--aqua-drop),
			inset 0 1px 0 rgba(255, 255, 255, 0.5);
		transition:
			box-shadow 0.15s ease,
			transform 0.1s ease;
	}
	.utility a:hover,
	.utility a:focus-visible {
		box-shadow:
			0 0 0 1.5px color-mix(in srgb, var(--mood-rose) 85%, #fff),
			0 0 0 3px rgba(255, 255, 255, 0.95),
			var(--aqua-drop-hover),
			inset 0 1px 0 rgba(255, 255, 255, 0.6);
	}
	/* pressed: the face sinks into its own shadow */
	.utility a:active {
		transform: translateY(1px);
		box-shadow: var(--aqua-well);
	}
	/* parcha keeps its iridescent drift — now under the glass. */
	.utility a.cyber {
		text-transform: lowercase;
		background-image: var(--aqua-face-accent),
			linear-gradient(100deg, #9a8cff, #6fc3ff, #7fe0b2, #ffd98f, #ff9ad5, #9a8cff);
		background-size:
			100% 100%,
			300% 100%;
		box-shadow:
			0 0 0 1.5px rgba(154, 140, 255, 0.55),
			0 0 0 3px rgba(255, 255, 255, 0.9),
			var(--aqua-drop),
			inset 0 1px 0 rgba(255, 255, 255, 0.5);
		animation: iridesce 9s linear infinite;
	}
	@keyframes iridesce {
		to {
			background-position:
				0 0,
				300% 0;
		}
	}

	/* ── Colophon: a recessed well under the raised channels ── */
	.colophon {
		display: grid;
		grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr);
		gap: 1.2rem 2.4rem;
		padding: 1rem 1.2rem;
		border-radius: 10px;
		background-color: rgba(255, 255, 255, 0.5);
		background-image: var(--aqua-pinstripe);
		box-shadow: var(--aqua-well);
	}
	.colophon h2 {
		font-family: var(--font-pixel);
		font-size: 0.8rem;
		font-weight: 400;
		text-transform: uppercase;
		letter-spacing: 0.1em;
		color: var(--mood-green);
		text-shadow: 0 1px 0 rgba(255, 255, 255, 0.95);
		margin-bottom: 0.45rem;
	}
	.start ul {
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: 0.28rem;
	}
	.start li {
		display: flex;
		align-items: baseline;
		gap: 0.42rem;
	}
	/* the same shiny sphere the channels use, in green */
	.start li::before {
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
				color-mix(in srgb, var(--mood-green) 74%, #fff) 0%,
				var(--mood-green) 55%,
				color-mix(in srgb, var(--mood-green) 70%, #000) 100%
			);
		box-shadow:
			0 1px 1px rgba(120, 60, 140, 0.3),
			inset 0 -1px 1px rgba(0, 0, 0, 0.2);
		transform: translateY(-0.05rem);
	}
	.start a {
		flex: 1;
		min-width: 0;
		display: flex;
		align-items: baseline;
		gap: 0.3rem;
		font-family: var(--font-ubuntu);
		font-size: 0.92rem;
		color: var(--ink);
	}
	.start .lead {
		flex: 1 1 0.8rem;
		min-width: 0.5rem;
		align-self: center;
		border-bottom: 1px dotted color-mix(in srgb, var(--ink) 34%, transparent);
		transform: translateY(0.12em);
	}
	.start .q {
		flex-shrink: 0;
	}
	.start a:hover .q,
	.start a:focus-visible .q {
		color: var(--mood-green);
		text-decoration: underline;
		text-underline-offset: 0.18em;
	}
	.start .note {
		flex-shrink: 0;
		font-family: var(--font-pixel);
		font-size: 0.72rem;
		color: var(--mute);
		white-space: nowrap;
	}

	@media (max-width: 700px) {
		.home {
			padding: 0.4rem 1.1rem 2.4rem;
		}
		.colophon {
			grid-template-columns: 1fr;
		}
	}
</style>
