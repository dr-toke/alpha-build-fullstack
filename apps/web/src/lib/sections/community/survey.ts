// Survey form definition — copy of ../dr-toke/packages/data/index.ts.
// Coupled to the backend survey table columns; option ORDER matters (the API
// stores answers as option indices), so never reorder without a migration.

export const SURVEY_STEPS = [
	{
		key: 'extractType' as const,
		question: 'What type of extract interests you most?',
		options: [
			'Full Spectrum (all cannabinoids + terpenes, trace THC)',
			'Broad Spectrum (CBD + terpenes, zero THC)',
			'CBD Isolate (pure CBD, nothing else)',
			'High-THC Vijaya (prescription Ayush formulation)'
		]
	},
	{
		key: 'useCase' as const,
		question: 'Primary use case?',
		options: [
			'Pain management (chronic/acute)',
			'Sleep & insomnia',
			'Anxiety & stress relief',
			'General wellness',
			'Skin conditions',
			'Fitness & recovery'
		]
	},
	{
		key: 'priceRange' as const,
		question: 'Budget for a 30ml bottle?',
		options: ['Under ₹500', '₹500 – ₹1,500', '₹1,500 – ₹3,000', '₹3,000+ (premium)']
	},
	{
		key: 'carrierOil' as const,
		question: 'Preferred carrier oil?',
		options: [
			'Hemp seed oil',
			'MCT / Coconut oil',
			'Sesame oil (traditional Ayurvedic)',
			'Olive oil',
			'No preference'
		]
	}
] as const;

export type SurveyKey = (typeof SURVEY_STEPS)[number]['key'];
