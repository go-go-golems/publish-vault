package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/blevesearch/bleve/v2"
)

type doc struct {
	Title     string    `json:"title"`
	DisplayAt time.Time `json:"display_at"`
}

type output struct {
	SchemaVersion int        `json:"schema_version"`
	HitIDs        []string   `json:"hit_ids"`
	SortValues    [][]string `json:"sort_values"`
}

func main() {
	m := bleve.NewIndexMapping()
	dm := bleve.NewDocumentMapping()
	df := bleve.NewDateTimeFieldMapping()
	df.Store = true
	dm.AddFieldMappingsAt("display_at", df)
	dm.AddFieldMappingsAt("title", bleve.NewTextFieldMapping())
	m.DefaultMapping = dm
	idx, err := bleve.NewMemOnly(m)
	if err != nil {
		panic(err)
	}
	defer func() { _ = idx.Close() }()
	for id, value := range map[string]string{
		"a": "2024-01-15T00:00:00Z", "b": "2024-01-16T12:00:00Z", "c": "2024-01-16T08:00:00Z", "d": "2024-01-17T00:00:00Z",
	} {
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			panic(err)
		}
		if err := idx.Index(id, doc{Title: id, DisplayAt: t}); err != nil {
			panic(err)
		}
	}
	start, _ := time.Parse(time.RFC3339, "2024-01-16T00:00:00Z")
	endExclusive, _ := time.Parse(time.RFC3339, "2024-01-17T00:00:00Z")
	inclusive, exclusive := true, false
	q := bleve.NewDateRangeInclusiveQuery(start, endExclusive, &inclusive, &exclusive)
	q.SetField("display_at")
	req := bleve.NewSearchRequestOptions(q, 10, 0, false)
	req.Fields = []string{"display_at"}
	req.SortBy([]string{"-display_at", "_id"})
	result, err := idx.Search(req)
	if err != nil {
		panic(err)
	}
	out := output{SchemaVersion: 1}
	for _, h := range result.Hits {
		out.HitIDs = append(out.HitIDs, h.ID)
		out.SortValues = append(out.SortValues, h.Sort)
	}
	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(encoded))
}
