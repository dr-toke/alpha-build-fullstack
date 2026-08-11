<!-- Sticky in-page navigation for the legality hub. Anchor links only —
     works without JavaScript. -->
<script lang="ts">
	const anchors = [
		{ href: '#overview', label: 'overview' },
		{ href: '#ndps', label: 'ndps' },
		{ href: '#bhang', label: 'bhang' },
		{ href: '#hemp', label: 'hemp & ayush' },
		{ href: '#timeline', label: 'timeline' },
		{ href: '#penalties', label: 'penalties' },
		{ href: '#states', label: 'states' },
		{ href: '#archives', label: 'archives' },
		{ href: '#faq', label: 'faq' },
		{ href: '#glossary', label: 'glossary' }
	];
</script>

<nav class="subnav" aria-label="Legality sections">
	{#each anchors as a (a.href)}
		<a href={a.href}>{a.label}</a>
	{/each}
</nav>

<style>
	.subnav {
		position: sticky;
		top: 0;
		z-index: 20;
		display: flex;
		gap: 0.35rem;
		flex-wrap: nowrap;
		overflow-x: auto;
		padding: 0.55rem 0;
		background: var(--paper);
		border-bottom: 1px solid var(--ink);
		scrollbar-width: none;
	}
	.subnav::-webkit-scrollbar {
		display: none;
	}
	.subnav a {
		font-family: var(--font-pixel);
		font-size: 0.88rem;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		white-space: nowrap;
		padding: 0.22rem 0.6rem;
		border: 1px solid var(--ink);
		color: var(--ink);
	}
	.subnav a:hover,
	.subnav a:focus-visible {
		background: var(--ink);
		color: var(--paper);
	}
</style>
