package s3

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"

	"github.com/jacobscunn07/duchess/internal/ui/theme"
)

// itemKind distinguishes the three kinds of S3 items displayed in the list.
type itemKind int

const (
	kindBucket itemKind = iota
	kindPrefix          // virtual folder with trailing /
	kindObject          // real S3 object
)

// s3Item implements list.Item and carries all data needed to render and navigate
// a single row in the S3 browser.
type s3Item struct {
	kind         itemKind
	name         string     // display name (bucket name, prefix path, or object key basename)
	fullKey      string     // full S3 key path (for object navigation)
	size         *int64     // nil for buckets and prefixes
	lastModified *time.Time // nil for buckets and prefixes
	storageClass string     // e.g. "STANDARD", "GLACIER" (empty for buckets/prefixes)
	etag         string     // raw ETag value (empty for buckets/prefixes)
}

// FilterValue implements list.Item.
func (i s3Item) FilterValue() string { return i.name }

// DisplayName returns the name shown in the left column.
// Prefixes display with trailing / to distinguish from objects.
func (i s3Item) DisplayName() string {
	return i.name
}

// SizeAndDate returns the right-column string for list rows.
// Returns "" for buckets (no size/date known from ListBuckets).
// Returns "DIR" for prefixes (virtual folder, no size).
// Returns "{size}  {relativeTime}" for objects.
func (i s3Item) SizeAndDate() string {
	switch i.kind {
	case kindBucket:
		return ""
	case kindPrefix:
		return "DIR"
	case kindObject:
		if i.size == nil || i.lastModified == nil {
			return ""
		}
		return humanize.IBytes(uint64(*i.size)) + "  " + humanize.Time(*i.lastModified)
	}
	return ""
}

// s3Delegate implements list.ItemDelegate to render single-line two-column rows.
type s3Delegate struct {
	width int
}

var (
	normalStyle   = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle = lipgloss.NewStyle().PaddingLeft(1).Foreground(theme.DefaultTheme.Accent).Bold(true)
	prefixStyle   = lipgloss.NewStyle().Foreground(theme.DefaultTheme.Accent)
)

func (d s3Delegate) Height() int                             { return 1 }
func (d s3Delegate) Spacing() int                            { return 0 }
func (d s3Delegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d s3Delegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(s3Item)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	left := i.DisplayName()
	right := i.SizeAndDate()

	// Truncate very long keys: if left alone would overflow, truncate with ...
	maxLeft := d.width - lipgloss.Width(right) - 4
	if lipgloss.Width(left) > maxLeft && maxLeft > 3 {
		// Truncate from right (show beginning of key, more informative than end)
		left = left[:maxLeft-3] + "..."
	}

	gap := d.width - lipgloss.Width(left) - lipgloss.Width(right) - 2 // -2 for left padding
	if gap < 1 {
		gap = 1
	}

	row := left + strings.Repeat(" ", gap) + right

	var style lipgloss.Style
	if isSelected {
		style = selectedStyle
	} else if i.kind == kindPrefix {
		style = prefixStyle.PaddingLeft(2)
	} else {
		style = normalStyle
	}

	fmt.Fprint(w, style.Render(row))
}

// bucketsToItems converts a slice of S3 bucket types to list.Item slice.
func bucketsToItems(buckets []types.Bucket) []list.Item {
	items := make([]list.Item, len(buckets))
	for idx, b := range buckets {
		name := ""
		if b.Name != nil {
			name = *b.Name
		}
		items[idx] = s3Item{
			kind:    kindBucket,
			name:    name,
			fullKey: name,
		}
	}
	return items
}

// prefixesToItems converts raw prefix strings and S3 object types to list.Item slice.
// Prefixes (virtual folders) appear first, followed by objects.
func prefixesToItems(prefixes []string, objects []types.Object) []list.Item {
	items := make([]list.Item, 0, len(prefixes)+len(objects))
	for _, p := range prefixes {
		items = append(items, s3Item{
			kind:    kindPrefix,
			name:    p,
			fullKey: p,
		})
	}
	for _, obj := range objects {
		key := ""
		if obj.Key != nil {
			key = *obj.Key
		}
		// Display the basename of the key (last segment after final /)
		name := key
		if idx := strings.LastIndex(key, "/"); idx >= 0 && idx < len(key)-1 {
			name = key[idx+1:]
		}
		storageClass := ""
		if obj.StorageClass != "" {
			storageClass = string(obj.StorageClass)
		}
		etag := ""
		if obj.ETag != nil {
			etag = *obj.ETag
		}
		items = append(items, s3Item{
			kind:         kindObject,
			name:         name,
			fullKey:      key,
			size:         obj.Size,
			lastModified: obj.LastModified,
			storageClass: storageClass,
			etag:         etag,
		})
	}
	return items
}
