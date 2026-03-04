package s3

import (
	"context"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// makeModel builds a Model for unit testing without AWS clients.
// It sets up list, filterInput, and state so that key handlers work correctly.
func makeModel(state panelState, items []list.Item) Model {
	d := s3Delegate{width: 80}
	l := list.New(items, d, 80, 20)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()

	fi := textinput.New()
	fi.Placeholder = "prefix filter"
	fi.Prompt = "Filter: "

	return Model{
		ctx:         context.Background(),
		state:       state,
		list:        l,
		filterInput: fi,
		width:       80,
		height:      24,
	}
}

// bucketItems returns a slice of list.Item bucket entries for testing.
func bucketItems(names ...string) []list.Item {
	items := make([]list.Item, len(names))
	for i, n := range names {
		items[i] = s3Item{kind: kindBucket, name: n}
	}
	return items
}

// slashKey is the KeyMsg for "/".
var slashKey = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}

// enterKey is the KeyMsg for Enter.
var enterKey = tea.KeyMsg{Type: tea.KeyEnter}

// escKey is the KeyMsg for Esc.
var escKey = tea.KeyMsg{Type: tea.KeyEsc}

// --- Tests for / key (filter activation) ---

// TestFilterSlash_BucketList verifies that pressing / when state==panelBucketList
// activates filter mode and saves the current items to savedItems.
func TestFilterSlash_BucketList(t *testing.T) {
	items := bucketItems("alpha", "bravo", "charlie")
	m := makeModel(panelBucketList, items)

	m2, _ := m.Update(slashKey)

	if !m2.filterMode {
		t.Error("expected filterMode=true after pressing / on panelBucketList, got false")
	}
	if len(m2.savedItems) == 0 {
		t.Error("expected savedItems to be populated after pressing / on panelBucketList")
	}
	if len(m2.savedItems) != len(items) {
		t.Errorf("expected %d savedItems, got %d", len(items), len(m2.savedItems))
	}
}

// TestFilterSlash_PrefixList verifies that pressing / when state==panelPrefixList
// still activates filter mode (regression guard).
func TestFilterSlash_PrefixList(t *testing.T) {
	items := []list.Item{s3Item{kind: kindPrefix, name: "logs/"}}
	m := makeModel(panelPrefixList, items)
	m.selectedBucket = "test-bucket"

	m2, _ := m.Update(slashKey)

	if !m2.filterMode {
		t.Error("expected filterMode=true after pressing / on panelPrefixList, got false")
	}
}

// --- Tests for Enter with filterMode (apply filter) ---

// TestFilterEnter_BucketList verifies that Enter with filterMode=true and
// state==panelBucketList applies a client-side substring filter without panicking
// (bucketClient is nil — FetchPrefixCmd must NOT be called).
func TestFilterEnter_BucketList(t *testing.T) {
	items := bucketItems("alpha", "bravo-special", "charlie", "alpha-extra")
	m := makeModel(panelBucketList, items)

	// Manually activate filter mode as the / handler would.
	m.filterMode = true
	m.savedItems = m.list.Items()
	m.filterInput.SetValue("alpha")
	// bucketClient is intentionally nil — any call to FetchPrefixCmd would panic.

	m2, _ := m.Update(enterKey)

	if m2.filterMode {
		t.Error("expected filterMode=false after Enter on panelBucketList, got true")
	}
	if m2.bucketClient != nil {
		t.Error("expected bucketClient to remain nil — FetchPrefixCmd must not be called")
	}

	// Confirm list was narrowed to items containing "alpha".
	got := m2.list.Items()
	if len(got) != 2 {
		t.Errorf("expected 2 items after filtering for 'alpha', got %d", len(got))
	}
	for _, item := range got {
		it, ok := item.(s3Item)
		if !ok {
			t.Fatal("list item is not s3Item")
		}
		if !testContains(it.name, "alpha") {
			t.Errorf("item %q does not contain 'alpha' — should have been filtered out", it.name)
		}
	}
}

// TestFilterEnter_EmptyQuery_BucketList verifies that an empty query returns all saved items
// when state==panelBucketList.
func TestFilterEnter_EmptyQuery_BucketList(t *testing.T) {
	items := bucketItems("alpha", "bravo", "charlie")
	m := makeModel(panelBucketList, items)

	m.filterMode = true
	m.savedItems = m.list.Items()
	m.filterInput.SetValue("")

	m2, _ := m.Update(enterKey)

	got := m2.list.Items()
	if len(got) != 3 {
		t.Errorf("expected all 3 items for empty query, got %d", len(got))
	}
}

// TestFilterEnter_PrefixList verifies that Enter with filterMode=true and
// state==panelPrefixList issues a FetchPrefixCmd (returns a non-nil tea.Cmd).
// The returned Cmd must not be nil — nil would mean server-side fetch was skipped.
func TestFilterEnter_PrefixList(t *testing.T) {
	items := []list.Item{s3Item{kind: kindPrefix, name: "logs/"}}
	m := makeModel(panelPrefixList, items)
	m.selectedBucket = "test-bucket"
	// NOTE: bucketClient is nil; the test only checks that a non-nil Cmd is returned
	// (proof that FetchPrefixCmd code path was taken, not the client-side filter path).

	m.filterMode = true
	m.savedItems = m.list.Items()
	m.filterInput.SetValue("2024")

	_, cmd := m.Update(enterKey)

	if cmd == nil {
		t.Error("expected non-nil tea.Cmd from FetchPrefixCmd path on panelPrefixList, got nil")
	}
}

// --- Tests for Esc (cancel filter) ---

// TestFilterEsc_BucketList verifies that Esc with filterMode=true restores savedItems
// and clears filterMode when state==panelBucketList.
func TestFilterEsc_BucketList(t *testing.T) {
	items := bucketItems("alpha", "bravo", "charlie")
	m := makeModel(panelBucketList, items)

	// Simulate active filter showing fewer items.
	m.filterMode = true
	m.savedItems = m.list.Items()
	filtered := []list.Item{bucketItems("alpha")[0]}
	_ = m.list.SetItems(filtered)

	m2, _ := m.Update(escKey)

	if m2.filterMode {
		t.Error("expected filterMode=false after Esc, got true")
	}
	if m2.savedItems != nil {
		t.Error("expected savedItems to be cleared after Esc")
	}
	got := m2.list.Items()
	if len(got) != 3 {
		t.Errorf("expected 3 items restored after Esc, got %d", len(got))
	}
}

// TestFilterEsc_PrefixList verifies that Esc with filterMode=true restores savedItems
// and clears filterMode when state==panelPrefixList (regression guard).
func TestFilterEsc_PrefixList(t *testing.T) {
	prefixItems := []list.Item{
		s3Item{kind: kindPrefix, name: "logs/"},
		s3Item{kind: kindPrefix, name: "data/"},
	}
	m := makeModel(panelPrefixList, prefixItems)
	m.selectedBucket = "test-bucket"

	m.filterMode = true
	m.savedItems = m.list.Items()
	_ = m.list.SetItems([]list.Item{prefixItems[0]})

	m2, _ := m.Update(escKey)

	if m2.filterMode {
		t.Error("expected filterMode=false after Esc on panelPrefixList, got true")
	}
	got := m2.list.Items()
	if len(got) != 2 {
		t.Errorf("expected 2 items restored after Esc on panelPrefixList, got %d", len(got))
	}
}

// testContains is a local helper to check substring membership.
func testContains(s, substr string) bool {
	if substr == "" {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
