package bubbles_test

import (
	"testing"

	"github.com/jacobscunn07/duchess/internal/bubbles"
	"github.com/stretchr/testify/assert"
)

func TestNewViewport(t *testing.T) {
	v := bubbles.NewViewport(
		5,
		5,
		bubbles.WithHeight(0))

	v.SetContent("abc")
	// fmt.Println(v.View())

	assert.Equal(t, "xyz", v.View())
}
