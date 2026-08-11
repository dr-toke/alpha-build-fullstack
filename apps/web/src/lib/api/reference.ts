// Shared types for self-correcting reference content served by the Go API:
// state legal guide (/api/states), routes of administration (/api/roa), and
// marketplace aggregators (/api/aggregators). Port of apps/web/lib/reference.ts
// — see ../dr-toke/apps/api/internal/api/reference.go for the source of truth.

export type LegalStatus = 'legal' | 'tolerated' | 'grey' | 'limited' | 'illegal';

export interface ApiState {
	slug: string;
	name: string;
	status: LegalStatus;
	bhang_shops: string;
	detail: string;
	excise_url?: string; // withheld by the API when the link is broken
	notes?: string;
	featured: boolean;
	last_verified: string; // YYYY-MM-DD
	stale: boolean; // last_verified older than the verify interval
}

export interface StateListResponse {
	states: ApiState[];
}

export interface ApiROAMethod {
	slug: string;
	method: string;
	onset: string;
	duration: string;
	bioavailability: string;
	pros: string[];
	cons: string[];
	best_for: string[];
	warning_note?: string;
	last_verified: string;
}

export interface ROAListResponse {
	methods: ApiROAMethod[];
}

export interface ApiAggregator {
	slug: string;
	name: string;
	url: string;
	description: string;
	brand_count?: string; // live-derived where we scrape the source, else label
	product_count?: string;
	last_verified: string;
	stale: boolean;
}

export interface AggregatorListResponse {
	aggregators: ApiAggregator[];
}
