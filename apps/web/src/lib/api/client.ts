// Centralised fetch wrapper — port of ../dr-toke/apps/web/lib/api.ts.
// - Base URL from VITE_API_URL (defaults to the demo-up.sh API address)
// - JSON parse + error normalisation
// - Injects the access token from localStorage (same dt_jwt key as apps/web)

export class ApiError extends Error {
	constructor(
		public status: number,
		message: string
	) {
		super(message);
		this.name = 'ApiError';
	}
}

export const API_BASE: string =
	(import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:8080';

function authHeader(): Record<string, string> {
	if (typeof window === 'undefined') return {};
	const token = localStorage.getItem('dt_jwt');
	return token ? { Authorization: `Bearer ${token}` } : {};
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(`${API_BASE}${path}`, {
		...init,
		headers: {
			'Content-Type': 'application/json',
			...authHeader(),
			...init?.headers
		}
	});

	if (!res.ok) {
		const body = (await res.json().catch(() => null)) as { error?: string } | null;
		throw new ApiError(res.status, body?.error ?? res.statusText ?? 'request failed');
	}

	// 204 / empty body tolerance
	const text = await res.text();
	return (text ? JSON.parse(text) : null) as T;
}
