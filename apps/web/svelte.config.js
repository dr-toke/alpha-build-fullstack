import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		// Fully static output, same philosophy as dr-toke's clearnet tier:
		// the site must be servable from any dumb file host (Vercel, nginx, Tor).
		adapter: adapter({ pages: 'build', assets: 'build', strict: true }),
		alias: {
			$sections: 'src/lib/sections'
		}
	}
};

export default config;
