<!-- Wraps the catalog routes in the dr-toke dark world. Everything inside is
     an EXACT copy of ../dr-toke/apps/web — do not restyle without checking
     against the original. Tokens live in app.css under "catalog".
     Also hosts the community plumbing: restores the pseudonymous session and
     mounts the AuthModal so any dark page can call auth.openAuth(). -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { auth } from '$lib/api/auth.svelte';
	import AuthModal from '$sections/community/AuthModal.svelte';

	let { children }: { children: Snippet } = $props();

	$effect(() => auth.init());
</script>

<div class="catalog-shell">
	{@render children()}
	<AuthModal />
</div>
