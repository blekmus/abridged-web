package server

import "strings"

func appShellForPath(requestPath string) string {
	if requestPath != "/" && requestPath != "" {
		return ""
	}

	var shell strings.Builder
	shell.WriteString(`<div class="page-shell">`)
	shell.WriteString(`<header class="site-header"><div class="container site-header-row"><div class="site-header-rail"><a href="/" class="site-mark">The abridged catalogue</a></div><nav class="site-nav" aria-label="Primary"><a href="/series" class="nav-link accent-shows ">Shows</a><a href="/shorts" class="nav-link accent-shorts ">Shorts</a><a href="/shots" class="nav-link accent-one-shots ">One-Shots</a><a href="/songs" class="nav-link accent-songs ">Songs</a></nav></div></header>`)
	shell.WriteString(`<main class="site-main">`)
	shell.WriteString(`<section class="hero"><div class="container hero-grid"><div><h1 class="hero-title"><span style="color: #52dfff;">The</span><br><span style="color: #34dcba;">Abridged</span><br><span style="color: #ef3e78;">Catalogue</span></h1></div><div class="hero-panel" aria-label="Catalog counts"><div class="hero-stat shows-stat"><strong>...</strong><span class="hero-stat-label">shows</span></div><div class="hero-stat shorts-stat"><strong>...</strong><span class="hero-stat-label">shorts</span></div><div class="hero-stat one-shots-stat"><strong>...</strong><span class="hero-stat-label">one-shots</span></div><div class="hero-stat songs-stat"><strong>...</strong><span class="hero-stat-label">songs</span></div></div><p class="hero-copy">A catalogue of my favourite abridged anime now in web form. This is more of an awareness effort than a full blown archive.</p></div></section>`)

	writeHomeLoadingSection(&shell, "Shows", "/series", "accent-shows")
	writeHomeLoadingSection(&shell, "Shorts", "/shorts", "accent-shorts")
	writeHomeLoadingSection(&shell, "One-Shots", "/shots", "accent-one-shots")
	writeHomeLoadingSection(&shell, "Songs", "/songs", "accent-songs")
	writeHomeQuestions(&shell)

	shell.WriteString(`</main>`)
	shell.WriteString(`<footer class="site-footer"><div class="container footer-simple"><span class="footer-rule" aria-hidden="true"></span><p class="footer-credit">Made with love by <a href="https://garden.dinil.dev" target="_blank" rel="noreferrer">blekmus</a></p></div></footer>`)
	shell.WriteString(`</div>`)

	return shell.String()
}

func writeHomeLoadingSection(shell *strings.Builder, title string, href string, accentClass string) {
	shell.WriteString(`<section class="browse-section"><div class="container"><div class="section-header "><h2><a href="`)
	shell.WriteString(href)
	shell.WriteString(`" class="section-title-link `)
	shell.WriteString(accentClass)
	shell.WriteString(`">`)
	shell.WriteString(title)
	shell.WriteString(`</a></h2><span class="section-rule"></span></div><div class="card-grid">`)

	for range 4 {
		shell.WriteString(`<div class="browse-card skeleton-card"><div class="card-thumbnail-shell skeleton-block"></div><div class="card-copy"><div class="skeleton-line short"></div><div class="skeleton-line"></div><div class="skeleton-line short"></div></div></div>`)
	}

	shell.WriteString(`</div></div></section>`)
}

func writeHomeQuestions(shell *strings.Builder) {
	shell.WriteString(`<section class="qa-section"><div class="container qa-container"><div class="qa-intro"><h2 class="qa-title">Q&A</h2></div><div class="qa-list">`)
	shell.WriteString(`<details class="qa-item" open><summary>Why does this exist?</summary><div class="qa-answer"><p>I love going through this collection about once per year and I've done everything I can to make that as seamless as possible. Maybe I'm overengineering this a whole lot more than it needs to be, but I enjoy it. I've already built a <a href="https://github.com/blekmus/abridged-cli" target="_blank" rel="noreferrer">CLI</a> and a <a href="https://github.com/blekmus/raycast-abridged">Raycast Plugin</a> so a website is sort of the natural evolution.</p></div></details>`)
	shell.WriteString(`<details class="qa-item"><summary>Where can I find more?</summary><div class="qa-answer"><p>Once upon a time I made a <a href="https://nyaa.si/view/1979033" target="_blank" rel="noreferrer">torrent</a> with a bunch of the entries here. Other than that, there's always the <a target="_blank" href="https://abridgedseries.fandom.com/wiki/Abridged_Archive" rel="noreferrer">abridged archive.</a></p></div></details>`)
	shell.WriteString(`<details class="qa-item"><summary>Why are some classics missing?</summary><div class="qa-answer"><p>As much as I want to include every abridgedment out there, I'm limited by how much storage I have on my server. So I decided to not host the big titles that are already easily accessible.</p></div></details>`)
	shell.WriteString(`<details class="qa-item"><summary>How are shows, shorts, and one-shots split?</summary><div class="qa-answer"><p><strong>Shows: </strong> A set of entries that run episodically or are part of a series of works by a creator or group of creators.</p><p><strong>One-shots: </strong> Single videos that are not part of a continuous series but share an overarching story. These are usually longer than 5 minutes.</p><p><strong>Shorts: </strong>Similar to Shots, but shorter in length and may not have a well-defined plot.</p></div></details>`)
	shell.WriteString(`</div></div></section>`)
}
