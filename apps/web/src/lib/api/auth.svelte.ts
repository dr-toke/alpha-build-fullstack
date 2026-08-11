// Pseudonymous auth — port of apps/web/components/auth/useAuth.tsx as a
// Svelte-runes store. Same localStorage keys (dt_jwt / dt_refresh / dt_tokens)
// so sessions and pending purchase tokens are interchangeable with apps/web.
// No PII anywhere: handle + password only.

import { apiFetch, ApiError } from './client';

export interface Account {
	id: string;
	handle: string;
}

interface AuthResponse {
	account_id: string;
	handle: string;
	access_token: string;
	refresh_token: string;
}

const JWT_KEY = 'dt_jwt';
const REFRESH_KEY = 'dt_refresh';
const TOKENS_KEY = 'dt_tokens'; // pending purchase tokens awaiting claim

function storeTokens(res: AuthResponse): void {
	localStorage.setItem(JWT_KEY, res.access_token);
	localStorage.setItem(REFRESH_KEY, res.refresh_token);
}

function clearTokens(): void {
	localStorage.removeItem(JWT_KEY);
	localStorage.removeItem(REFRESH_KEY);
}

class AuthStore {
	account = $state<Account | null>(null);
	isLoading = $state(true);
	isAuthOpen = $state(false);

	#initialised = false;

	openAuth = (): void => {
		this.isAuthOpen = true;
	};

	closeAuth = (): void => {
		this.isAuthOpen = false;
	};

	/** Restore the session once on first client render (call from $effect). */
	init = (): void => {
		if (this.#initialised || typeof window === 'undefined') return;
		this.#initialised = true;
		void this.#restore();
	};

	login = async (handle: string, password: string): Promise<void> => {
		const res = await apiFetch<AuthResponse>('/api/auth/login', {
			method: 'POST',
			body: JSON.stringify({ handle, password })
		});
		await this.#finishAuth(res);
	};

	register = async (handle: string, password: string): Promise<void> => {
		const res = await apiFetch<AuthResponse>('/api/auth/register', {
			method: 'POST',
			body: JSON.stringify({ handle, password })
		});
		await this.#finishAuth(res);
	};

	logout = (): void => {
		const refresh = localStorage.getItem(REFRESH_KEY);
		if (refresh) {
			void apiFetch('/api/auth/logout', {
				method: 'POST',
				body: JSON.stringify({ refresh_token: refresh })
			}).catch(() => {});
		}
		clearTokens();
		this.account = null;
	};

	claimToken = async (purchaseToken: string): Promise<void> => {
		await apiFetch('/api/auth/claim-token', {
			method: 'POST',
			body: JSON.stringify({ purchase_token: purchaseToken })
		});
	};

	async #finishAuth(res: AuthResponse): Promise<void> {
		storeTokens(res);
		this.account = { id: res.account_id, handle: res.handle };
		this.isAuthOpen = false;
		await this.#autoClaimPending();
	}

	// Claim any purchase tokens collected before the user had an account.
	async #autoClaimPending(): Promise<void> {
		let pending: string[] = [];
		try {
			pending = JSON.parse(localStorage.getItem(TOKENS_KEY) ?? '[]') as string[];
		} catch {
			pending = [];
		}
		if (!pending.length) return;

		const remaining: string[] = [];
		for (const token of pending) {
			try {
				await this.claimToken(token);
			} catch (e) {
				// Keep tokens that failed transiently; drop ones rejected as invalid/claimed.
				if (e instanceof ApiError && e.status >= 500) remaining.push(token);
			}
		}
		localStorage.setItem(TOKENS_KEY, JSON.stringify(remaining));
	}

	// Validate access token, refresh once if expired.
	async #restore(): Promise<void> {
		const jwt = localStorage.getItem(JWT_KEY);
		const refresh = localStorage.getItem(REFRESH_KEY);
		if (!jwt && !refresh) {
			this.isLoading = false;
			return;
		}

		try {
			const me = await apiFetch<{ account_id: string; handle: string }>('/api/auth/me');
			this.account = { id: me.account_id, handle: me.handle };
		} catch {
			// Access token likely expired — try a single refresh.
			if (refresh) {
				try {
					const res = await apiFetch<AuthResponse>('/api/auth/refresh', {
						method: 'POST',
						body: JSON.stringify({ refresh_token: refresh })
					});
					storeTokens(res);
					this.account = { id: res.account_id, handle: res.handle };
				} catch {
					clearTokens();
				}
			} else {
				clearTokens();
			}
		} finally {
			this.isLoading = false;
		}
	}
}

export const auth = new AuthStore();
