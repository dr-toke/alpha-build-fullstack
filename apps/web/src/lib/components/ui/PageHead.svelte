<script lang="ts">
	let { word, lede }: { word: string; lede: string } = $props();
</script>

<header class="head">
	<h1>{word}</h1>
	<p>{lede}</p>
</header>

<style>
	.head {
		padding: 2.6rem 0 2.2rem;
	}
	h1 {
		font-family: var(--font-pixel);
		font-size: clamp(2.6rem, 6.5vw, 4.4rem);
		line-height: 1;
		letter-spacing: 0.01em;
		color: var(--ink);
	}
	p {
		font-family: var(--font-pixel);
		font-size: 1.1rem;
		color: var(--mute);
		max-width: 40rem;
		margin-top: 0.7rem;
	}
</style>
