// Shared types + display helpers for the live product catalog (from the Go API).
// Exact port of ../dr-toke/apps/web/lib/catalog.ts — keep in sync with it.

export interface ApiBrandSummary {
	slug: string;
	name: string;
	ayush: boolean;
	fssai: boolean;
	verified: boolean;
}

export interface ApiCannabinoids {
	cbd_mg: number | null;
	thc_mg: number | null;
	total_cannabinoids_mg: number | null;
	concentration_type: string; // cbd | thc | total | hemp_seed | unknown
}

export interface ApiListing {
	listing_id?: string;
	source: string;
	price_inr: number;
	source_url: string;
	affiliate_url: string | null;
	in_stock: boolean;
}

export interface ApiProduct {
	id: string;
	name: string;
	slug: string;
	brand: ApiBrandSummary;
	category: string;
	categories: string[];
	extract_type: string;
	carrier_oil: string;
	cannabinoids: ApiCannabinoids;
	volume_ml: number | null;
	weight_g: number | null;
	best_price_inr: number | null;
	best_price_per_mg: number | null;
	cbd_price_per_mg: number | null;
	thc_price_per_mg: number | null;
	price_per_mg_basis: string;
	best_listing: ApiListing | null;
	other_listings: ApiListing[];
	image_url: string | null;
	coa_available: boolean;
	in_stock: boolean;
	prescription_required: boolean;
	first_seen_at: string;
}

export interface ProductListResponse {
	products: ApiProduct[];
	total: number;
	page: number;
	per_page: number;
}

export interface ApiBrand {
	id: string;
	slug: string;
	name: string;
	full_name?: string;
	city?: string;
	state?: string;
	website_url?: string;
	description?: string;
	ayush: boolean;
	fssai: boolean;
	coa_available: boolean;
	verified: boolean;
	highlight?: string;
	last_verified?: string;
	categories: string[];
	product_count: number;
}

export interface BrandListResponse {
	brands: ApiBrand[];
}

interface CategoryStyle {
	bg: string;
	border: string;
	text: string;
}

export const CATEGORY_COLORS: Record<string, CategoryStyle> = {
	tincture: { bg: 'rgba(200,168,75,0.15)', border: 'var(--gold)', text: 'var(--gold)' },
	smokable: { bg: 'rgba(224,122,90,0.15)', border: 'var(--red2)', text: 'var(--red2)' },
	vapeable: { bg: 'rgba(90,170,74,0.15)', border: 'var(--green2)', text: 'var(--green2)' },
	edible: { bg: 'rgba(200,168,75,0.10)', border: 'var(--gold3)', text: 'var(--gold2)' },
	topical: { bg: 'rgba(176,160,128,0.15)', border: 'var(--cream2)', text: 'var(--cream2)' },
	extract: { bg: 'rgba(200,168,75,0.12)', border: 'var(--gold)', text: 'var(--gold)' },
	nutrition: { bg: 'rgba(58,122,47,0.12)', border: 'var(--green3)', text: 'var(--green2)' },
	beverage: { bg: 'rgba(90,170,74,0.12)', border: 'var(--green2)', text: 'var(--green2)' },
	pet: { bg: 'rgba(176,160,128,0.12)', border: 'var(--cream2)', text: 'var(--cream)' },
	apparel: { bg: 'rgba(176,160,128,0.10)', border: 'var(--cream2)', text: 'var(--cream2)' },
	other: { bg: 'rgba(176,160,128,0.08)', border: 'var(--cream2)', text: 'var(--cream2)' }
};

const OTHER_STYLE: CategoryStyle = {
	bg: 'rgba(176,160,128,0.08)',
	border: 'var(--cream2)',
	text: 'var(--cream2)'
};

export function categoryStyle(cat: string): CategoryStyle {
	return CATEGORY_COLORS[cat] ?? OTHER_STYLE;
}

/** ₹/mg value tier → colour. Returns null when ₹/mg is not applicable. */
export function pmgColor(perMg: number | null): string | null {
	if (perMg == null) return null;
	if (perMg < 3) return 'var(--green2)';
	if (perMg <= 8) return 'var(--gold)';
	return 'var(--cream2)';
}

export function formatINR(n: number | null | undefined): string {
	if (n == null) return '—';
	return '₹' + Math.round(n).toLocaleString('en-IN');
}

export function hasSmokable(p: ApiProduct): boolean {
	return p.category === 'smokable' || p.categories.includes('smokable');
}

export function prettyExtract(s: string): string {
	if (!s || s === 'unknown') return '';
	return s.replace(/_/g, ' ');
}
