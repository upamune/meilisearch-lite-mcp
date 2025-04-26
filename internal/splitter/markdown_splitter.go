package splitter

import (
	"strings"

	"github.com/ikawaha/kagome/v2/tokenizer"
	"github.com/pkoukk/tiktoken-go"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type Chunk struct {
	Text     string
	StartIdx int
	EndIdx   int
	Type     string   // "code" or "text"
	Headings []string // heading hierarchy
}

// SplitMarkdown returns code-block単独チャンク + 日本語文チャンク化
func SplitMarkdown(md string, chunkTok, overlapTok int) ([]Chunk, error) {
	enc, _ := tiktoken.GetEncoding("cl100k_base")
	countTok := enc.CountTokens
	jaTok := tokenizer.New()
	var chunks []Chunk

	root := goldmark.DefaultParser().Parse(text.NewReader([]byte(md)))
	var headings []string
	var cur strings.Builder
	curStart := 0

	flushText := func() {
		if cur.Len() == 0 {
			return
		}
		txt := Sanitize(cur.String())
		chunks = append(chunks, Chunk{
			Text:     txt,
			Type:     "text",
			StartIdx: curStart,
			EndIdx:   curStart + len(txt),
			Headings: append([]string(nil), headings...),
		})
		cur.Reset()
	}

	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch v := n.(type) {
		case *ast.Heading:
			if entering {
				seg := v.Text(md)
				headings = append(headings[:v.Level-1], string(seg))
			}
		case *ast.FencedCodeBlock:
			if entering {
				flushText()
				seg := v.Text(md)
				chunks = append(chunks, Chunk{
					Text:     "```" + string(seg) + "```",
					Type:     "code",
					Headings: append([]string(nil), headings...),
					StartIdx: v.Segment.Start,
					EndIdx:   v.Segment.Stop,
				})
			}
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			if entering && v.Parent().Kind() != ast.KindFencedCodeBlock {
				seg := v.Segment
				sub := md[seg.Start:seg.Stop]
				sents := jaTok.SentenceSplitter().Split(sub)
				for _, s := range sents {
					if countTok(cur.String()+s) > chunkTok {
						flushText()
						curStart = seg.Start + strings.Index(sub, s)
					}
					cur.WriteString(s)
				}
			}
		}
		return ast.WalkContinue, nil
	})
	flushText()
	return chunks, nil
}
