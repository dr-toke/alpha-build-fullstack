<!-- The dr. toke logotype (static/images/wordmark.png — the real mark, green
     #52AD85 on paper) standing in a slow sunburst halo. Holy iconography,
     Aphex-flat execution: the rays turn, nothing else moves. -->
<div class="mark">
	<span class="halo" aria-hidden="true"></span>
	<h1>
		<img src="/images/wordmark.png" alt="dr. toke" width="555" height="54" />
	</h1>
</div>

<style>
	.mark {
		position: relative;
		display: flex;
		justify-content: center;
		padding: 2.2rem 0 0.4rem;
	}
	/* the sunburst: hard rays, faded out to nothing at the rim */
	.halo {
		position: absolute;
		top: 50%;
		left: 50%;
		width: min(46rem, 108%);
		aspect-ratio: 1;
		transform: translate(-50%, -50%);
		background: var(--halo-rays);
		mask-image: radial-gradient(
			circle,
			rgba(0, 0, 0, 0) 12%,
			rgba(0, 0, 0, 0.55) 26%,
			rgba(0, 0, 0, 0.18) 46%,
			rgba(0, 0, 0, 0) 62%
		);
		animation: turn 240s linear infinite;
		pointer-events: none;
		z-index: 0;
	}
	/* a soft nimbus sitting under the letters */
	.mark::after {
		content: '';
		position: absolute;
		top: 50%;
		left: 50%;
		width: min(30rem, 92%);
		aspect-ratio: 2 / 1;
		transform: translate(-50%, -50%);
		background: radial-gradient(
			ellipse,
			rgba(255, 255, 255, 0.9) 0%,
			rgba(255, 255, 255, 0.5) 45%,
			rgba(255, 255, 255, 0) 72%
		);
		pointer-events: none;
		z-index: 0;
	}
	@keyframes turn {
		to {
			transform: translate(-50%, -50%) rotate(360deg);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.halo {
			animation: none;
		}
	}

	h1 {
		position: relative;
		z-index: 1;
		line-height: 0;
	}
	img {
		display: block;
		width: clamp(15rem, 34vw, 23rem);
		height: auto;
		/* the mark lifts off the paper just slightly */
		filter: drop-shadow(0 1px 0 rgba(255, 255, 255, 0.95))
			drop-shadow(0 3px 7px rgba(120, 60, 140, 0.18));
	}
</style>
