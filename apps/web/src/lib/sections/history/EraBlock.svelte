<script lang="ts">
	import type { Era } from './content';

	let { era }: { era: Era } = $props();
</script>

<article class="era" id={era.id}>
	<p class="years">{era.years}</p>
	<h2>{era.title}</h2>
	<p class="body">{era.body}</p>
	<ul>
		{#each era.points as point (point)}
			<li>{point}</li>
		{/each}
	</ul>
	{#if era.todo}
		<p class="todo">✎ to write: {era.todo}</p>
	{/if}
</article>

<style>
	.era {
		padding: 2.1rem 0;
		border-top: 1px solid var(--line);
	}
	.years {
		font-family: var(--font-pixel);
		font-size: 0.9rem;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--mute);
	}
	h2 {
		font-family: var(--font-pixel);
		font-size: clamp(1.9rem, 4vw, 2.8rem);
		line-height: 1.05;
		color: var(--ink);
		margin: 0.2rem 0 0.8rem;
	}
	.body {
		max-width: 46rem;
		font-size: 1rem;
		line-height: 1.75;
	}
	ul {
		margin: 0.9rem 0 0 1.1rem;
		max-width: 44rem;
	}
	li {
		margin-bottom: 0.3rem;
		font-size: 0.95rem;
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
</style>
