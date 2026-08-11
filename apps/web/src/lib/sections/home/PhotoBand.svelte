<!-- The altarpiece: four side panels flanking a taller centre one, each picture
     set in its own niche behind Wii-lacquered glass. The arches are all
     different — round, dome, squared, ogee — so the row is freeform up close
     and perfectly ordered from across the street. One accent colour only.
     Every niche is a doorway into a section; every caption says what it is. -->
<script lang="ts">
	const tiles: {
		src: string;
		alt: string;
		caption: string;
		href: string;
		arch: string;
		centre?: boolean;
	}[] = [
		{
			src: '/images/thandai.jpg',
			alt: 'Steel cups of bhang thandai raised together at Holi',
			caption: 'thandai at holi',
			href: '/science#roa',
			// a plain round arch
			arch: '46% 46% 5px 5px / 30% 30% 5px 5px'
		},
		{
			src: '/images/sadhus.jpg',
			alt: 'Three sadhus seated on temple steps',
			caption: 'sadhus',
			href: '/history#courts',
			// squared off, barely lifted at the corners
			arch: '14% 14% 5px 5px / 9% 9% 5px 5px'
		},
		{
			src: '/images/shiva-bhang.jpg',
			alt: 'Miniature painting of Shiva grinding bhang in a mortar',
			caption: 'shiva grinding bhang',
			href: '/history#roots',
			// the shikhara: a tall dome over the centre panel
			arch: '50% 50% 6px 6px / 44% 44% 6px 6px',
			centre: true
		},
		{
			src: '/images/shrine.jpg',
			alt: 'Man and two dogs asleep beneath a roadside Shiva shrine',
			caption: 'a roadside shrine',
			href: '/history#tools',
			// a high, narrow arch
			arch: '44% 44% 5px 5px / 40% 40% 5px 5px'
		},
		{
			src: '/images/shiva-eyes.jpg',
			alt: "Pastel drawing of Shiva's half-closed eyes and third eye",
			caption: 'the third eye',
			href: '/parcha',
			// almost a square — the odd one out at the end of the row
			arch: '8% 8% 5px 5px / 6% 6% 5px 5px'
		}
	];
</script>

<div class="band">
	{#each tiles as t (t.src)}
		<a class="niche" class:centre={t.centre} href={t.href} style="--arch: {t.arch}">
			<span class="frame">
				<img src={t.src} alt={t.alt} loading="lazy" decoding="async" />
			</span>
			<span class="caption">{t.caption}</span>
		</a>
	{/each}
</div>

<style>
	.band {
		position: relative;
		z-index: 1;
		display: grid;
		/* the altarpiece: 1 · 1 · centre · 1 · 1, standing on one baseline */
		grid-template-columns: repeat(2, minmax(0, 1fr)) minmax(0, 1.4fr) repeat(2, minmax(0, 1fr));
		align-items: end;
		gap: 0.8rem;
	}

	.niche {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
		min-width: 0;
	}

	.frame {
		position: relative;
		display: block;
		aspect-ratio: 3 / 4;
		padding: 4px;
		border-radius: var(--arch);
		background-color: #fff;
		background-image: var(--wii-face);
		/* one accent: a green hairline, a white bezel, then the drop */
		box-shadow:
			0 0 0 1.25px color-mix(in srgb, var(--mood-green) 50%, #fff),
			0 0 0 3px rgba(255, 255, 255, 0.92),
			var(--aqua-drop),
			inset 0 1px 0 rgba(255, 255, 255, 0.95);
		transition:
			transform 0.24s cubic-bezier(0.22, 1, 0.36, 1),
			box-shadow 0.24s ease;
	}
	.centre .frame {
		aspect-ratio: 4 / 5;
	}
	.frame img {
		width: 100%;
		height: 100%;
		display: block;
		object-fit: cover;
		border-radius: inherit;
		box-shadow: inset 0 1px 4px rgba(120, 60, 140, 0.45);
	}
	/* glass: a gleam running over the crown of the arch */
	.frame::after {
		content: '';
		position: absolute;
		inset: 2px 2px 48% 2px;
		border-radius: inherit;
		background: var(--wii-gleam);
		opacity: 0.5;
		pointer-events: none;
	}
	.niche:hover .frame,
	.niche:focus-visible .frame {
		transform: translateY(-4px);
		box-shadow:
			0 0 0 1.25px var(--mood-green),
			0 0 0 3px rgba(255, 255, 255, 0.96),
			var(--aqua-drop-hover),
			inset 0 1px 0 rgba(255, 255, 255, 0.95);
	}

	.caption {
		font-family: var(--font-pixel);
		font-size: 0.74rem;
		letter-spacing: 0.07em;
		line-height: 1.25;
		text-align: center;
		text-transform: uppercase;
		color: var(--mute);
		text-shadow: 0 1px 0 rgba(255, 255, 255, 0.9);
		transition: color 0.2s ease;
	}
	.centre .caption {
		font-size: 0.82rem;
		color: color-mix(in srgb, var(--mood-green) 82%, #000);
	}
	.niche:hover .caption,
	.niche:focus-visible .caption {
		color: color-mix(in srgb, var(--mood-green) 82%, #000);
	}

	@media (max-width: 820px) {
		.band {
			/* the shrine folds: centre panel leads, the rest sit under it */
			grid-template-columns: repeat(4, minmax(0, 1fr));
		}
		.centre {
			grid-column: 1 / -1;
		}
		.centre .frame {
			aspect-ratio: 16 / 9;
			border-radius: 22% 22% 6px 6px / 44% 44% 6px 6px;
		}
	}
	@media (max-width: 520px) {
		.band {
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 0.6rem;
		}
		.centre .frame {
			aspect-ratio: 4 / 3;
		}
	}
</style>
