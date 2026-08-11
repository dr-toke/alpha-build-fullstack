<script lang="ts">
	import PageHead from '$lib/components/ui/PageHead.svelte';
	import TopicBlock from '$sections/science/TopicBlock.svelte';
	import RoaLive from '$sections/science/RoaLive.svelte';
	import { topics } from '$sections/science/content';
</script>

<svelte:head>
	<title>Science — Dr Toke</title>
	<meta
		name="description"
		content="The plant, the cannabinoids, extraction methods, decarboxylation, routes of administration, and how to read a COA — harm reduction first."
	/>
</svelte:head>

<main class="wrap">
	<PageHead
		word="SCIENCE"
		lede="What's actually in the bottle, how it got there, and how it behaves in your body. Harm reduction first."
	/>
	{#each topics as topic (topic.id)}
		<TopicBlock {topic} />
	{/each}

	<RoaLive />
</main>

<style>
	.wrap {
		max-width: 960px;
		margin: 0 auto;
		padding: 0 2rem 4rem;
	}
</style>
