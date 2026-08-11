<!-- The art layer: hairline marks scattered across the page and left to drift —
     concentric rings, a freeform lobe, a halftone field, an open eye, a spiral,
     a crescent with something in orbit, and a few technical crosses. All in the
     one accent, all faint, none of it clickable. Nothing here carries meaning;
     it is the surreal ground the rigid index stands on. -->
<div class="art" aria-hidden="true">
	<!-- concentric rings, upper left -->
	<svg class="m rings" viewBox="0 0 200 200">
		<g fill="none" stroke="currentColor" stroke-width="1.4">
			<circle cx="100" cy="100" r="94" />
			<circle cx="100" cy="100" r="72" opacity="0.7" />
			<circle cx="100" cy="100" r="48" opacity="0.5" />
			<circle cx="100" cy="100" r="21" opacity="0.85" />
			<circle cx="100" cy="100" r="5" fill="currentColor" stroke="none" />
		</g>
	</svg>

	<!-- a freeform lobe, upper right -->
	<svg class="m lobe" viewBox="0 0 300 300">
		<g fill="none" stroke="currentColor" stroke-width="1.5">
			<path d="M64 118c34-72 148-84 186-16s-10 152-78 172-142-14-150-72 12-56 42-84z" />
			<path d="M92 132c26-52 108-62 136-12s-8 112-58 126-104-10-110-52 8-42 32-62z" opacity="0.6" />
			<path d="M120 148c18-34 72-40 90-8s-6 74-38 84-70-8-74-36 6-28 22-40z" opacity="0.35" />
		</g>
	</svg>

	<!-- a halftone field, mid left -->
	<svg class="m dots" viewBox="0 0 120 160">
		<defs>
			<pattern id="ht" width="12" height="12" patternUnits="userSpaceOnUse">
				<circle cx="3" cy="3" r="1.9" fill="currentColor" />
			</pattern>
		</defs>
		<rect width="120" height="160" fill="url(%23ht)" />
	</svg>

	<!-- the open eye, mid right — the one figurative mark -->
	<svg class="m eye" viewBox="0 0 240 140">
		<g fill="none" stroke="currentColor" stroke-width="1.6">
			<path d="M12 70C56 20 184 20 228 70 184 120 56 120 12 70z" />
			<circle cx="120" cy="70" r="30" />
			<circle cx="120" cy="70" r="11" fill="currentColor" stroke="none" />
			<!-- lashes as rays: holy, not anatomical -->
			<path d="M120 24V4M120 116v20M60 40 46 24M180 40l14-16M60 100 46 116M180 100l14 16" opacity="0.7" />
		</g>
	</svg>

	<!-- a spiral, lower left -->
	<svg class="m spiral" viewBox="0 0 200 200">
		<path
			fill="none"
			stroke="currentColor"
			stroke-width="1.5"
			d="M100 100c0-10 12-14 20-8s8 26-6 34-42 2-52-16 2-52 26-64 66-8 86 20 14 74-14 96"
		/>
	</svg>

	<!-- a crescent with something in orbit, lower right -->
	<svg class="m moon" viewBox="0 0 180 180">
		<g fill="none" stroke="currentColor" stroke-width="1.5">
			<path d="M112 22a70 70 0 1 0 0 136 84 84 0 0 1 0-136z" />
			<ellipse cx="90" cy="90" rx="84" ry="34" opacity="0.5" transform="rotate(-18 90 90)" />
			<circle class="bead" cx="170" cy="64" r="5" fill="currentColor" stroke="none" />
		</g>
	</svg>

	<!-- technical crosses, scattered -->
	<svg class="m crosses" viewBox="0 0 400 300">
		<g stroke="currentColor" stroke-width="1.5">
			<path d="M28 44h16M36 36v16M212 18h14M219 11v14M356 96h14M363 89v14" />
			<path d="M84 232h14M91 225v14M268 268h14M275 261v14M330 186h14M337 179v14" />
		</g>
	</svg>
</div>

<style>
	.art {
		position: fixed;
		inset: 0;
		z-index: 0;
		color: var(--mood-green);
		overflow: hidden;
		pointer-events: none;
	}
	.m {
		position: absolute;
		display: block;
	}

	.rings {
		top: -6rem;
		left: -8rem;
		width: 30rem;
		opacity: 0.13;
		animation: turn 300s linear infinite;
	}
	.lobe {
		top: 3rem;
		right: -9rem;
		width: 34rem;
		opacity: 0.12;
		animation: breathe 76s ease-in-out infinite;
	}
	.dots {
		top: 46%;
		left: 1.5rem;
		width: 7rem;
		opacity: 0.14;
		animation: sway 58s ease-in-out infinite;
	}
	.eye {
		top: 34%;
		right: 2rem;
		width: 13rem;
		opacity: 0.14;
		animation: gaze 44s ease-in-out infinite;
	}
	.spiral {
		bottom: -5rem;
		left: 6vw;
		width: 22rem;
		opacity: 0.11;
		animation: turn 220s linear infinite reverse;
	}
	.moon {
		bottom: 4rem;
		right: 7vw;
		width: 15rem;
		opacity: 0.12;
		animation: breathe 92s ease-in-out infinite;
	}
	.crosses {
		inset: 12% 8% auto;
		width: 84%;
		opacity: 0.16;
	}
	/* the bead runs its own slow lap around the crescent */
	.bead {
		transform-origin: 90px 90px;
		animation: turn 120s linear infinite;
	}

	@keyframes turn {
		to {
			transform: rotate(360deg);
		}
	}
	@keyframes breathe {
		50% {
			transform: rotate(7deg) scale(1.07);
		}
	}
	@keyframes sway {
		50% {
			transform: translate(1.4rem, -2.2rem);
		}
	}
	/* the eye drifts and narrows, as if half-closing */
	@keyframes gaze {
		35% {
			transform: translate(-1rem, 1.4rem) scaleY(0.92);
		}
		70% {
			transform: translate(0.6rem, -0.8rem) scaleY(1);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.m,
		.bead {
			animation: none;
		}
	}
	/* on a phone the art gets in the way of the reading — keep only the edges */
	@media (max-width: 700px) {
		.dots,
		.eye,
		.crosses {
			display: none;
		}
		.rings,
		.lobe,
		.spiral,
		.moon {
			opacity: 0.09;
		}
	}
</style>
