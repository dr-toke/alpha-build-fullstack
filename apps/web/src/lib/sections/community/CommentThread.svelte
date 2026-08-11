<!-- Exact copy of ../dr-toke/apps/web/components/comments/CommentThread.tsx.
     Used on /product (reviews) and /forum/post (discussion). -->
<script lang="ts">
	import { apiFetch, ApiError } from '$lib/api/client';
	import { auth } from '$lib/api/auth.svelte';
	import VerifiedBadge from './VerifiedBadge.svelte';

	interface Comment {
		id: string;
		handle: string;
		body: string;
		verified_purchase: boolean;
		rating: number | null;
		created_at: string;
		is_own: boolean;
	}

	interface CommentList {
		comments: Comment[];
		total: number;
		page: number;
		per_page: number;
	}

	let {
		targetType,
		targetId
	}: { targetType: 'post' | 'product'; targetId: string } = $props();

	let comments = $state<Comment[]>([]);
	let total = $state(0);
	let page = $state(1);
	let loading = $state(true);
	let body = $state('');
	let rating = $state(0);
	let submitting = $state(false);
	let error = $state<string | null>(null);

	function basePath(type: 'post' | 'product', id: string): string {
		return type === 'post'
			? `/api/forum/posts/${encodeURIComponent(id)}/comments`
			: `/api/products/${encodeURIComponent(id)}/comments`;
	}

	function timeAgo(iso: string): string {
		const then = new Date(iso).getTime();
		if (Number.isNaN(then)) return '';
		const diff = Date.now() - then;
		const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });
		const mins = Math.round(diff / 60000);
		if (Math.abs(mins) < 60) return rtf.format(-mins, 'minute');
		const hours = Math.round(mins / 60);
		if (Math.abs(hours) < 24) return rtf.format(-hours, 'hour');
		const days = Math.round(hours / 24);
		return rtf.format(-days, 'day');
	}

	async function load(p: number): Promise<void> {
		loading = true;
		try {
			const data = await apiFetch<CommentList>(`${basePath(targetType, targetId)}?page=${p}&limit=20`);
			total = data.total;
			comments = p === 1 ? data.comments : [...comments, ...data.comments];
		} catch {
			// Leave existing comments; surface nothing noisy for a read failure.
		} finally {
			loading = false;
		}
	}

	// Reload from page 1 whenever the target changes.
	$effect(() => {
		void targetType;
		void targetId;
		page = 1;
		void load(1);
	});

	async function submit(e: SubmitEvent): Promise<void> {
		e.preventDefault();
		error = null;
		const trimmed = body.trim();
		if (trimmed.length < 10) {
			error = 'Comment must be at least 10 characters.';
			return;
		}

		submitting = true;
		try {
			const payload: { body: string; rating?: number } = { body: trimmed };
			if (targetType === 'product' && rating > 0) payload.rating = rating;

			const created = await apiFetch<Comment>(basePath(targetType, targetId), {
				method: 'POST',
				body: JSON.stringify(payload)
			});
			comments = [created, ...comments];
			total += 1;
			body = '';
			rating = 0;
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Could not post comment.';
		} finally {
			submitting = false;
		}
	}

	async function remove(id: string): Promise<void> {
		// Optimistic removal.
		const snapshot = comments;
		comments = comments.filter((c) => c.id !== id);
		total = Math.max(0, total - 1);
		try {
			await apiFetch(`/api/comments/${encodeURIComponent(id)}`, { method: 'DELETE' });
		} catch {
			comments = snapshot; // revert on failure
			total += 1;
		}
	}

	function loadMore(): void {
		page += 1;
		void load(page);
	}

	const canLoadMore = $derived(comments.length < total);
</script>

<section class="mt-12">
	<h2 class="font-display font-bold text-cream text-[1.4rem] mb-5">
		Community {targetType === 'product' ? 'Reviews' : 'Discussion'}
		<span class="text-cream2 text-[1rem] font-normal">({total})</span>
	</h2>

	<!-- Composer -->
	{#if auth.account}
		<form
			onsubmit={submit}
			class="mb-8 rounded-[5px] p-4"
			style="background: var(--bg2); border: 1px solid rgba(58,122,47,0.28);"
		>
			{#if targetType === 'product'}
				<div class="flex items-center gap-1 mb-3" role="radiogroup" aria-label="Rating">
					{#each [1, 2, 3, 4, 5] as n (n)}
						<button
							type="button"
							onclick={() => (rating = n === rating ? 0 : n)}
							class="text-[1.3rem] leading-none cursor-pointer transition-transform duration-150 hover:scale-110"
							style={`color: ${n <= rating ? 'var(--gold)' : 'var(--cream2)'};`}
							aria-label={`${n} star${n > 1 ? 's' : ''}`}
							aria-pressed={n <= rating}
						>
							★
						</button>
					{/each}
					<span class="text-[0.72rem] text-cream2 ml-2">
						{rating > 0 ? `${rating}/5` : 'Rate (optional)'}
					</span>
				</div>
			{/if}
			<textarea
				bind:value={body}
				placeholder={targetType === 'product'
					? 'Share your experience with this product…'
					: 'Add to the discussion…'}
				maxlength={1000}
				rows={3}
				class="w-full px-3 py-2 rounded-[3px] text-cream text-[0.9rem] outline-none resize-y"
				style="background: var(--bg); border: 1px solid rgba(58,122,47,0.3);"
			></textarea>
			<div class="flex justify-between items-center mt-2">
				<span class="text-[0.7rem] text-cream2">
					Posting as <span class="text-gold">@{auth.account.handle}</span> · {body.length}/1000
				</span>
				{#if error}
					<span class="text-[0.75rem]" style="color: var(--red2);">{error}</span>
				{/if}
				<button
					type="submit"
					disabled={submitting}
					class="px-4 py-1.5 rounded-[3px] font-semibold text-[0.78rem] tracking-[0.05em] uppercase cursor-pointer disabled:opacity-50"
					style="background: var(--gold); color: var(--bg);"
				>
					{submitting ? 'Posting…' : 'Post'}
				</button>
			</div>
		</form>
	{:else}
		<button
			onclick={auth.openAuth}
			class="mb-8 px-4 py-2.5 rounded-[3px] font-medium text-[0.82rem] cursor-pointer transition-transform duration-200 hover:-translate-y-[2px]"
			style="background: transparent; border: 1px solid var(--gold); color: var(--gold);"
		>
			Join to {targetType === 'product' ? 'review' : 'comment'} →
		</button>
	{/if}

	<!-- List -->
	{#if comments.length === 0 && !loading}
		<p class="text-cream2 text-[0.9rem] opacity-70 py-6">
			No {targetType === 'product' ? 'reviews' : 'comments'} yet. Be the first.
		</p>
	{:else}
		<ul class="flex flex-col gap-4">
			{#each comments as c (c.id)}
				<li
					class="rounded-[5px] p-4"
					style="background: var(--bg2); border: 1px solid rgba(58,122,47,0.2);"
				>
					<div class="flex items-center gap-2 mb-2 flex-wrap">
						<span
							class="w-7 h-7 rounded-full flex items-center justify-center text-[0.8rem] font-bold"
							style="background: var(--green3); color: var(--cream);"
							aria-hidden="true"
						>
							{c.handle.charAt(0).toUpperCase()}
						</span>
						<span class="text-gold text-[0.85rem] font-medium">@{c.handle}</span>
						{#if c.rating}
							<span class="text-[0.8rem]" style="color: var(--gold);">
								{'★'.repeat(c.rating)}<span style="color: var(--cream2);">{'★'.repeat(5 - c.rating)}</span>
							</span>
						{/if}
						{#if c.verified_purchase}
							<VerifiedBadge />
						{/if}
						<span class="text-cream2 text-[0.72rem] ml-auto">{timeAgo(c.created_at)}</span>
					</div>
					<p class="text-cream text-[0.9rem] leading-relaxed whitespace-pre-wrap">{c.body}</p>
					{#if c.is_own}
						<button
							onclick={() => remove(c.id)}
							class="text-[0.72rem] text-cream2 hover:text-[var(--red2)] mt-2 cursor-pointer"
						>
							Delete
						</button>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}

	{#if canLoadMore}
		<button
			onclick={loadMore}
			disabled={loading}
			class="mt-6 px-4 py-2 rounded-[3px] text-[0.8rem] text-cream2 cursor-pointer disabled:opacity-50"
			style="border: 1px solid rgba(58,122,47,0.3);"
		>
			{loading ? 'Loading…' : 'Load more'}
		</button>
	{/if}
</section>
