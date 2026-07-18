// Widget page: tag overview.
// Served at /api/widget/pages/tags and rendered by the SPA at /w/tags.
const widget = require("widget.dsl");
const vault = require("vault.data");

const rows = vault.tags().map((t) => ({ tag: t.tag, count: t.count }));

const schema = widget.data
	.fields("tags", (f) => f.key("tag", { label: "Tag" }).count("count", { label: "Notes" }))
	.build();

const table = widget.data
	.collection("tags", rows, (c) =>
		c.schema(schema).table((t) => t.rowSelect(widget.act.navigate("/search?q=%23${row.tag}"))),
	)
	.toNode();

const page = widget.page("Tags", (p) =>
	p.id("tags").section("All tags", (s) =>
		s
			.metric("Distinct tags", String(rows.length))
			.caption("Click a tag to search notes carrying it.")
			.view(table),
	),
);
