// Purchase-token stash — port of stashPurchaseToken from apps/web's useAuth.
// Uses the same localStorage key so tokens survive a later port of the full
// pseudonymous auth system (claiming tokens for Verified Purchase badges).

const TOKENS_KEY = 'dt_tokens'; // pending purchase tokens awaiting claim

/** Append a purchase token to the pending-claim list (used by BuyButton). */
export function stashPurchaseToken(token: string): void {
	if (typeof window === 'undefined') return;
	let tokens: string[] = [];
	try {
		tokens = JSON.parse(localStorage.getItem(TOKENS_KEY) ?? '[]') as string[];
	} catch {
		tokens = [];
	}
	tokens.push(token);
	localStorage.setItem(TOKENS_KEY, JSON.stringify(tokens));
}
