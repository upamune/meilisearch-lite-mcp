package splitter

import (
	"strings"

	"github.com/ikawaha/kagome-dict/uni"
	"github.com/ikawaha/kagome/v2/tokenizer"
	"github.com/pkoukk/tiktoken-go"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var jaTokenizer *tokenizer.Tokenizer

func init() {
	var err error
	jaTokenizer, err = tokenizer.New(uni.Dict())
	if err != nil {
		panic(err)
	}
}

type Chunk struct {
	Text     string
	StartIdx int
	EndIdx   int
	Type     string   // "code" or "text"
	Headings []string // heading hierarchy
}

// SplitMarkdown returns code-block単独チャンク + 日本語文チャンク化
func SplitMarkdown(md string, chunkTok, overlapTok int) ([]Chunk, error) {
	enc, err := tiktoken.GetEncoding(tiktoken.MODEL_CL100K_BASE)
	if err != nil {
		return nil, err
	}
	countTok := func(s string) int { return len(enc.Encode(s, nil, nil)) }

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

	if err := ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch v := n.(type) {
		case *ast.Heading:
			if entering {
				seg := v.Lines()
				if seg.Len() > 0 {
					headings = append(headings[:v.Level-1], string(v.Text([]byte(md))))
				}
			}
		case *ast.FencedCodeBlock:
			if entering {
				flushText()
				// extract code block text and segments
				seg := v.Text([]byte(md))
				lines := v.Lines()
				startIdx, endIdx := 0, 0
				if lines.Len() > 0 {
					first := lines.At(0)
					last := lines.At(lines.Len() - 1)
					startIdx = first.Start
					endIdx = last.Stop
				}
				chunks = append(chunks, Chunk{
					Text:     "```" + string(seg) + "```",
					Type:     "code",
					Headings: append([]string(nil), headings...),
					StartIdx: startIdx,
					EndIdx:   endIdx,
				})
			}
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			if entering && v.Parent().Kind() != ast.KindFencedCodeBlock {
				seg := v.Segment
				sub := md[seg.Start:seg.Stop]
				// split text into sentences
				sents, err := splitSentences(sub)
				if err != nil {
					return ast.WalkSkipChildren, err
				}
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
	}); err != nil {
		return nil, err
	}

	flushText()
	return chunks, nil
}

// splitSentences splits Japanese text into sentences using Kagome.
func splitSentences(s string) ([]string, error) {
	var result []string
	var buf strings.Builder
	// use morphological segmentation
	morphs := jaTokenizer.Wakati(s)
	for _, surface := range morphs {
		if _, err := buf.WriteString(surface); err != nil {
			return nil, err
		}
		if surface == "。" || surface == "！" || surface == "？" {
			result = append(result, buf.String())
			buf.Reset()
		}
	}
	if buf.Len() > 0 {
		result = append(result, buf.String())
	}
	return result, nil
}
