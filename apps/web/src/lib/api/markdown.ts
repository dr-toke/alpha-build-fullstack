// Minimal, dependency-free markdown → safe HTML renderer for forum posts.
// Exact port of ../dr-toke/apps/web/lib/markdown.ts. Escapes ALL input first,
// then applies a small set of markdown transforms, so authored content can
// never inject HTML. Supports: headings, bold, italic, inline code, links
// (http/https only), unordered lists, blockquotes, paragraphs.

function escapeHtml(s: string): string {
	return s
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;')
		.replace(/'/g, '&#39;');
}

function inline(s: string): string {
	// Links: [text](http(s)://...) — escaped already, so match the escaped quotes too.
	s = s.replace(
		/\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g,
		(_m, text: string, url: string) =>
			`<a href="${url}" target="_blank" rel="noopener noreferrer">${text}</a>`
	);
	s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
	s = s.replace(/\*([^*]+)\*/g, '<em>$1</em>');
	s = s.replace(/`([^`]+)`/g, '<code>$1</code>');
	return s;
}

export function renderMarkdown(md: string): string {
	const lines = escapeHtml(md).replace(/\r\n/g, '\n').split('\n');
	const out: string[] = [];
	let inList = false;
	let para: string[] = [];

	const flushPara = () => {
		if (para.length) {
			out.push(`<p>${inline(para.join(' '))}</p>`);
			para = [];
		}
	};
	const closeList = () => {
		if (inList) {
			out.push('</ul>');
			inList = false;
		}
	};

	for (const raw of lines) {
		const line = raw.trimEnd();

		if (line.trim() === '') {
			flushPara();
			closeList();
			continue;
		}

		const heading = /^(#{1,4})\s+(.*)$/.exec(line);
		if (heading) {
			flushPara();
			closeList();
			const level = (heading[1] ?? '#').length;
			out.push(`<h${level}>${inline(heading[2] ?? '')}</h${level}>`);
			continue;
		}

		const li = /^[-*]\s+(.*)$/.exec(line);
		if (li) {
			flushPara();
			if (!inList) {
				out.push('<ul>');
				inList = true;
			}
			out.push(`<li>${inline(li[1] ?? '')}</li>`);
			continue;
		}

		const quote = /^>\s+(.*)$/.exec(line);
		if (quote) {
			flushPara();
			closeList();
			out.push(`<blockquote>${inline(quote[1] ?? '')}</blockquote>`);
			continue;
		}

		para.push(line.trim());
	}

	flushPara();
	closeList();
	return out.join('\n');
}
