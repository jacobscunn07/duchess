package viewport_test

import (
	"fmt"
	"testing"

	"github.com/jacobscunn07/duchess/internal/bubbles/viewport"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	v := viewport.NewViewport(
		80,
		50,
		viewport.WithTitle("zzzzzzzzzzzz"))

	v.SetContent("blahblahblah")

	v.SetWidth(200)
	v.SetHeight(10)
	fmt.Println(v.View())

	assert.Equal(t,
		`╭──────────────╮                                                                                                                                                                                        
│ zzzzzzzzzzzz ├────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
╰──────────────╯                                                                                                                                                                                        
blahblahblah                                                                                                                                                                                            
                                                                                                                                                                                                        
                                                                                                                                                                                                        
                                                                                                                                                                                                        
                                                                                                                                                                                                        
                                                                                                                                                                                                        
                                                                                                                                                                                                        
                                                                                                                                                                                                        
                                                                                                                                                                                                        
                                                                                                                                                                                                        
                                                                                                                                                                                                ╭──────╮
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤ 100% │
                                                                                                                                                                                                ╰──────╯`,
		v.View())
}
