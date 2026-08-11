<script lang="ts">
	// Migrated from ../dr-toke/apps/web/app/account/page.tsx.
	import CatalogShell from '$sections/products/CatalogShell.svelte';
	import { apiFetch, ApiError } from '$lib/api/client';
	import { auth } from '$lib/api/auth.svelte';
	import { Remote } from '$lib/api/remote.svelte';

	interface MyComment {
		id: string;
		body: string;
		rating: number | null;
		verified_purchase: boolean;
		created_at: string;
		is_post: boolean;
		post_slug?: string;
		post_title?: string;
		cluster_id?: string;
		product_name?: string;
	}

	interface MeData {
		account_id: string;
		handle: string;
		created_at: string;
	}

	const me = new Remote<MeData>();
	const myComments = new Remote<{ comments: MyComment[] }>();

	// (Re)load account data whenever a session appears.
	$effect(() => {
		if (auth.account) {
			me.load('/api/auth/me');
			myComments.load('/api/auth/my-comments');
		}
	});

	const comments = $derived(myComments.data?.comments ?? []);

	let token = $state('');
	let claimMsg = $state<string | null>(null);
	let claiming = $state(false);

	async function claim(e: SubmitEvent): Promise<void> {
		e.preventDefault();
		claimMsg = null;
		const t = token.trim();
		if (!t) return;
		claiming = true;
		try {
			await apiFetch('/api/auth/claim-token', {
				method: 'POST',
				body: JSON.stringify({ purchase_token: t })
			});
			claimMsg = '✓ Token claimed. Your reviews for that product are now verified.';
			token = '';
			myComments.refetch('/api/auth/my-comments');
		} catch (err) {
			claimMsg = err instanceof ApiError ? err.message : 'Could not claim token.';
		} finally {
			claiming = false;
		}
	}

	function fmtDate(iso: string): string {
		const d = new Date(iso);
		return Number.isNaN(d.getTime())
			? ''
			: d.toLocaleDateString('en-IN', { year: 'numeric', month: 'short', day: 'numeric' });
	}
</script>

<svelte:head>
	<title>Your Account — Dr Toke</title>
</svelte:head>

<CatalogShell>
	{#if auth.isLoading}
		<main class="max-w-[760px] mx-auto px-8 py-20 text-cream2">Loading…</main>
	{:else if !auth.account}
		<main class="max-w-[760px] mx-auto px-8 py-20 text-center">
			<h1 class="font-display font-bold text-cream text-[2rem] mb-4">Your Account</h1>
			<p class="text-cream2 mb-6">Sign in or create a pseudonymous handle to view your account.</p>
			<button
				onclick={auth.openAuth}
				class="px-5 py-2.5 rounded-[3px] font-semibold text-[0.85rem] uppercase tracking-[0.05em] cursor-pointer"
				style="background: var(--gold); color: var(--bg);"
			>
				Join / Sign in
			</button>
		</main>
	{:else}
		<main class="max-w-[760px] mx-auto px-8 py-16">
			<div class="flex items-center justify-between flex-wrap gap-3 mb-8">
				<div>
					<h1 class="font-display font-bold text-cream text-[2rem] leading-tight">
						@{auth.account.handle}
					</h1>
					{#if me.data}
						<p class="text-cream2 text-[0.8rem]">
							Joined {fmtDate(me.data.created_at)} · pseudonymous, no PII stored
						</p>
					{/if}
				</div>
				<button
					onclick={auth.logout}
					class="px-4 py-2 rounded-[3px] text-[0.78rem] text-cream2 cursor-pointer"
					style="border: 1px solid rgba(58,122,47,0.3);"
				>
					Log out
				</button>
			</div>

			<!-- Claim a purchase token -->
			<section
				class="rounded-[5px] p-5 mb-10"
				style="background: var(--bg2); border: 1px solid rgba(58,122,47,0.25);"
			>
				<h2 class="font-display font-bold text-cream text-[1.2rem] mb-1">Claim a purchase token</h2>
				<p class="text-cream2 text-[0.82rem] mb-3 leading-relaxed">
					Bought through a Dr Toke link before signing in? Paste the token to earn a Verified
					Purchase badge on your reviews. Tokens collected on this device are claimed automatically
					when you log in.
				</p>
				<form onsubmit={claim} class="flex gap-2 flex-wrap">
					<input
						type="text"
						bind:value={token}
						placeholder="purchase token"
						class="flex-1 min-w-[200px] px-3 py-2 rounded-[3px] text-cream text-[0.85rem] outline-none"
						style="background: var(--bg); border: 1px solid rgba(58,122,47,0.3);"
					/>
					<button
						type="submit"
						disabled={claiming}
						class="px-4 py-2 rounded-[3px] font-semibold text-[0.78rem] uppercase tracking-[0.05em] cursor-pointer disabled:opacity-50"
						style="background: var(--gold); color: var(--bg);"
					>
						{claiming ? 'Claiming…' : 'Claim'}
					</button>
				</form>
				{#if claimMsg}
					<p class="text-[0.8rem] mt-2 text-cream2">{claimMsg}</p>
				{/if}
			</section>

			<!-- My comments -->
			<section>
				<h2 class="font-display font-bold text-cream text-[1.3rem] mb-4">
					Your contributions
					<span class="text-cream2 text-[0.95rem] font-normal">({comments.length})</span>
				</h2>
				{#if comments.length === 0}
					<p class="text-cream2 opacity-70 text-[0.9rem]">
						Nothing yet. Review a
						<a href="/products" class="text-gold hover:underline">product</a>
						or join a
						<a href="/forum" class="text-gold hover:underline">forum discussion</a>.
					</p>
				{:else}
					<ul class="flex flex-col gap-3">
						{#each comments as c (c.id)}
							<li
								class="rounded-[5px] p-4"
								style="background: var(--bg2); border: 1px solid rgba(58,122,47,0.2);"
							>
								<div class="flex items-center gap-2 mb-1 flex-wrap text-[0.75rem]">
									{#if c.is_post}
										<a
											href={`/forum/post?slug=${encodeURIComponent(c.post_slug ?? '')}`}
											class="text-gold hover:underline"
										>
											{c.post_title || 'Forum post'}
										</a>
									{:else}
										<a
											href={`/product?id=${encodeURIComponent(c.cluster_id ?? '')}`}
											class="text-gold hover:underline"
										>
											{c.product_name || 'Product'}
										</a>
									{/if}
									{#if c.rating}<span class="text-gold">{'★'.repeat(c.rating)}</span>{/if}
									{#if c.verified_purchase}
										<span class="text-green2 text-[0.65rem]">✓ Verified</span>
									{/if}
									<span class="text-cream2 ml-auto">{fmtDate(c.created_at)}</span>
								</div>
								<p class="text-cream text-[0.88rem] leading-relaxed whitespace-pre-wrap">{c.body}</p>
							</li>
						{/each}
					</ul>
				{/if}
			</section>
		</main>
	{/if}
</CatalogShell>
