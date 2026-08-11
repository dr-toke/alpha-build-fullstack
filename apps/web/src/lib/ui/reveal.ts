// Scroll-reveal for `.reveal` elements — port of apps/web/lib/reveal.ts.
// Watches the DOM for nodes added after mount (async product/state cards),
// and degrades safely: reduced-motion or no IntersectionObserver → show all,
// plus a 1.5s failsafe so nothing can stay hidden forever.
//
// Call from the root layout inside $effect; returns a cleanup function.

export function initReveal(): () => void {
	const revealAllNow = () => {
		document
			.querySelectorAll<HTMLElement>('.reveal:not(.visible)')
			.forEach((el) => el.classList.add('visible'));
	};

	const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;

	if (reducedMotion || typeof IntersectionObserver === 'undefined') {
		revealAllNow();
		return () => {};
	}

	const io = new IntersectionObserver(
		(entries) => {
			entries.forEach((e) => {
				if (e.isIntersecting) {
					e.target.classList.add('visible');
					io.unobserve(e.target);
				}
			});
		},
		{ threshold: 0.12, rootMargin: '0px 0px -40px 0px' }
	);

	const observeNew = () => {
		document
			.querySelectorAll<HTMLElement>('.reveal:not(.visible)')
			.forEach((el) => io.observe(el)); // re-observing is idempotent
	};

	observeNew();

	// Catch elements added later (async fetches). Debounced via rAF so a burst
	// of inserted cards triggers a single rescan.
	let scheduled = false;
	const mo = new MutationObserver(() => {
		if (scheduled) return;
		scheduled = true;
		requestAnimationFrame(() => {
			scheduled = false;
			observeNew();
		});
	});
	mo.observe(document.body, { childList: true, subtree: true });

	const failsafe = window.setTimeout(revealAllNow, 1500);

	return () => {
		io.disconnect();
		mo.disconnect();
		window.clearTimeout(failsafe);
	};
}
