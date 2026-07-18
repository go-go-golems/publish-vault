const widget = require("widget.dsl");

const rows = [
	{ id: "sess-intro", title: "Intro to context windows", turns: 12, status: "ready" },
	{ id: "sess-debug", title: "Debugging retrieval", turns: 28, status: "review" },
];

const schema = widget.data
	.fields("sessions", (f) =>
		f
			.key("id", { label: "ID" })
			.primary("title", { label: "Title" })
			.count("turns", { label: "Turns" })
			.status("status", { label: "Status" }),
	)
	.build();

const table = widget.data.collection("sessions", rows, (c) => c.schema(schema).table()).toNode();

const page = widget.page("Simple table", (p) =>
	p.section("Sessions", (s) =>
		s.caption("A data.collection table emitted by widget.dsl v3.").view(table),
	),
);
