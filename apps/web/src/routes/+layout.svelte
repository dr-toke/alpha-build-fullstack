<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { initReveal } from '$lib/ui/reveal';
	import Disclaimer from '$lib/components/layout/Disclaimer.svelte';
	import Nav from '$lib/components/layout/Nav.svelte';
	import Footer from '$lib/components/layout/Footer.svelte';

	let { children } = $props();

	// The front page is its own navigation (the stacked words), so no nav bar there.
	const isHome = $derived(page.url.pathname === '/');

	$effect(() => initReveal());
</script>

<Disclaimer />
{#if !isHome}
	<Nav />
{/if}

{@render children()}

<Footer />
