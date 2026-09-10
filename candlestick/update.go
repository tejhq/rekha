package candlestick

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/tejhq/rekha/canvas"
)

type KeyMap struct {
	Left      key.Binding
	Right     key.Binding
	PageLeft  key.Binding
	PageRight key.Binding
	Latest    key.Binding
	Escape    key.Binding
	ZoomIn    key.Binding
	ZoomOut   key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Left:      key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "prev candle")),
		Right:     key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next candle")),
		PageLeft:  key.NewBinding(key.WithKeys("pgup", "H"), key.WithHelp("PgUp/H", "scroll back")),
		PageRight: key.NewBinding(key.WithKeys("pgdown", "L"), key.WithHelp("PgDn/L", "scroll forward")),
		Latest:    key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("End/G", "latest")),
		Escape:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear cursor")),
		ZoomIn:    key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "wider candles")),
		ZoomOut:   key.NewBinding(key.WithKeys("-"), key.WithHelp("-", "narrower candles")),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		m.handleKey(msg)
	case tea.MouseClickMsg:
		m.handleClick(msg)
	case tea.MouseWheelMsg:
		m.handleWheel(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyPressMsg) {
	k := m.KeyMap
	switch {
	case key.Matches(msg, k.Left):
		m.MoveCursor(-1)
	case key.Matches(msg, k.Right):
		m.MoveCursor(1)
	case key.Matches(msg, k.PageLeft):
		m.Scroll(max(m.capacity()/2, 1))
	case key.Matches(msg, k.PageRight):
		m.Scroll(-max(m.capacity()/2, 1))
	case key.Matches(msg, k.Latest):
		m.ClearCursor()
		m.ScrollToLatest()
	case key.Matches(msg, k.Escape):
		m.ClearCursor()
	case key.Matches(msg, k.ZoomIn):
		m.SetCandleWidth(min(m.candleWidth+2, 7))
	case key.Matches(msg, k.ZoomOut):
		m.SetCandleWidth(max(m.candleWidth-2, 1))
	}
}

func (m *Model) handleClick(msg tea.MouseClickMsg) {
	if m.zoneManager == nil || msg.Mouse().Button != tea.MouseLeft {
		return
	}
	z := m.zoneManager.Get(m.zoneID)
	if !z.InBounds(msg) {
		return
	}
	x, _ := z.Pos(msg)
	m.layout()
	if x < m.pad {
		m.ClearCursor()
		return
	}
	i := m.visStart + (x-m.pad)/m.stride()
	if i >= m.visEnd {
		m.ClearCursor()
		return
	}
	m.SetCursor(i)
}

func (m *Model) handleWheel(msg tea.MouseWheelMsg) {
	if m.zoneManager != nil && !m.zoneManager.Get(m.zoneID).InBounds(msg) {
		return
	}
	switch msg.Mouse().Button {
	case tea.MouseWheelUp:
		m.Scroll(3)
	case tea.MouseWheelDown:
		m.Scroll(-3)
	}
}

func (m Model) View() string {
	m.Draw()
	r := m.Canvas.View()
	if m.zoneManager != nil {
		r = m.zoneManager.Mark(m.zoneID, r)
	}
	return r
}

func (m *Model) Cell(p canvas.Point) canvas.Cell { return m.Canvas.Cell(p) }
