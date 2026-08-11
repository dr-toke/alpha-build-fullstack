<script lang="ts">
	// Exact port of ../dr-toke/apps/web/components/ui/Tag.tsx.
	import type { Snippet } from 'svelte';

	type TagVariant = 'green' | 'gold' | 'red' | 'muted';

	let {
		variant = 'green',
		class: className = '',
		children
	}: { variant?: TagVariant; class?: string; children: Snippet } = $props();

	const variants: Record<TagVariant, string> = {
		green: 'bg-[rgba(58,122,47,0.2)] border border-[rgba(58,122,47,0.5)] text-green2',
		gold: 'bg-[rgba(200,168,75,0.12)] border border-[rgba(200,168,75,0.4)] text-gold2',
		red: 'bg-[rgba(196,92,58,0.15)] border border-[rgba(196,92,58,0.4)] text-red2',
		muted: 'bg-[rgba(184,160,128,0.12)] border border-[rgba(184,160,128,0.3)] text-cream2'
	};
</script>

<span
	class={`inline-block text-[0.6rem] font-medium px-[0.48rem] py-[0.15rem] rounded-[2px] tracking-[0.08em] uppercase ${variants[variant]} ${className}`}
>
	{@render children()}
</span>
