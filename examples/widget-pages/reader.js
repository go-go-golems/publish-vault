// Widget page: JS note reader composed from the vault.widgets grammar.
// /w/reader renders the home note; /w/reader?slug=<slug> renders that note.
// Declares its own sidebar navigation via the v3 shell spec.
const widget = require("widget.dsl");
const vault = require("vault.data");
const vw = require("vault.widgets");

const slug = (request.query && request.query.slug) || "index";
const note = vault.note(slug) || vault.note(vault.notes()[0].slug);

const shell = widget.app.shell((s) =>
	s.navigation((nav) =>
		nav
			.placement("sidebar")
			.active("reader")
			.section("pages", "Pages", (items) =>
				items
					.item("reader", "Reader", widget.act.navigate("/w/reader"))
					.item("recent", "Recently updated", widget.act.navigate("/w/recent"))
					.item("tags", "Tags", widget.act.navigate("/w/tags")),
			)
			.section("notes", "Latest notes", (items) => {
				vault
					.notes()
					.slice(0, 5)
					.forEach((n) =>
						items.item(n.slug, n.title, widget.act.navigate("/w/reader?slug=" + n.slug)),
					);
			}),
	),
);

const page = widget.page(note.title, (p) =>
	p
		.id("reader")
		.shell(shell)
		// Page-level views: the WidgetPage h1 already shows the note title, so
		// no section wrapper around the note body itself.
		.view(vw.breadcrumb(note))
		.view(vw.frontmatter(note))
		.view(vw.noteHtml(note))
		.section("Linked mentions", (s) =>
			s.view(vw.backlinks(note, { onSelect: widget.act.navigate("/note/${slug}") })),
		),
);
