// History section content — one era per entry. This is the writing surface:
// replace/extend `body`, `points`, and clear each `todo` as the copy lands.

export interface Era {
	id: string;
	years: string;
	title: string;
	body: string;
	points: string[];
	todo?: string;
}

export const eras: Era[] = [
	{
		id: 'roots',
		years: '~1500 BCE — 1500s',
		title: 'Roots',
		body: 'Cannabis enters the Indian record in the Atharvaveda, named among five sacred plants — a reliever of anxiety. For most of recorded history here it was not a vice but a material: medicine in Ayurvedic preparations, sacrament in Shaivite practice, and bhang in the festival cup.',
		points: [
			'Bhang, ganja, charas — three preparations, three very different social meanings',
			'Ayurvedic use: digestion, sleep, pain — always dosed, always prepared',
			'Shivratri and Holi thandai as the oldest continuous use tradition'
		],
		todo: 'regional traditions, primary sources, dates worth quoting'
	},
	{
		id: 'courts',
		years: '1200s — 1700s',
		title: 'Courts & mystics',
		body: 'Emperors and ascetics consumed the same plant in opposite registers: majoun (a cannabis edible) in refined court kitchens, and the chillum passed around sadhu fires. Folk medicine used leaf poultices and decoctions across the subcontinent.',
		points: [
			'Majoun — the edible of the courts',
			'Sadhus, chillums, and renunciate culture',
			'Unani and folk-medicine preparations'
		],
		todo: 'names, anecdotes, and sources — who consumed what, and how'
	},
	{
		id: 'colonial',
		years: '1798 — 1894',
		title: 'The colonial audit',
		body: 'The British taxed ganja rather than banning it, then commissioned the deepest study of cannabis ever done: the Indian Hemp Drugs Commission of 1894. Across seven volumes it concluded that moderate use produced "practically no injurious effects" — and recommended regulation, not prohibition.',
		points: [
			'1798: colonial taxation of ganja begins',
			'1894: Indian Hemp Drugs Commission — 3,000+ pages, 1,193 witnesses',
			'Its verdict: prohibition unjustified; tax and regulate instead'
		],
		todo: 'pull direct quotes from the Commission report (public domain)'
	},
	{
		id: 'ban',
		years: '1961 — 1985',
		title: 'The ban',
		body: 'The 1961 UN Single Convention classed cannabis with the hardest drugs. India resisted, won a 25-year grace period for traditional use, and finally passed the NDPS Act in 1985 — banning flower and resin, while carving the leaf (bhang) out to state excise control. The ban arrived by treaty pressure, not by Indian evidence.',
		points: [
			'1961: Single Convention on Narcotic Drugs — India holds out',
			'1985: NDPS Act — ganja and charas banned, bhang exempted nationally',
			'The leaf/flower split still defines Indian cannabis law today'
		],
		todo: 'the geopolitics — US pressure, the 25-year clause, parliamentary debate'
	},
	{
		id: 'world',
		years: '1976 — now',
		title: 'How the world moved on',
		body: 'While India froze its law in 1985, other countries ran the experiment: Dutch tolerance, American state-by-state legalisation, Canadian federal legality, Thai decriminalisation and partial retreat, German reform. Each is a different answer to the same question the 1894 Commission already asked.',
		points: [
			'1976 Netherlands · 2012 Colorado & Washington · 2013 Uruguay',
			'2018 Canada · 2020 UN reschedules cannabis · 2022 Thailand · 2024 Germany',
			'What each model got right and wrong — the honest comparison'
		],
		todo: 'one paragraph per country model, with what India can actually learn'
	},
	{
		id: 'tools',
		years: 'timeless',
		title: 'Tools & rituals',
		body: 'The instruments are part of the history: the clay chillum and its etiquette, the hookah, the mortar-and-pestle thandai of festival kitchens, majoun sweets. Documented here as culture and heritage — how people actually consumed, and what each method meant.',
		points: [
			'Chillum — the renunciate’s pipe and its shared-fire etiquette',
			'Thandai — bhang’s festival form, ground into milk, spice, and almonds',
			'Majoun — the edible tradition centuries before the word "edible"'
		],
		todo: 'illustrations/photography plan; keep framing cultural + harm-aware'
	}
];
