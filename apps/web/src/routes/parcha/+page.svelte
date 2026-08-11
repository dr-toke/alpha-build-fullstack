<script lang="ts">
	import ConceptCard from '$sections/parcha/ConceptCard.svelte';
	import { concepts } from '$sections/parcha/content';
</script>

<svelte:head>
	<title>Parcha — Dr Toke</title>
	<meta
		name="description"
		content="Parcha — Dr Toke's own thing. Discreet paper concepts in the lab: nothing for sale, the site stays a guide."
	/>
</svelte:head>

<main class="wrap">
	<header class="head">
		<p class="eyebrow">under products · our own thing</p>
		<h1>parcha</h1>
		<p class="lede">
			पर्चा — the paper. The first thing we'd ever put our own name on: discreet, unbranded,
			honest paper goods. These are lab concepts — <strong>nothing here is for sale</strong> and
			the site stays a guide. Informed by what the catalogue and the survey tell us people
			actually want.
		</p>
	</header>

	<div class="grid">
		{#each concepts as concept (concept.id)}
			<ConceptCard {concept} />
		{/each}
	</div>

	<p class="note">✎ this is the whole section for now — specs, materials, and the rest land here later.</p>
</main>

<style>
	.wrap {
		max-width: 1080px;
		margin: 0 auto;
		padding: 0 2rem 4rem;
	}
	.head {
		padding: 2.6rem 0 2.2rem;
	}
	.eyebrow {
		font-family: var(--font-pixel);
		font-size: 0.9rem;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--mute);
	}
	/* The parcha mark: the normal pixel face, lowercase — but iridescent
	   and slightly mysterious. Same scale as the other section heads. */
	h1 {
		font-family: var(--font-pixel);
		font-weight: 400;
		font-size: clamp(2.6rem, 6.5vw, 4.4rem);
		line-height: 1;
		background: linear-gradient(100deg, #9a8cff, #6fc3ff, #7fe0b2, #ffd98f, #ff9ad5, #9a8cff);
		background-size: 300% 100%;
		-webkit-background-clip: text;
		background-clip: text;
		color: transparent;
		animation: iridesce 9s linear infinite;
		margin: 0.2rem 0 0.8rem;
	}
	@keyframes iridesce {
		to {
			background-position: 300% 0;
		}
	}
	.lede {
		font-size: 1rem;
		line-height: 1.75;
		max-width: 44rem;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
		gap: 1rem;
		margin-top: 1rem;
	}
	.note {
		margin-top: 2rem;
		font-family: var(--font-pixel);
		font-size: 0.9rem;
		color: var(--mute);
		border: 1px dashed var(--mute);
		display: inline-block;
		padding: 0.25rem 0.6rem;
	}
</style>
