<script lang="ts">
	import PageHead from '$lib/components/ui/PageHead.svelte';
	import EraBlock from '$sections/history/EraBlock.svelte';
	import { eras } from '$sections/history/content';
</script>

<svelte:head>
	<title>History — Dr Toke</title>
	<meta
		name="description"
		content="Cannabis in India from the Atharvaveda to the NDPS Act: how it started, how emperors and mystics consumed it, why it was banned, and how the rest of the world moved on."
	/>
</svelte:head>

<main class="wrap">
	<PageHead
		word="HISTORY"
		lede="How it started, how it was consumed, why it was banned — and what the rest of the world did next."
	/>
	{#each eras as era (era.id)}
		<EraBlock {era} />
	{/each}
</main>

<style>
	.wrap {
		max-width: 960px;
		margin: 0 auto;
		padding: 0 2rem 4rem;
	}
</style>
