package parser

import (
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	goldparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// expandedTableExtension preserves extra cells in malformed-but-common vault
// tables and pipes inside math or wiki links. Goldmark's GFM extension truncates
// extra body cells and splits on pipes before inline extensions can recognize
// them. Other conforming tables continue through Goldmark unchanged.
type expandedTableExtension struct{}

func (expandedTableExtension) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		goldparser.WithParagraphTransformers(util.Prioritized(expandedTableParagraph{}, 190)),
		goldparser.WithASTTransformers(util.Prioritized(expandedTableCodeEscapes{}, 50)),
	)
}

var expandedTablePipesKey = goldparser.NewContextKey()

type expandedTableParagraph struct{}

func (expandedTableParagraph) Transform(p *ast.Paragraph, reader text.Reader, pc goldparser.Context) {
	lines := p.Lines()
	source := reader.Source()
	for i := 1; i < lines.Len(); i++ {
		alignments, ok := tableAlignments(lines.At(i), source)
		if !ok {
			continue
		}
		header, _, protected := tableSegments(lines.At(i-1), source)
		if len(header) != len(alignments) {
			return
		}
		maxColumns := len(alignments)
		for j := i + 1; j < lines.Len(); j++ {
			cells, _, hasProtected := tableSegments(lines.At(j), source)
			protected = protected || hasProtected
			if len(cells) > maxColumns {
				maxColumns = len(cells)
			}
		}
		if maxColumns == len(alignments) && !protected {
			return
		}
		for len(alignments) < maxColumns {
			alignments = append(alignments, extast.AlignNone)
		}
		table := extast.NewTable()
		table.Alignments = alignments
		table.SetPos(p.Pos())
		table.AppendChild(table, extast.NewTableHeader(expandedTableRow(lines.At(i-1), source, alignments, pc)))
		for j := i + 1; j < lines.Len(); j++ {
			table.AppendChild(table, expandedTableRow(lines.At(j), source, alignments, pc))
		}
		p.Lines().SetSliced(0, i-1)
		parent := p.Parent()
		parent.InsertAfter(parent, p, table)
		if p.Lines().Len() == 0 {
			parent.RemoveChild(parent, p)
		} else {
			last := p.Lines().At(p.Lines().Len() - 1)
			if last.Stop > last.Start && source[last.Stop-1] == '\n' {
				last.Stop--
				p.Lines().Set(p.Lines().Len()-1, last)
			}
		}
		return
	}
}

func tableAlignments(line text.Segment, source []byte) ([]extast.Alignment, bool) {
	value := line.Value(source)
	if width, _ := util.IndentWidth(value, 0); width > 3 {
		return nil, false
	}
	for _, c := range value {
		if !util.IsSpace(c) && c != '-' && c != '|' && c != ':' {
			return nil, false
		}
	}
	cells, _, _ := tableSegments(line, source)
	if len(cells) == 0 {
		return nil, false
	}
	out := make([]extast.Alignment, 0, len(cells))
	for _, cell := range cells {
		value := strings.TrimSpace(string(cell.Value(source)))
		left, right := strings.HasPrefix(value, ":"), strings.HasSuffix(value, ":")
		value = strings.TrimPrefix(value, ":")
		value = strings.TrimSuffix(value, ":")
		if value == "" || strings.Trim(value, "-") != "" {
			return nil, false
		}
		align := extast.AlignNone
		switch {
		case left && right:
			align = extast.AlignCenter
		case left:
			align = extast.AlignLeft
		case right:
			align = extast.AlignRight
		}
		out = append(out, align)
	}
	return out, true
}

// tableSegments preserves pipes inside math and wiki nodes as well as escaped
// pipes. Escape positions let the code-span pass discard table-specific escapes.
func tableSegments(line text.Segment, source []byte) ([]text.Segment, []int, bool) {
	line = line.TrimLeftSpace(source)
	line = line.TrimRightSpace(source)
	value := line.Value(source)
	type span struct{ start, end int }
	var protected []span
	for _, math := range ScanMath(value) {
		protected = append(protected, span{math.Start, math.End})
	}
	var codes []span
	for i := 0; i < len(value); {
		if value[i] == '\\' && i+1 < len(value) {
			i += 2
			continue
		}
		if value[i] == '`' {
			end := skipCodeSpan(value, i)
			if end > i+1 {
				codes = append(codes, span{i, end})
			}
			i = end
			continue
		}
		i++
	}
	for _, loc := range wikiLinkRegex.FindAllIndex(value, -1) {
		backslashes := 0
		for i := loc[0] - 1; i >= 0 && value[i] == '\\'; i-- {
			backslashes++
		}
		if backslashes%2 != 0 {
			continue
		}
		inCode := false
		for _, code := range codes {
			if loc[0] >= code.start && loc[0] < code.end {
				inCode = true
				break
			}
		}
		if !inCode {
			protected = append(protected, span{loc[0], loc[1]})
		}
	}
	inProtected := func(pos int) bool {
		for _, s := range protected {
			if pos >= s.start && pos < s.end {
				return true
			}
		}
		return false
	}
	start, end := 0, len(value)
	if start < end && value[start] == '|' {
		start++
	}
	if start < end && value[end-1] == '|' {
		end--
	}
	var cells []text.Segment
	var escapes []int
	hasProtected := false
	for start < end {
		stop := start
		for stop < end {
			if value[stop] == '|' {
				if inProtected(stop) {
					hasProtected = true
					stop++
					continue
				}
				if stop > 0 && value[stop-1] == '\\' {
					escapes = append(escapes, line.Start+stop-1)
					stop++
					continue
				}
				break
			}
			stop++
		}
		cell := text.NewSegment(line.Start+start, line.Start+stop)
		cell = cell.TrimLeftSpace(source)
		cell = cell.TrimRightSpace(source)
		cells = append(cells, cell)
		start = stop + 1
	}
	return cells, escapes, hasProtected
}
func expandedTableRow(line text.Segment, source []byte, alignments []extast.Alignment, pc goldparser.Context) *extast.TableRow {
	segments, escapes, _ := tableSegments(line, source)
	if len(escapes) > 0 {
		old, _ := pc.Get(expandedTablePipesKey).([]int)
		pc.Set(expandedTablePipesKey, append(old, escapes...))
	}
	row := extast.NewTableRow(alignments)
	row.SetPos(line.Start)
	for i, align := range alignments {
		cell := extast.NewTableCell()
		cell.Alignment = align
		if i < len(segments) {
			cell.SetPos(segments[i].Start)
			cell.Lines().Append(segments[i])
		}
		row.AppendChild(row, cell)
	}
	return row
}

type expandedTableCodeEscapes struct{}

func (expandedTableCodeEscapes) Transform(root *ast.Document, _ text.Reader, pc goldparser.Context) {
	positions, _ := pc.Get(expandedTablePipesKey).([]int)
	if len(positions) == 0 {
		return
	}
	sort.Ints(positions)
	_ = ast.Walk(root, func(node ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter || node.Kind() != ast.KindCodeSpan {
			return ast.WalkContinue, nil
		}
		for c := node.FirstChild(); c != nil; {
			next := c.NextSibling()
			value, ok := c.(*ast.Text)
			if ok {
				segment := value.Segment
				cursor := segment.Start
				var parts []text.Segment
				for _, pos := range positions {
					if pos >= cursor && pos < segment.Stop {
						part := segment.WithStart(cursor)
						parts = append(parts, part.WithStop(pos))
						cursor = pos + 1
					}
				}
				if len(parts) > 0 {
					parts = append(parts, segment.WithStart(cursor))
					for _, part := range parts {
						node.InsertBefore(node, c, ast.NewRawTextSegment(part))
					}
					node.RemoveChild(node, c)
				}
			}
			c = next
		}
		return ast.WalkContinue, nil
	})
}
