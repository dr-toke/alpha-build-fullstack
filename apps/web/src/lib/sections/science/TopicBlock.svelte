<script lang="ts">
	import type { Topic } from './content';

	let { topic }: { topic: Topic } = $props();
</script>

<article class="topic" id={topic.id}>
	<h2>{topic.title}</h2>
	<p class="body">{topic.body}</p>

	{#if topic.methods}
		<div class="methods">
			{#each topic.methods as m (m.name)}
				<div class="method">
					<h3>{m.name}</h3>
					<p>{m.note}</p>
				</div>
			{/each}
		</div>
	{/if}

	{#if topic.todo}
		<p class="todo">✎ to write: {topic.todo}</p>
	{/if}
</article>

<style>
	.topic {
		padding: 2.1rem 0;
		border-top: 1px solid var(--line);
	}
	h2 {
		font-family: var(--font-pixel);
		font-size: clamp(1.9rem, 4vw, 2.8rem);
		line-height: 1.05;
		color: var(--ink);
		margin-bottom: 0.8rem;
	}
	.body {
		max-width: 46rem;
		font-size: 1rem;
		line-height: 1.75;
	}
	.methods {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
		gap: 0.9rem;
		margin-top: 1.2rem;
		max-width: 46rem;
	}
	.method {
		border: 1px solid var(--ink);
		padding: 0.8rem 0.9rem;
	}
	.method h3 {
		font-family: var(--font-pixel);
		font-size: 1.15rem;
		margin-bottom: 0.25rem;
	}
	.method p {
		font-size: 0.88rem;
		line-height: 1.55;
		color: var(--mute);
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
