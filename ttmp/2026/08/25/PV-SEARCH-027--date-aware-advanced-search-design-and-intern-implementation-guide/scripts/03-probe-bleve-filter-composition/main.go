package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/blevesearch/bleve/v2"
	bq "github.com/blevesearch/bleve/v2/search/query"
)

type doc struct {
	Title     string    `json:"title"`
	TagsKW    []string  `json:"tags_kw"`
	PathKW    string    `json:"path_kw"`
	DisplayAt time.Time `json:"display_at"`
}

type result struct {
	SchemaVersion int                 `json:"schema_version"`
	Queries       map[string][]string `json:"queries"`
}

func main() {
	mapping := bleve.NewIndexMapping()
	dm := bleve.NewDocumentMapping()
	keyword := bleve.NewKeywordFieldMapping()
	keyword.Store = true
	dm.AddFieldMappingsAt("tags_kw", keyword)
	dm.AddFieldMappingsAt("path_kw", keyword)
	dt := bleve.NewDateTimeFieldMapping()
	dt.Store = true
	dm.AddFieldMappingsAt("display_at", dt)
	dm.AddFieldMappingsAt("title", bleve.NewTextFieldMapping())
	mapping.DefaultMapping = dm

	idx, err := bleve.NewMemOnly(mapping)
	if err != nil {
		panic(err)
	}
	defer func() { _ = idx.Close() }()
	mustDate := func(v string) time.Time {
		t, e := time.Parse(time.RFC3339, v)
		if e != nil {
			panic(e)
		}
		return t
	}
	docs := map[string]doc{
		"a": {Title: "Alpha", TagsKW: []string{"go", "search"}, PathKW: "research/go/a.md", DisplayAt: mustDate("2024-01-15T00:00:00Z")},
		"b": {Title: "Beta", TagsKW: []string{"go", "memory"}, PathKW: "research/go/b.md", DisplayAt: mustDate("2024-01-16T00:00:00Z")},
		"c": {Title: "Gamma", TagsKW: []string{"search", "memory"}, PathKW: "projects/c.md", DisplayAt: mustDate("2024-01-16T12:00:00Z")},
	}
	for id, d := range docs {
		if err := idx.Index(id, d); err != nil {
			panic(err)
		}
	}

	term := func(field, value string) bq.Query { q := bleve.NewTermQuery(value); q.SetField(field); return q }
	prefix := func(field, value string) bq.Query { q := bleve.NewPrefixQuery(value); q.SetField(field); return q }
	start, end := mustDate("2024-01-16T00:00:00Z"), mustDate("2024-01-17T00:00:00Z")
	inclusive, exclusive := true, false
	dateRange := bleve.NewDateRangeInclusiveQuery(start, end, &inclusive, &exclusive)
	dateRange.SetField("display_at")

	queries := map[string]bq.Query{
		"tag_go":                 term("tags_kw", "go"),
		"tags_go_and_search":     bleve.NewConjunctionQuery(term("tags_kw", "go"), term("tags_kw", "search")),
		"tags_go_or_search":      bleve.NewDisjunctionQuery(term("tags_kw", "go"), term("tags_kw", "search")),
		"path_research_go":       prefix("path_kw", "research/go/"),
		"go_and_date_2024_01_16": bleve.NewConjunctionQuery(term("tags_kw", "go"), dateRange),
	}
	out := result{SchemaVersion: 1, Queries: map[string][]string{}}
	for name, query := range queries {
		req := bleve.NewSearchRequestOptions(query, 10, 0, false)
		req.SortBy([]string{"_id"})
		response, err := idx.Search(req)
		if err != nil {
			panic(err)
		}
		for _, hit := range response.Hits {
			out.Queries[name] = append(out.Queries[name], hit.ID)
		}
		if out.Queries[name] == nil {
			out.Queries[name] = []string{}
		}
	}
	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(encoded))
}
