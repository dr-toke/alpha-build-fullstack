// Science section content — conceptual and harm-reduction-first by design.
// Overviews only, never procedures. The backend also serves /api/roa with
// DB-backed routes-of-administration data if a live version is wanted later.

export interface Method {
	name: string;
	note: string;
}

export interface Topic {
	id: string;
	title: string;
	body: string;
	methods?: Method[];
	todo?: string;
}

export const topics: Topic[] = [
	{
		id: 'plant',
		title: 'The plant',
		body: 'One plant, dozens of active compounds. THC is the intoxicating cannabinoid; CBD is non-intoxicating and blunts some of THC’s edge; minor cannabinoids and terpenes shape the rest of the experience. Ratios matter more than totals — a 1:1 product is a different thing from an isolate.',
		todo: 'CB1/CB2 receptors, entourage effect, what "full spectrum" actually means'
	},
	{
		id: 'extraction',
		title: 'Extraction',
		body: 'How the market gets from plant to bottle — conceptually, and what each approach means for the consumer reading a label. This is a reading guide, not a manual.',
		methods: [
			{ name: 'CO₂', note: 'industry standard; clean, tunable, expensive equipment' },
			{ name: 'Ethanol', note: 'efficient at scale; typical for full-spectrum tinctures' },
			{ name: 'Solventless', note: 'ice-water hash and rosin — no chemistry, just cold and pressure' },
			{ name: 'Infusion', note: 'milk or ghee — how bhang has been prepared for centuries' }
		],
		todo: 'what residual-solvent testing is and why a COA should show it'
	},
	{
		id: 'decarb',
		title: 'Decarboxylation',
		body: 'Raw plant carries THCA and CBDA, which are not active as-is; gentle heat converts them into THC and CBD. It’s why edibles are cooked, why raw leaf juice doesn’t intoxicate, and why lab numbers list both acid and active forms.',
		todo: 'the THCA→THC numbers a label reader should know'
	},
	{
		id: 'roa',
		title: 'Routes & onset',
		body: 'How you take it matters as much as what you take. Inhaled: onset in minutes, fades in a couple of hours. Oral (edibles, bhang): onset 30–120 minutes, lasts 6–8 hours. The golden rule of harm reduction: wait for the full onset window before taking more — festival thandai is exactly where this goes wrong.',
		todo: 'bioavailability table per route; consider wiring the live /api/roa data here'
	},
	{
		id: 'coa',
		title: 'Reading a COA',
		body: 'A Certificate of Analysis is the third-party lab report behind an honest product: cannabinoid content, heavy metals, pesticides, residual solvents. If a brand can’t produce one, that is the answer. Verify the lab is NABL-accredited.',
		todo: 'annotated example COA, link to the NABL directory'
	}
];
