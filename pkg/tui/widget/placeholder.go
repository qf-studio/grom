package widget

import (
	"github.com/qf-studio/grom/pkg/tui/render"
	"github.com/qf-studio/grom/pkg/tui/theme"
)

// Placeholder fills a panel slot grom cannot render — an unsupported Grafana
// panel type — with a quiet centered note, so an imported dashboard keeps its
// layout instead of leaving a hole.
type Placeholder struct {
	data
	title string
	msg   string
}

// NewPlaceholder creates a placeholder panel showing msg.
func NewPlaceholder(title, msg string) *Placeholder {
	return &Placeholder{title: title, msg: msg}
}

func (p *Placeholder) Title() string       { return p.title }
func (p *Placeholder) MinSize() (int, int) { return 12, 3 }

func (p *Placeholder) Render(w, h int, th theme.Theme, focused bool) string {
	iw, ih := render.InnerSize(w, h)
	ps := p.panelStyle(th, focused)
	body := vCenter(th.DimMoreStyle().Render(render.Center(p.msg, iw)), iw, ih)
	return render.Panel(p.title, body, w, h, ps)
}
