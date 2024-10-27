package highlighter

import (
	"regexp"
	"slices"
)

type TokenPattern struct {
	Name     string
	Pattern  *regexp.Regexp
	Priority int
}

// ordered by precedence (lowest to highest)
var allPatterns = []TokenPattern{
	{Name: "whitespace", Pattern: regexp.MustCompile(`\s+`), Priority: 0},
	{Name: "boolean", Pattern: regexp.MustCompile(`true|false|True|False`), Priority: 10},
	{Name: "operator", Pattern: regexp.MustCompile(`[+\-*/\.\=\{\}\(\)\[\];:]`), Priority: 20},
	{Name: "number", Pattern: regexp.MustCompile(`\d+(\.\d+)?`), Priority: 30},
	{Name: "string", Pattern: regexp.MustCompile(`"[^"]+"`), Priority: 40},
	{Name: "comment", Pattern: regexp.MustCompile(`//.*`), Priority: 50},
}

type Span struct {
	Start        int
	End          int
	Value        string
	Kind         string
	tokenPattern TokenPattern
}

func parseLine(line string) []Span {
	allMatches := []Span{}
	for _, p := range allPatterns {
		matches := p.Pattern.FindAllStringIndex(line, -1)
		for _, match := range matches {
			span := Span{
				Start:        match[0],
				End:          match[1],
				Value:        line[match[0]:match[1]],
				Kind:         p.Name,
				tokenPattern: p,
			}

			allMatches = append(allMatches, span)
		}
	}

	slices.SortFunc(allMatches, func(a, b Span) int {
		return a.Start - b.Start
	})

	mergedSpans := []Span{}
	for _, match := range allMatches {
		if len(mergedSpans) == 0 {
			mergedSpans = append(mergedSpans, match)
			continue
		}

		lastSpan := mergedSpans[len(mergedSpans)-1]
		if lastSpan.Start <= match.Start && match.Start < lastSpan.End {
			if match.tokenPattern.Priority > lastSpan.tokenPattern.Priority {
				mergedSpans[len(mergedSpans)-1] = match
			}
		} else {
			mergedSpans = append(mergedSpans, match)
		}
	}

	// fill in gaps of spans as words
	completeSpans := []Span{}
	for i, span := range mergedSpans {
		if i == 0 {
			if span.Start > 0 {
				completeSpans = append(completeSpans, Span{
					Start: 0,
					End:   span.Start,
					Value: line[0:span.Start],
					Kind:  "word",
				})
			}

			completeSpans = append(completeSpans, span)
		} else {
			lastSpan := mergedSpans[i-1]
			if lastSpan.End == span.Start {
				completeSpans = append(completeSpans, span)
			} else {
				completeSpans = append(completeSpans, Span{
					Start: lastSpan.End,
					End:   span.Start,
					Value: line[lastSpan.End:span.Start],
					Kind:  "word",
				})
				completeSpans = append(completeSpans, span)
			}
		}
	}

	// fill in the last word if there is one
	if len(mergedSpans) != 0 {
		lastSpan := mergedSpans[len(mergedSpans)-1]
		if lastSpan.End < len(line)-1 {
			completeSpans = append(completeSpans, Span{
				Start: lastSpan.End,
				End:   len(line),
				Value: line[lastSpan.End:],
				Kind:  "word",
			})
		}
	}

	return completeSpans
}

func ParseCode(code string) [][]Span {
	lineSpans := [][]Span{}
	for _, line := range regexp.MustCompile(`\n`).Split(code, -1) {
		spans := parseLine(line)
		lineSpans = append(lineSpans, spans)
	}

	return lineSpans
}
