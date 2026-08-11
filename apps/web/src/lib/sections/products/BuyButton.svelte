<script lang="ts">
	// Exact port of ../dr-toke/apps/web/components/products/BuyButton.tsx —
	// records the affiliate click via /api/checkout/initiate, stashes the
	// purchase token, and always lets the user through even if tracking fails.
	import { apiFetch } from '$lib/api/client';
	import { stashPurchaseToken } from '$lib/api/purchase';

	interface Props {
		listingId?: string;
		sourceUrl: string;
		affiliateUrl?: string | null;
		label?: string;
		pagePath?: string;
		/** Dense variant for table rows (small, outline style). */
		compact?: boolean;
	}

	interface CheckoutResponse {
		token: string;
		redirect_url: string;
	}

	let { listingId, sourceUrl, affiliateUrl, label, pagePath, compact }: Props = $props();

	let toast = $state<string | null>(null);
	let busy = $state(false);

	function openExternal(url: string) {
		window.open(url, '_blank', 'noopener,noreferrer');
	}

	async function handleClick() {
		const fallback = affiliateUrl || sourceUrl;

		// No tracked listing → just open the affiliate/source link.
		if (!listingId) {
			openExternal(fallback);
			return;
		}

		busy = true;
		try {
			const res = await apiFetch<CheckoutResponse>('/api/checkout/initiate', {
				method: 'POST',
				body: JSON.stringify({
					listing_id: listingId,
					page_path: pagePath ?? (typeof window !== 'undefined' ? window.location.pathname : '')
				})
			});
			stashPurchaseToken(res.token);
			openExternal(res.redirect_url || fallback);
			toast = 'Purchase saved. Create an account to earn a Verified Purchase badge.';
			window.setTimeout(() => (toast = null), 6000);
		} catch {
			// Checkout tracking failed — the user must still be able to buy.
			openExternal(fallback);
		} finally {
			busy = false;
		}
	}

	const btnClass = $derived(
		compact
			? 'inline-flex items-center gap-1 px-2 py-[3px] rounded-[3px] text-[0.7rem] cursor-pointer disabled:opacity-60 hover:opacity-90'
			: 'inline-flex items-center gap-1 px-4 py-2 rounded-[3px] font-semibold text-[0.8rem] tracking-[0.05em] uppercase cursor-pointer transition-transform duration-200 hover:-translate-y-[2px] disabled:opacity-60'
	);
	const btnStyle = $derived(
		compact
			? 'border: 1px solid var(--gold); color: var(--gold); background: transparent;'
			: 'background: var(--gold); color: var(--bg);'
	);
</script>

<button onclick={handleClick} disabled={busy} class={btnClass} style={btnStyle}>
	{busy ? 'Opening…' : (label ?? 'Buy')} ↗
</button>

{#if toast}
	<div
		class="fixed bottom-5 left-1/2 -translate-x-1/2 z-[300] px-4 py-3 rounded-[4px] text-[0.82rem] max-w-[90vw]"
		style="background: var(--bg3); border: 1px solid var(--green2); color: var(--cream);"
		role="status"
	>
		{toast}
	</div>
{/if}
