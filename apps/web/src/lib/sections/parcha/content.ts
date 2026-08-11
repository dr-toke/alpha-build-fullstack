// PARCHA — our own thing. Concept previews only: nothing here is for sale,
// nothing has a price, and the site stays a guide (CLAUDE.md hard rule:
// never facilitate a transaction).

export type ConceptVisual = 'cig' | 'paper' | 'grid' | 'pouch';

export interface Concept {
	id: string;
	name: string;
	desc: string;
	visual: ConceptVisual;
}

export const concepts: Concept[] = [
	{
		id: 'cigarette-roach',
		name: 'the cigarette roach',
		desc: 'Filter tips that read as a plain cigarette — orange wrap, white body. Discreet by design, nothing to announce.',
		visual: 'cig'
	},
	{
		id: 'full-white',
		name: 'full white',
		desc: 'Papers with no brand, no watermark, no colour. A clean, slow, full-white sheet and that’s it.',
		visual: 'paper'
	},
	{
		id: 'grid-sheet',
		name: 'the grid sheet',
		desc: 'One sheet of roach card, dot-perforated into a grid — tear one off when you need it, the rest stays flat in your wallet.',
		visual: 'grid'
	},
	{
		id: 'stealth-pouch',
		name: 'the stealth pouch',
		desc: 'Matte, smell-tight, zero branding. Holds the kit and looks like nothing at all.',
		visual: 'pouch'
	}
];
