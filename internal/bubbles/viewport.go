package bubbles

import "github.com/charmbracelet/bubbles/viewport"

func NewViewport(width, height int, options ...func(*viewport.Model)) viewport.Model {

	var ()

	DefaultStyles := func(v *viewport.Model) {

	}

	v := viewport.New(width, height)

	DefaultStyles(&v)

	for _, o := range options {
		o(&v)
	}

	return v
}

func WithHeight(height int) func(*viewport.Model) {
	return func(m *viewport.Model) {}
}
