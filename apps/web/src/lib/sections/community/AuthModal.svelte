<!-- Exact copy of ../dr-toke/apps/web/components/auth/AuthModal.tsx.
     Mounted once inside CatalogShell so it is available on every dark page. -->
<script lang="ts">
	import { ApiError } from '$lib/api/client';
	import { auth } from '$lib/api/auth.svelte';

	type Tab = 'register' | 'login';

	const HANDLE_RE = /^[A-Za-z0-9_]{3,24}$/;

	let tab = $state<Tab>('register');
	let handle = $state('');
	let password = $state('');
	let confirm = $state('');
	let showPw = $state(false);
	let error = $state<string | null>(null);
	let busy = $state(false);

	// Reset fields when opened/closed.
	$effect(() => {
		if (!auth.isAuthOpen) {
			handle = '';
			password = '';
			confirm = '';
			error = null;
			busy = false;
		}
	});

	function onKeydown(e: KeyboardEvent) {
		if (auth.isAuthOpen && e.key === 'Escape') auth.closeAuth();
	}

	function setTab(t: Tab) {
		tab = t;
		error = null;
	}

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		error = null;

		if (!HANDLE_RE.test(handle)) {
			error = 'Handle must be 3–24 characters: letters, numbers, underscores.';
			return;
		}
		if (password.length < 8) {
			error = 'Password must be at least 8 characters.';
			return;
		}
		if (tab === 'register' && password !== confirm) {
			error = 'Passwords do not match.';
			return;
		}

		busy = true;
		try {
			if (tab === 'register') {
				await auth.register(handle, password);
			} else {
				await auth.login(handle, password);
			}
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Something went wrong. Try again.';
			busy = false;
		}
	}
</script>

<svelte:window onkeydown={onKeydown} />

{#if auth.isAuthOpen}
	<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
	<div
		onclick={auth.closeAuth}
		class="fixed inset-0 z-[200] flex items-center justify-center p-4"
		style="background: rgba(8,15,8,0.8); backdrop-filter: blur(6px);"
		role="dialog"
		aria-modal="true"
		aria-label="Account"
		tabindex="-1"
	>
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
		<div
			onclick={(e) => e.stopPropagation()}
			class="w-full max-w-[420px] rounded-[6px] p-7"
			style="background: var(--bg2); border: 1px solid rgba(58,122,47,0.3);"
		>
			<div class="flex justify-between items-start mb-5">
				<h2 class="font-display font-bold text-cream text-[1.5rem]">
					{tab === 'register' ? 'Create a handle' : 'Sign in'}
				</h2>
				<button
					onclick={auth.closeAuth}
					class="text-cream2 hover:text-cream text-xl leading-none cursor-pointer"
					aria-label="Close"
				>
					×
				</button>
			</div>

			<!-- Tabs -->
			<div class="flex gap-1 mb-6 p-1 rounded-[4px]" style="background: var(--bg);">
				{#each ['register', 'login'] as const as t (t)}
					<button
						onclick={() => setTab(t)}
						class="flex-1 text-[0.8rem] py-2 rounded-[3px] font-medium tracking-[0.04em] uppercase transition-colors duration-200 cursor-pointer"
						style={`background: ${tab === t ? 'var(--gold)' : 'transparent'}; color: ${tab === t ? 'var(--bg)' : 'var(--cream2)'};`}
					>
						{t === 'register' ? 'Create Handle' : 'Sign In'}
					</button>
				{/each}
			</div>

			<form onsubmit={submit} class="flex flex-col gap-4">
				<div>
					<label class="block text-[0.7rem] tracking-[0.12em] uppercase text-cream2 mb-1" for="auth-handle">
						Handle
					</label>
					<input
						id="auth-handle"
						type="text"
						bind:value={handle}
						autocomplete="username"
						placeholder="e.g. green_seeker"
						class="w-full px-3 py-2 rounded-[3px] text-cream text-[0.9rem] outline-none"
						style="background: var(--bg); border: 1px solid rgba(58,122,47,0.35);"
					/>
				</div>

				<div>
					<label class="block text-[0.7rem] tracking-[0.12em] uppercase text-cream2 mb-1" for="auth-password">
						Password
					</label>
					<div class="relative">
						{#if showPw}
							<input
								id="auth-password"
								type="text"
								bind:value={password}
								autocomplete={tab === 'register' ? 'new-password' : 'current-password'}
								class="w-full px-3 py-2 pr-14 rounded-[3px] text-cream text-[0.9rem] outline-none"
								style="background: var(--bg); border: 1px solid rgba(58,122,47,0.35);"
							/>
						{:else}
							<input
								id="auth-password"
								type="password"
								bind:value={password}
								autocomplete={tab === 'register' ? 'new-password' : 'current-password'}
								class="w-full px-3 py-2 pr-14 rounded-[3px] text-cream text-[0.9rem] outline-none"
								style="background: var(--bg); border: 1px solid rgba(58,122,47,0.35);"
							/>
						{/if}
						<button
							type="button"
							onclick={() => (showPw = !showPw)}
							class="absolute right-2 top-1/2 -translate-y-1/2 text-[0.7rem] text-gold cursor-pointer"
						>
							{showPw ? 'Hide' : 'Show'}
						</button>
					</div>
				</div>

				{#if tab === 'register'}
					<div>
						<label class="block text-[0.7rem] tracking-[0.12em] uppercase text-cream2 mb-1" for="auth-confirm">
							Confirm password
						</label>
						{#if showPw}
							<input
								id="auth-confirm"
								type="text"
								bind:value={confirm}
								autocomplete="new-password"
								class="w-full px-3 py-2 rounded-[3px] text-cream text-[0.9rem] outline-none"
								style="background: var(--bg); border: 1px solid rgba(58,122,47,0.35);"
							/>
						{:else}
							<input
								id="auth-confirm"
								type="password"
								bind:value={confirm}
								autocomplete="new-password"
								class="w-full px-3 py-2 rounded-[3px] text-cream text-[0.9rem] outline-none"
								style="background: var(--bg); border: 1px solid rgba(58,122,47,0.35);"
							/>
						{/if}
					</div>
				{/if}

				{#if error}
					<p class="text-[0.8rem]" style="color: var(--red2);">{error}</p>
				{/if}

				<button
					type="submit"
					disabled={busy}
					class="mt-1 py-2.5 rounded-[3px] font-semibold text-[0.85rem] tracking-[0.05em] uppercase cursor-pointer transition-transform duration-200 hover:-translate-y-[2px] disabled:opacity-50 disabled:cursor-not-allowed"
					style="background: var(--gold); color: var(--bg);"
				>
					{busy ? 'Please wait…' : tab === 'register' ? 'Create account' : 'Sign in'}
				</button>
			</form>

			<p class="text-[0.72rem] text-cream2 leading-relaxed mt-5 opacity-80">
				No email required. Your handle is pseudonymous — we store no name, email,
				phone, or IP. If you lose your password it cannot be recovered.
			</p>
		</div>
	</div>
{/if}
