// Legality primer content. The state-by-state data itself is NOT written here —
// it lives in the backend (GET /api/states, DB-backed and health-checked).
// These blocks are the static framing above the live grid.

export interface LawBlock {
	id: string;
	title: string;
	body: string;
	todo?: string;
}

export const lawBlocks: LawBlock[] = [
	{
		id: 'ndps',
		title: 'What the NDPS Act actually bans',
		body: 'Since 1985 the NDPS Act prohibits cannabis flower (ganja) and resin (charas) nationally. The definition deliberately excludes the leaf — which is why bhang exists in a different legal universe from the same plant’s flower.',
		todo: 'plain-language walkthrough of §2(iii), penalties table, verify with counsel'
	},
	{
		id: 'bhang',
		title: 'The bhang carve-out',
		body: 'Bhang (leaf preparations) is regulated by each state’s excise department, not the NDPS Act. Some states license shops, some tolerate festival sale, some ban it outright — the grid below is the live, verified answer per state.',
		todo: 'how state excise licensing actually works, with citations per state'
	},
	{
		id: 'hemp',
		title: 'Hemp, Ayush & the legal market',
		body: 'The products Dr Toke catalogues are the legal market: Ayush-licensed cannabis medicine and FSSAI-compliant hemp food products under THC limits. Compliance is labelled honestly on every product — verified, pending, or prescription-required.',
		todo: 'THC thresholds, Ayush licence verification steps, FSSAI rules — verify wording'
	}
];
