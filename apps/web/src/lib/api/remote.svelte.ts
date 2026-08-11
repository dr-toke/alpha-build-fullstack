// Svelte-runes equivalent of apps/web's useFetch hook: fetch whenever the
// path changes, expose data / error / loading, ignore stale responses.
//
// Usage inside a component:
//   const remote = new Remote<ProductListResponse>();
//   $effect(() => remote.load(`/api/products?${query}`));

import { apiFetch, ApiError } from './client';

export class Remote<T> {
	data = $state<T | null>(null);
	error = $state<ApiError | null>(null);
	// Starts true so the first client render shows the loading skeleton
	// instead of flashing an empty/error state (matches useFetch).
	loading = $state(true);

	#seq = 0;

	load(path: string | null): void {
		const seq = ++this.#seq;
		if (path === null) {
			this.loading = false;
			return;
		}
		this.loading = true;
		this.error = null;

		apiFetch<T>(path)
			.then((d) => {
				if (seq === this.#seq) this.data = d;
			})
			.catch((e: unknown) => {
				if (seq !== this.#seq) return;
				this.error = e instanceof ApiError ? e : new ApiError(0, 'network error');
			})
			.finally(() => {
				if (seq === this.#seq) this.loading = false;
			});
	}

	refetch(path: string): void {
		this.load(path);
	}
}
