package splitter

import (
	"log"
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

	var root = goldmark.DefaultParser().Parse(text.NewReader([]byte(md)))
	var headings []string
	var cur strings.Builder
	var curStart int
	// var curEnd int // No longer track EndIdx per sentence, calculate on flush
	var curTokens int
	var prevAbsEnd int = -1 // Tracks the *end* index of the *previously* added sentence segment for separator calculation

	flushText := func() {
		if cur.Len() == 0 {
			return
		}
		chunkText := cur.String()
		// Calculate final EndIdx based on StartIdx and final chunk length
		finalEndIdx := curStart + len(chunkText) // Exclusive index
		log.Printf("flushText called. Current Headings: %v, curStart: %d, calculated finalEndIdx: %d, curTokens: %d", headings, curStart, finalEndIdx, curTokens)

		// Create a copy of headings to store with the chunk
		var chunkHeadings []string
		if len(headings) > 0 {
			chunkHeadings = make([]string, len(headings))
			copy(chunkHeadings, headings)
		}
		// else chunkHeadings remains nil

		log.Printf("  >> Appending Chunk: Text: \"%.30s...\", StartIdx: %d, EndIdx: %d, Headings: %v", chunkText, curStart, finalEndIdx, chunkHeadings) // Log the copy

		chunks = append(chunks, Chunk{
			Text:     chunkText,
			StartIdx: curStart,
			EndIdx:   finalEndIdx, // Use the final calculated EndIdx
			Type:     "text",
			Headings: chunkHeadings, // Assign the copy
		})
		cur.Reset()
		curTokens = 0
		curStart = 0
		// curEnd = 0 // No longer needed
		prevAbsEnd = -1 // Reset prevAbsEnd when buffer is flushed
	}

	if err := ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch v := n.(type) {
		case *ast.Heading:
			if entering {
				flushText() // Ensure pending text is flushed before processing heading
				level := v.Level
				headingText := string(v.Text([]byte(md)))
				log.Printf("  Entering Heading L%d: '%s'. Current headings: %v", level, headingText, headings)
				// Ensure headings slice is long enough or resize
				if level > len(headings) {
					newHeadings := make([]string, level)
					copy(newHeadings, headings)
					headings = newHeadings
					log.Printf("    Resized headings slice to length %d", level)
				} else {
					// Truncate deeper levels when a shallower heading is encountered
					headings = headings[:level]
					log.Printf("    Truncated headings slice to length %d", level)
				}
				// Set heading at the correct level
				headings[level-1] = headingText
				log.Printf("    Updated headings: %v", headings)
			} else {
				log.Printf("  Exiting Heading L%d. Headings remain: %v", v.Level, headings)
				// No action needed on exiting heading, state persists
			}
			return ast.WalkContinue, nil // Continue walking children (though headings usually don't have complex children affecting text flow here)
		case *ast.FencedCodeBlock:
			if entering {
				flushText() // Flush any preceding text

				var blockText strings.Builder
				blockText.WriteString("```")
				if v.Info != nil {
					blockText.Write(v.Info.Text([]byte(md)))
				}
				blockText.WriteString("\n")
				startIdx, endIdx := -1, -1 // Initialize indices
				if v.Lines().Len() > 0 {
					firstLine := v.Lines().At(0)
					lastLine := v.Lines().At(v.Lines().Len() - 1)
					startIdx = firstLine.Start // Start index of the first line's content
					endIdx = lastLine.Stop    // End index of the last line's content (exclusive)

					for i := 0; i < v.Lines().Len(); i++ {
						line := v.Lines().At(i)
						blockText.Write(line.Value([]byte(md)))
					}
					// Ensure final newline if not present in source, affecting display but not indices
					if md[lastLine.Stop-1] != '\n' {
						blockText.WriteString("\n") // Append for visual consistency in output text
					}
				} else {
					// Handle empty code block case if necessary, indices might be tricky
					// Maybe use v.Segment.Start/Stop? For now, -1 indicates content-based indexing failed.
				}
				blockText.WriteString("```")

				log.Printf("  Code Block Found: StartIdx: %d, EndIdx: %d", startIdx, endIdx)

				// Create a copy of headings for the code chunk
				var codeChunkHeadings []string
				if len(headings) > 0 {
					codeChunkHeadings = make([]string, len(headings))
					copy(codeChunkHeadings, headings)
				}

				chunks = append(chunks, Chunk{
					Text:     blockText.String(),
					StartIdx: startIdx,
					EndIdx:   endIdx, // Use calculated indices based on content lines
					Type:     "code",
					Headings: codeChunkHeadings, // Assign the copy
				})
			}
			return ast.WalkSkipChildren, nil // Don't process text nodes within code blocks
		case *ast.Text:
			if entering {
				// Ignore text nodes that are direct children of headings
				if parent := v.Parent(); parent != nil && parent.Kind() == ast.KindHeading {
					return ast.WalkContinue, nil
				}

				nodeText := string(v.Text([]byte(md)))
				if nodeText == "" {
					return ast.WalkContinue, nil
				}
				// Trim node text only for sentence splitting, keep original for index calculation
				trimmedNodeTextForSplitting := strings.TrimSpace(nodeText)
				if trimmedNodeTextForSplitting == "" {
					return ast.WalkContinue, nil // Skip if node is only whitespace
				}

				// Use the trimmed text for sentence splitting
				sentences := splitSentences(trimmedNodeTextForSplitting)
				// Base offset within the *original* nodeText needs careful tracking
				currentOffsetInNode := 0

				for _, sentence := range sentences {
					sanitizedSentence := Sanitize(sentence) // Sanitize the sentence from splitSentences
					if sanitizedSentence == "" {
						// Still need to advance offset in original nodeText based on raw sentence length
						// Find the raw sentence in the remaining nodeText to calculate length correctly
						rawSentenceIndex := strings.Index(nodeText[currentOffsetInNode:], sentence)
						if rawSentenceIndex != -1 {
							currentOffsetInNode += rawSentenceIndex + len(sentence)
						} else {
                             // This shouldn't happen if sentence came from nodeText, log warning
							log.Printf("Warning: Could not find raw sentence '%s' in nodeText remnant '%s'", sentence, nodeText[currentOffsetInNode:])
							// Best guess recovery: advance by length of sanitized sentence? Or original sentence?
							currentOffsetInNode += len(sentence) // Advance by original sentence len as fallback
                        }
						continue
					}

					tokenCount := countTok(sanitizedSentence)

					// Find the start of the *raw* sentence in the *original* nodeText[currentOffsetInNode:]
					relStartInRemnant := strings.Index(nodeText[currentOffsetInNode:], sentence)
					if relStartInRemnant == -1 {
						log.Printf("Warning: Could not find raw sentence substring for index. Offset: %d, NodeText: '%s', Sentence: '%s'", currentOffsetInNode, nodeText, sentence)
						// Best guess recovery: advance offset and skip sentence
						currentOffsetInNode += len(sentence)
						continue
					}
                     // Absolute start index in the full markdown string
					absStart := v.Segment.Start + currentOffsetInNode + relStartInRemnant
                    // Absolute end index = start + length of the *raw* sentence
					absEnd := absStart + len(sentence)

					log.Printf("  Processing Sentence: '%.30s...', absStart: %d, absEnd: %d, tokens: %d, curTokens: %d, maxTokens: %d", sanitizedSentence, absStart, absEnd, tokenCount, curTokens, chunkTok)

					// --- Token Limit Check ---
					// Rough check: If adding this sentence exceeds limit, flush first.
					// TODO: A more precise check would include separator tokens.
					if cur.Len() > 0 && curTokens+tokenCount >= chunkTok {
						log.Printf("    Token limit would be exceeded (Sentence Tokens: %d). Flushing current buffer.", tokenCount)
						flushText() // Resets prevAbsEnd
					}

					// --- Buffer Management ---
					if cur.Len() == 0 {
						log.Printf("    Buffer empty. Setting curStart = %d", absStart)
						curStart = absStart
						prevAbsEnd = -1 // Ensure prevAbsEnd is reset correctly when starting fresh buffer
					}

					// --- Append Separator (if needed) ---
					if cur.Len() > 0 { // Only add separator if buffer isn't empty
                        if prevAbsEnd != -1 && absStart > prevAbsEnd {
                            // Ensure indices are within the bounds of the original markdown string 'md'
                            if prevAbsEnd >= 0 && absStart <= len(md) {
                                separator := md[prevAbsEnd:absStart] // Capture original separator
                                cur.WriteString(separator)
                                // Optional: Add separator tokens if significant: curTokens += countTok(separator)
								log.Printf("    Appended separator: '%s'", separator)
                            } else {
                                log.Printf("Warning: Invalid indices for separator slicing. prevAbsEnd: %d, absStart: %d, mdLen: %d. Appending fallback space.", prevAbsEnd, absStart, len(md))
                                cur.WriteString(" ") // Fallback separator
                            }
						} else if prevAbsEnd != -1 && absStart == prevAbsEnd { // Sentences abut, add default space
							log.Printf("    Sentences abut (absStart == prevAbsEnd == %d). Adding default space.", absStart)
							cur.WriteString(" ")
						} else if prevAbsEnd != -1 && absStart < prevAbsEnd { // Should not happen
                            // This might indicate overlapping sentences or index issues, log warning
                            log.Printf("Warning: absStart (%d) < prevAbsEnd (%d). Skipping separator, potential overlap.", absStart, prevAbsEnd)
                            // Don't add a separator in this unusual case. Consider adding a space if desired.
						} else {
							// If prevAbsEnd is -1 (start of buffer) or cur.Len() == 0, no separator needed yet.
						}
					}

					// --- Append Sanitized Sentence ---
					cur.WriteString(sanitizedSentence)
					curTokens += tokenCount
					// curEnd = absEnd // No longer track EndIdx here
					prevAbsEnd = absEnd // Store the end of the *raw* sentence segment

					// --- Advance Offset in Node ---
					// Advance offset to the position *after* the raw sentence in the nodeText
					currentOffsetInNode += relStartInRemnant + len(sentence)

				}
			}
			// No 'else' needed for entering=false for ast.Text usually
			return ast.WalkContinue, nil
		case *ast.Paragraph:
            if !entering && cur.Len() > 0 {
                 // If we are exiting a paragraph and have content buffered,
                 // check if the next node might need a separator that isn't naturally captured.
                 // This is heuristic. A simple approach is to add a space if the buffer
                 // doesn't end with one, preparing for potentially joining with the next block.
                 // However, relying on capturing separators between text nodes is safer.
                 // Let's rely on the text node logic for now.
                 // log.Printf("Exiting Paragraph. Buffer content: '%s'", cur.String())
            }
             return ast.WalkContinue, nil
         case *ast.Document:
              // Nothing specific needed for Document node itself usually
              return ast.WalkContinue, nil
          // Add other cases as needed (e.g., lists, blockquotes) if they affect chunking logic
		}
		return ast.WalkContinue, nil // Default action
	}); err != nil {
		return nil, err
	}

	// Final flush for any remaining text in the buffer
	flushText()

	return chunks, nil
}

// splitSentences splits Japanese text into sentences using Kagome.
func splitSentences(s string) []string {
	var result []string
	var buf strings.Builder
	morphs := jaTokenizer.Wakati(s)
	for _, surface := range morphs {
		if _, err := buf.WriteString(surface); err != nil {
			log.Printf("Warning: failed to write string to buffer in splitSentences: %v", err)
			continue
		}

		if surface == "。" || surface == "！" || surface == "？" || surface == "." || surface == "!" || surface == "?" {
			result = append(result, buf.String())
			buf.Reset()
		}
	}

	if buf.Len() > 0 {
		result = append(result, buf.String())
	}

	if len(result) == 0 && s != "" {
		return []string{s}
	}

	return result
}
