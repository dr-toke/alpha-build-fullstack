<script lang="ts">
	import { page } from '$app/state';
	import { auth } from '$lib/api/auth.svelte';

	const links: { href: string; label: string; cyber?: boolean; core?: boolean }[] = [
		{ href: '/history', label: 'history', core: true },
		{ href: '/science', label: 'science', core: true },
		{ href: '/legality', label: 'legality', core: true },
		{ href: '/products', label: 'products', core: true },
		{ href: '/brands', label: 'brands' },
		{ href: '/compare', label: 'compare' },
		{ href: '/forum', label: 'forum' },
		{ href: '/survey', label: 'survey' },
		{ href: '/parcha', label: 'parcha', cyber: true }
	];

	const path = $derived(page.url.pathname);

	// Show @handle instead of "account" once the pseudonymous session restores.
	$effect(() => auth.init());

	function isActive(href: string): boolean {
		if (path === href || path.startsWith(href + '/')) return true;
		// /product?id=… belongs to the products section
		return href === '/products' && path === '/product';
	}
</script>

<header class="nav">
	<a class="mark" href="/">dr. toke</a>
	<nav aria-label="Sections">
		{#each links as l (l.href)}
			<a href={l.href} class:active={isActive(l.href)} class:cyber={l.cyber} class:core={l.core}>{l.label}</a>
		{/each}
		<a href="/account" class="account" class:active={isActive('/account')}>
			{auth.account ? `@${auth.account.handle}` : 'account'}
		</a>
	</nav>
</header>

<style>
	.nav {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 1rem;
		flex-wrap: wrap;
		padding: 0.85rem 1.6rem;
		background: var(--paper);
		border-bottom: 1px solid var(--line);
	}
	.mark {
		font-family: var(--font-logo);
		font-weight: 700;
		font-size: 1.4rem;
		line-height: 1;
		color: var(--mood-green);
		user-select: none;
		transition: transform 0.16s steps(3, end);
	}
	/* Funky: the wordmark goes iridescent and does a pixel-jump on hover. */
	.mark:hover {
		background: linear-gradient(100deg, #9a8cff, #6fc3ff, #7fe0b2, #ffd98f, #ff9ad5, #9a8cff);
		background-size: 300% 100%;
		-webkit-background-clip: text;
		background-clip: text;
		color: transparent;
		animation: navIridesce 3s linear infinite;
		transform: translateY(-2px) rotate(-2deg);
	}
	nav {
		display: flex;
		gap: 1rem;
		flex-wrap: wrap;
		align-items: baseline;
	}
	nav a {
		font-family: var(--font-pixel);
		font-size: 1rem;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: var(--mute);
		padding: 0.12rem 0.5rem;
		border: 1px solid transparent;
		/* steps() = chunky 8-bit motion instead of a smooth glide */
		transition:
			transform 0.14s steps(2, end),
			box-shadow 0.14s steps(2, end),
			background 0.14s steps(2, end),
			color 0.14s steps(2, end);
	}
	nav a.core {
		color: var(--mood-purple-deep);
	}
	/* Funky: links flip into hard-shadowed pixel chips. */
	nav a:hover {
		color: var(--paper);
		background: var(--ink);
		border-color: var(--ink);
		transform: translate(-2px, -2px) rotate(-1.5deg);
		box-shadow: 3px 3px 0 var(--mood-green-neon);
	}
	nav a.active {
		color: var(--paper);
		background: var(--mood-green);
		border-color: var(--mood-green);
		box-shadow: 2px 2px 0 var(--mood-purple);
	}
	nav a.active:hover {
		background: var(--ink);
		border-color: var(--ink);
		box-shadow: 3px 3px 0 var(--mood-green-neon);
	}
	/* account: lowercase, rose — the door to the pseudonymous tier */
	nav a.account {
		text-transform: lowercase;
		color: var(--mood-rose);
	}
	nav a.account:hover {
		color: var(--paper);
		background: var(--mood-rose);
		border-color: var(--mood-rose);
		box-shadow: 3px 3px 0 var(--mood-peach);
	}
	nav a.account.active {
		color: var(--paper);
		background: var(--mood-rose);
		border-color: var(--mood-rose);
		box-shadow: 2px 2px 0 var(--mood-peach);
	}
	/* parcha: same size as its neighbours, lowercase, iridescent and a little mysterious */
	nav a.cyber {
		text-transform: lowercase;
		background: linear-gradient(100deg, #9a8cff, #6fc3ff, #7fe0b2, #ffd98f, #ff9ad5, #9a8cff);
		background-size: 300% 100%;
		-webkit-background-clip: text;
		background-clip: text;
		color: transparent;
		animation: navIridesce 9s linear infinite;
	}
	nav a.cyber:hover {
		filter: drop-shadow(0 0 5px rgba(154, 140, 255, 0.5));
		transform: translate(-2px, -2px) rotate(-1.5deg);
		background: linear-gradient(100deg, #9a8cff, #6fc3ff, #7fe0b2, #ffd98f, #ff9ad5, #9a8cff);
		background-size: 300% 100%;
		-webkit-background-clip: text;
		background-clip: text;
		border-color: transparent;
		box-shadow: none;
	}
	@keyframes navIridesce {
		to {
			background-position: 300% 0;
		}
	}
</style>
