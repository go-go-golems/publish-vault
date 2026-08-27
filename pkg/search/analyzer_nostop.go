package search

import (
	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/registry"
)

// nostopAnalyzerName is the name of the custom analyzer used by the text fields
// in buildMapping. It is the unicode tokenizer + the lowercase token filter,
// intentionally WITHOUT the English stop-word filter that bleve's built-in
// "standard" analyzer applies. See buildMapping for why stopwords must be kept.
const nostopAnalyzerName = "nostop"

// registerNostopAnalyzer registers a "nostop" analyzer constructor in the
// bleve registry. AddCustomAnalyzer on an IndexMapping alone is not enough:
// bleve.New / NewMemOnly validate the mapping against the global registry, so
// the analyzer's composite constructor must be registered before the index is
// created. This mirrors how bleve's own "standard" analyzer registers itself.
func registerNostopAnalyzer() {
	_ = registry.RegisterAnalyzer(nostopAnalyzerName, func(config map[string]interface{}, cache *registry.Cache) (analysis.Analyzer, error) {
		tokenizer, err := cache.TokenizerNamed(unicode.Name)
		if err != nil {
			return nil, err
		}
		toLowerFilter, err := cache.TokenFilterNamed(lowercase.Name)
		if err != nil {
			return nil, err
		}
		return &analysis.DefaultAnalyzer{
			Tokenizer: tokenizer,
			TokenFilters: []analysis.TokenFilter{
				toLowerFilter,
			},
		}, nil
	})
}

func init() {
	registerNostopAnalyzer()
}
