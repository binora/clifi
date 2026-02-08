package cli

import (
	"fmt"
	"strings"

	"github.com/yolodolo42/clifi/internal/agent"
)

func renderBlocks(width int, blocks []agent.UIBlock) string {
	if len(blocks) == 0 {
		return ""
	}

	var b strings.Builder
	for i, blk := range blocks {
		if i > 0 {
			b.WriteString("\n")
		}
		switch blk.Kind {
		case agent.UIBlockTable:
			if blk.Table != nil {
				b.WriteString(renderTable(width, blk.Table))
			}
		case agent.UIBlockKV:
			if blk.KV != nil {
				b.WriteString(renderKV(width, blk.KV))
			}
		default:
			// Unknown block: ignore to keep rendering robust.
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderKV(width int, kv *agent.UIKV) string {
	var b strings.Builder
	if kv.Title != "" {
		b.WriteString(kv.Title)
		b.WriteString("\n")
	}
	maxKey := 0
	for _, it := range kv.Items {
		if len(it.Key) > maxKey {
			maxKey = len(it.Key)
		}
	}
	if maxKey > 24 {
		maxKey = 24
	}

	for _, it := range kv.Items {
		key := it.Key
		if len(key) > maxKey {
			key = key[:maxKey]
		}
		line := fmt.Sprintf("%-*s  %s", maxKey, key, it.Value)
		b.WriteString(truncate(line, width))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderTable(width int, t *agent.UITable) string {
	cols := len(t.Headers)
	if cols == 0 {
		return ""
	}

	colW := make([]int, cols)
	for c := 0; c < cols; c++ {
		colW[c] = len(t.Headers[c])
	}
	for _, row := range t.Rows {
		for c := 0; c < cols && c < len(row); c++ {
			if l := len(row[c]); l > colW[c] {
				colW[c] = l
			}
		}
	}

	// Clamp to keep within width (best effort; shrink last columns first).
	sep := 3 // " | "
	avail := width
	if avail < 20 {
		avail = 20
	}
	for totalWidth(colW, sep) > avail {
		shrunk := false
		for c := cols - 1; c >= 0; c-- {
			if colW[c] > 6 {
				colW[c]--
				shrunk = true
				break
			}
		}
		if !shrunk {
			break
		}
	}

	var b strings.Builder
	if t.Title != "" {
		b.WriteString(t.Title)
		b.WriteString("\n")
	}

	b.WriteString(renderTableRow(t.Headers, colW, sep))
	b.WriteString("\n")
	b.WriteString(renderTableSep(colW, sep))
	b.WriteString("\n")
	for i, row := range t.Rows {
		b.WriteString(renderTableRow(row, colW, sep))
		if i < len(t.Rows)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func totalWidth(colW []int, sep int) int {
	total := 0
	for _, w := range colW {
		total += w
	}
	total += sep * (len(colW) - 1)
	return total
}

func renderTableSep(colW []int, sep int) string {
	var b strings.Builder
	for c, w := range colW {
		if c > 0 {
			b.WriteString(strings.Repeat("-", sep))
		}
		b.WriteString(strings.Repeat("-", w))
	}
	return b.String()
}

func renderTableRow(cells []string, colW []int, sep int) string {
	var b strings.Builder
	for c, w := range colW {
		if c > 0 {
			b.WriteString(" | ")
		}
		val := ""
		if c < len(cells) {
			val = cells[c]
		}
		b.WriteString(padRight(truncate(val, w), w))
	}
	return b.String()
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func truncate(s string, w int) string {
	if w <= 0 || len(s) <= w {
		return s
	}
	if w <= 3 {
		return s[:w]
	}
	return s[:w-3] + "..."
}
