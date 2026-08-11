<!-- Section icon in the Wii channel language: a white lacquered disc, ringed in
     the section's colour, with a grey-accent glyph and a specular gleam across
     the top half. Tokens: --wii-face / --wii-gleam / --aqua-* in app.css. -->
<script lang="ts">
	type Glyph = 'history' | 'science' | 'legality' | 'products';

	let {
		name,
		color = 'var(--mood-purple-deep)',
		size = '2.3rem'
	}: { name: Glyph; color?: string; size?: string } = $props();
</script>

<span class="disc" style="--c: {color}; --s: {size}" aria-hidden="true">
	<!-- stroke/fill come from CSS below: color-mix() is unreliable in SVG
	     presentation attributes, so the fills use currentColor. -->
	<svg viewBox="0 0 24 24" fill="none" stroke-width="1.9" stroke-linecap="round"
		stroke-linejoin="round">
		{#if name === 'history'}
			<!-- hourglass: the sand has already run -->
			<path d="M7 3.5h10M7 20.5h10" />
			<path d="M8 3.5h8L12 11z" fill="currentColor" fill-opacity="0.45" stroke="none" />
			<path d="M8 20.5h8L12 13z" fill="currentColor" stroke="none" />
			<path d="M8 3.5c0 4 8 5.5 8 8.5s-8 4.5-8 8.5" opacity="0.6" />
		{:else if name === 'science'}
			<!-- conical flask, half full -->
			<path d="M9.5 3h5M10.5 3v5L6 17.4A1.9 1.9 0 0 0 7.7 20.5h8.6a1.9 1.9 0 0 0 1.7-3.1L13.5 8V3" />
			<path
				d="M8.15 14.6h7.7l2.15 4.3a1.4 1.4 0 0 1-1.25 1.6H7.25A1.4 1.4 0 0 1 6 18.9z"
				fill="currentColor" fill-opacity="0.5"
				stroke="none"
			/>
			<path d="M8.2 14.6h7.6" opacity="0.75" />
		{:else if name === 'legality'}
			<!-- scales, level -->
			<path d="M12 4.5v15M8 19.5h8M5 8.5h14" />
			<path d="M2.6 8.5 5 13.6l2.4-5.1M16.6 8.5 19 13.6l2.4-5.1" />
			<circle cx="12" cy="6" r="1.1" fill="currentColor" stroke="none" />
		{:else}
			<!-- price tag -->
			<path d="M3.5 4.2h8.2l8.8 8.8-7.5 7.5-9.5-9.5z" />
			<circle cx="7.6" cy="8.3" r="1.35" fill="currentColor" stroke="none" />
		{/if}
	</svg>
</span>

<style>
	.disc {
		position: relative;
		flex-shrink: 0;
		display: grid;
		place-items: center;
		width: var(--s);
		height: var(--s);
		border-radius: 50%;
		background-color: #fff;
		background-image: var(--wii-face);
		/* the ring: a hairline of the colour, then a white bezel, then the drop */
		box-shadow:
			0 0 0 1.5px color-mix(in srgb, var(--c) 60%, #fff),
			0 0 0 3px rgba(255, 255, 255, 0.95),
			var(--aqua-drop),
			inset 0 1px 0 rgba(255, 255, 255, 0.95),
			inset 0 -4px 6px -4px rgba(120, 60, 140, 0.28);
	}
	/* the gleam on the domed upper half */
	.disc::after {
		content: '';
		position: absolute;
		inset: 5% 14% auto;
		height: 42%;
		border-radius: 50%;
		background: var(--wii-gleam);
		pointer-events: none;
	}
	svg {
		width: 60%;
		height: 60%;
		/* the engraved glyph: a dark tint of the section colour */
		stroke: color-mix(in srgb, var(--c) 74%, #2a2030);
		color: color-mix(in srgb, var(--c) 58%, #fff);
		filter: drop-shadow(0 1px 0 rgba(255, 255, 255, 0.9));
	}
</style>
