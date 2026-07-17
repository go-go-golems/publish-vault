// Widget page: recently updated notes.
// Served at /api/widget/pages/recent and rendered by the SPA at /w/recent.
const widget = require("widget.dsl");
const vault = require("vault.data");

const config = vault.config();
const notes = vault.notes();

const rows = notes
	.slice()
	.sort((a, b) => (a.modTime < b.modTime ? 1 : -1))
	.slice(0, 25)
	.map((n) => ({
		slug: n.slug,
		title: n.title,
		modTime: n.modTime,
		tags: n.tags.join(", "),
	}));

const schema = widget.data
	.fields("recent", (f) =>
		f
			.key("slug", { label: "Slug" })
			.primary("title", { label: "Title" })
			.date("modTime", { label: "Updated" })
			.short("tags", { label: "Tags" }),
	)
	.build();

const table = widget.data
	.collection("recent", rows, (c) =>
		c.schema(schema).table((t) => t.rowSelect(widget.act.navigate("/note/${row.slug}"))),
	)
	.toNode();

const page = widget.page("Recently updated", (p) =>
	p.id("recent").section("Last 25 notes", (s) =>
		s
			.metric("Notes in vault", String(config.notes))
			.caption("Newest first; click a row to open the note.")
			.view(table),
	),
);
