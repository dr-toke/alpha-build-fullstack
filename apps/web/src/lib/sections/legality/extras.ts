// Supplementary legality content: timeline, penalties, FAQ, glossary.
// Skeleton copy — placeholder-grade where marked; verify everything marked
// "verify" with counsel before launch. Lorem-ish filler is intentional.

export interface TimelineEntry {
	year: string;
	text: string;
}

export const timeline: TimelineEntry[] = [
	{ year: '~1500 BCE', text: 'Atharvaveda names cannabis among the five sacred plants.' },
	{ year: '1798', text: 'British colonial administration begins taxing ganja rather than banning it.' },
	{ year: '1894', text: 'Indian Hemp Drugs Commission: moderate use "practically no injurious effects" — recommends regulation, not prohibition.' },
	{ year: '1961', text: 'UN Single Convention classes cannabis with hard drugs; India negotiates a 25-year window for traditional use.' },
	{ year: '1985', text: 'NDPS Act passed — flower (ganja) and resin (charas) banned; leaf (bhang) left to state excise.' },
	{ year: '2017', text: 'First Ayush-licensed cannabis medicine ventures appear; hemp food category begins to formalise.' },
	{ year: '2020', text: 'UN votes to remove cannabis from Schedule IV; India votes in favour.' },
	{ year: 'today', text: 'Legal hemp/CBD market grows under Ayush + FSSAI frameworks — the market this site catalogues.' }
];

export interface PenaltyRow {
	offence: string;
	quantity: string;
	punishment: string;
}

// NDPS ganja thresholds — VERIFY with counsel; figures are the commonly cited ones.
export const penalties: PenaltyRow[] = [
	{
		offence: 'Possession (small quantity)',
		quantity: 'up to 1 kg ganja',
		punishment: 'up to 1 year rigorous imprisonment and/or fine up to ₹10,000'
	},
	{
		offence: 'Possession (intermediate)',
		quantity: '1 kg – 20 kg ganja',
		punishment: 'up to 10 years + fine up to ₹1,00,000'
	},
	{
		offence: 'Possession (commercial quantity)',
		quantity: '20 kg+ ganja',
		punishment: '10–20 years + fine ₹1,00,000–₹2,00,000'
	},
	{
		offence: 'Bhang (leaf) possession/sale',
		quantity: 'per state excise rules',
		punishment: 'outside NDPS — governed by state excise acts; varies from licensed to prohibited'
	},
	{
		offence: 'Licensed Ayush/FSSAI hemp products',
		quantity: 'compliant products',
		punishment: 'legal to buy and possess — keep invoice and COA'
	}
];

export interface Faq {
	q: string;
	a: string;
}

export const faqs: Faq[] = [
	{
		q: 'Is bhang actually legal?',
		a: 'The NDPS Act’s cannabis definition covers flower and resin, not the leaf — bhang is regulated by each state’s excise department instead. Licensed shops exist in some states; in others it’s tolerated seasonally or banned. Check your state in the grid below.'
	},
	{
		q: 'Can I legally order CBD oil online in India?',
		a: 'Ayush-licensed and hemp-derived products within THC limits are sold openly by the brands this site catalogues. Some products are prescription-gated (marked Rx in the catalog). Lorem ipsum caveat block — detail per product category to be written.'
	},
	{
		q: 'Is growing a plant at home legal?',
		a: 'Cultivation of cannabis for flower/resin is an offence under the NDPS Act, with limited state-licensed exceptions for hemp cultivation (fibre/seed). Placeholder — expand with state licensing detail.'
	},
	{
		q: 'What is the difference between hemp and marijuana here?',
		a: 'Legally, it’s about THC content and plant part, not botany. Low-THC hemp (fibre, seed, leaf-derived) products under the permitted limits live in the legal market; flower and resin do not. Lorem ipsum — precise thresholds to be confirmed.'
	},
	{
		q: 'What should I do if stopped by police with a legal product?',
		a: 'Carry the invoice and the product’s COA/licence details; stay calm and cooperative. This is general information, not legal advice — a proper know-your-rights explainer is planned with counsel review.'
	}
];

// ── The Archives — statutes, reports, judgments, international instruments,
//    and press. Everything the law says, said, and got told. Entries without
//    a URL are placeholders awaiting sourcing (lorem-grade, marked "verify").
export type ArchiveKind = 'statute' | 'report' | 'judgment' | 'international' | 'news';

export interface ArchiveEntry {
	kind: ArchiveKind;
	year: string;
	title: string;
	source: string;
	url?: string;
	note: string;
}

export const archives: ArchiveEntry[] = [
	{
		kind: 'statute',
		year: '1985',
		title: 'NDPS Act, 1985 — full text',
		source: 'India Code',
		url: 'https://www.indiacode.nic.in/handle/123456789/1791',
		note: 'The central law. §2(iii) defines "cannabis" as flower and resin — the leaf is excluded.'
	},
	{
		kind: 'statute',
		year: '1985',
		title: 'NDPS Rules & quantity notifications',
		source: 'Dept. of Revenue',
		note: 'Small (1 kg) vs commercial (20 kg) ganja thresholds live here. Verify current notification.'
	},
	{
		kind: 'statute',
		year: 'various',
		title: 'State excise acts (bhang licensing)',
		source: 'state gazettes',
		note: 'One per state — Rajasthan, UP, MP licensing frameworks first. To be collected; lorem for now.'
	},
	{
		kind: 'report',
		year: '1894',
		title: 'Indian Hemp Drugs Commission Report',
		source: 'public domain',
		note: 'Seven volumes, 1,193 witnesses: "moderate use produces practically no injurious effects." The founding document of this whole argument.'
	},
	{
		kind: 'report',
		year: '2021',
		title: 'FSSAI — hemp seed & seed products regulation',
		source: 'fssai.gov.in',
		url: 'https://www.fssai.gov.in',
		note: 'The rule that put hemp foods on legal shelves. Pin the exact gazette PDF.'
	},
	{
		kind: 'report',
		year: 'ongoing',
		title: 'Ayush licensing framework for cannabis medicine',
		source: 'ayush.gov.in',
		url: 'https://ayush.gov.in',
		note: 'How vijaya-based medicine is licensed and why Rx products exist in the catalog.'
	},
	{
		kind: 'international',
		year: '1961',
		title: 'UN Single Convention on Narcotic Drugs',
		source: 'UNODC',
		url: 'https://www.unodc.org/unodc/en/treaties/single-convention.html',
		note: 'The treaty that pushed the ban. India negotiated a 25-year window for traditional use.'
	},
	{
		kind: 'international',
		year: '2020',
		title: 'CND vote removing cannabis from Schedule IV',
		source: 'UN CND',
		note: 'India voted in favour. The international tide turning, on the record.'
	},
	{
		kind: 'judgment',
		year: '——',
		title: 'Leaf vs flower: the bhang exclusion cases',
		source: 'High Courts',
		note: 'Placeholder — collect the HC rulings holding that seeds and leaves fall outside NDPS "cannabis". Verify citations with counsel. Lorem ipsum dolor sit amet.'
	},
	{
		kind: 'judgment',
		year: '——',
		title: 'Quantity & bail jurisprudence',
		source: 'Supreme Court / HCs',
		note: 'Placeholder — the small-vs-commercial quantity rulings that decide real outcomes. Lorem ipsum.'
	},
	{
		kind: 'judgment',
		year: '——',
		title: 'CBD / hemp product seizure cases',
		source: 'various',
		note: 'Placeholder — documented cases involving licensed products, and how they resolved. Lorem ipsum.'
	},
	{
		kind: 'news',
		year: '2019',
		title: 'Delhi HC PIL seeking cannabis re-legalisation',
		source: 'press archive',
		note: 'Placeholder summary — track the petition trail. Lorem ipsum dolor sit.'
	},
	{
		kind: 'news',
		year: '2023',
		title: 'Himachal legalises cultivation study committee',
		source: 'press archive',
		note: 'Placeholder — state-level pilot movements worth archiving. Lorem ipsum.'
	},
	{
		kind: 'news',
		year: 'rolling',
		title: 'The clippings file',
		source: 'everything else',
		note: 'Raids, rulings, policy noise — dated, sourced, archived. This is the living part of the archive.'
	}
];

export interface GlossaryTerm {
	term: string;
	def: string;
}

export const glossary: GlossaryTerm[] = [
	{ term: 'bhang', def: 'preparation of cannabis leaf — outside the NDPS definition of "cannabis", state-excise regulated' },
	{ term: 'ganja', def: 'the flowering tops — banned under NDPS since 1985' },
	{ term: 'charas', def: 'the resin (incl. hash) — banned under NDPS, highest penalties' },
	{ term: 'vijaya', def: 'the Ayurvedic name for cannabis; used for Ayush-licensed medicine' },
	{ term: 'NDPS', def: 'Narcotic Drugs and Psychotropic Substances Act, 1985 — the central law' },
	{ term: 'Ayush', def: 'ministry governing Ayurvedic medicine licensing, incl. cannabis medicine' },
	{ term: 'FSSAI', def: 'food regulator — hemp seed foods are licensed under it' },
	{ term: 'COA', def: 'Certificate of Analysis — third-party lab report behind an honest product' },
	{ term: 'small quantity', def: 'NDPS threshold that keeps a possession offence bailable and lightly punished (ganja: 1 kg)' },
	{ term: 'commercial quantity', def: 'NDPS threshold that triggers the harshest penalties (ganja: 20 kg)' }
];
