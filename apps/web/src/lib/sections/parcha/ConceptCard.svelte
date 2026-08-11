<script lang="ts">
	import type { Concept } from './content';

	let { concept }: { concept: Concept } = $props();
</script>

<article class="concept">
	<div class="visual" aria-hidden="true">
		{#if concept.visual === 'cig'}
			<div class="v-cig">
				<div class="cig-body"></div>
				<div class="cig-tip"></div>
			</div>
		{:else if concept.visual === 'paper'}
			<div class="v-paper"></div>
		{:else if concept.visual === 'grid'}
			<div class="v-grid">
				{#each Array(12) as _, i (i)}
					<div class="cell"></div>
				{/each}
			</div>
		{:else if concept.visual === 'pouch'}
			<div class="v-pouch">
				<div class="flap"></div>
				<div class="seal"></div>
			</div>
		{/if}
	</div>

	<h3>{concept.name}</h3>
	<p class="desc">{concept.desc}</p>
	<p class="status">concept · not for sale (yet)</p>
</article>

<style>
	.concept {
		border: 1px solid var(--ink);
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.55rem;
	}
	.visual {
		height: 170px;
		display: flex;
		align-items: center;
		justify-content: center;
		border: 1px dashed var(--line);
		background: var(--paper);
	}

	/* ── the cigarette roach ── */
	.v-cig {
		display: flex;
		width: 170px;
		height: 30px;
		border-radius: 15px;
		overflow: hidden;
		border: 1px solid #cfcfcf;
		box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
	}
	.cig-body {
		flex: 1;
		background: #ffffff;
	}
	.cig-tip {
		width: 36%;
		background: #e08a3c;
		background-image:
			radial-gradient(rgba(120, 62, 12, 0.35) 1px, transparent 1px),
			radial-gradient(rgba(255, 214, 160, 0.5) 1px, transparent 1px);
		background-size: 5px 5px, 7px 7px;
		background-position: 0 0, 3px 2px;
		border-left: 1px solid #c97a2f;
	}

	/* ── full white papers ── */
	.v-paper {
		width: 110px;
		height: 140px;
		background: #ffffff;
		border: 1px solid #dcdcdc;
		box-shadow: 0 3px 10px rgba(0, 0, 0, 0.07);
		position: relative;
	}
	.v-paper::after {
		content: '';
		position: absolute;
		top: 0;
		right: 0;
		width: 0;
		height: 0;
		border-left: 22px solid #efefef;
		border-top: 22px solid var(--paper);
	}

	/* ── the grid sheet ── */
	.v-grid {
		width: 160px;
		height: 120px;
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		grid-template-rows: repeat(3, 1fr);
		border: 1px solid #cfcfcf;
		background: #ffffff;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
	}
	.cell {
		border: 1px dashed #c4c4c4;
		margin: -0.5px;
	}

	/* ── the stealth pouch ── */
	.v-pouch {
		width: 110px;
		height: 140px;
		background: #232323;
		border-radius: 8px 8px 14px 14px;
		position: relative;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.18);
	}
	.flap {
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		height: 34px;
		background: #1a1a1a;
		border-radius: 8px 8px 0 0;
		border-bottom: 1px dashed #3d3d3d;
	}
	.seal {
		position: absolute;
		top: 26px;
		left: 50%;
		transform: translateX(-50%);
		width: 26px;
		height: 9px;
		background: #3d3d3d;
		border-radius: 2px;
	}

	h3 {
		font-family: var(--font-pixel);
		font-size: 1.3rem;
		line-height: 1.1;
	}
	.desc {
		font-size: 0.9rem;
		line-height: 1.6;
		color: var(--ink);
		flex-grow: 1;
	}
	.status {
		font-family: var(--font-pixel);
		font-size: 0.8rem;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: var(--mute);
		border-top: 1px dashed var(--line);
		padding-top: 0.5rem;
	}
</style>
