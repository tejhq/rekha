package candlestick

import (
	"fmt"
	"math"

	"charm.land/lipgloss/v2"

	"github.com/tejhq/rekha/canvas"
	"github.com/tejhq/rekha/canvas/graph"
	"github.com/tejhq/rekha/canvas/runes"
)

type SeriesKind int

const (
	SeriesLine SeriesKind = iota
	SeriesBars
)

type PaneSeries struct {
	Name   string
	Kind   SeriesKind
	Values []float64
	Style  lipgloss.Style
	Down   lipgloss.Style
}

type PaneLine struct {
	Value float64
	Style lipgloss.Style
}

type Pane struct {
	Name   string
	Rows   int
	Min    float64
	Max    float64
	Series []PaneSeries
	Lines  []PaneLine
	Format func(float64) string
}

func (m *Model) SetPane(p Pane) {
	if p.Rows < 1 {
		p.Rows = 1
	}
	for i := range m.panes {
		if m.panes[i].Name == p.Name {
			m.panes[i] = p
			m.dirty = true
			return
		}
	}
	m.panes = append(m.panes, p)
	m.dirty = true
}

func (m *Model) RemovePane(name string) {
	for i := range m.panes {
		if m.panes[i].Name == name {
			m.panes = append(m.panes[:i], m.panes[i+1:]...)
			m.dirty = true
			return
		}
	}
}

func (m *Model) Pane(name string) *Pane {
	for i := range m.panes {
		if m.panes[i].Name == name {
			return &m.panes[i]
		}
	}
	return nil
}

func (m *Model) PushPaneValue(pane, series string, v float64) {
	p := m.Pane(pane)
	if p == nil {
		return
	}
	for i := range p.Series {
		if p.Series[i].Name != series {
			continue
		}
		s := &p.Series[i]
		if len(s.Values) >= len(m.candles) && len(s.Values) > 0 {
			s.Values[len(s.Values)-1] = v
		} else {
			s.Values = append(s.Values, v)
		}
		m.dirty = true
		return
	}
}

func (m *Model) paneRows() int {
	n := 0
	for _, p := range m.panes {
		n += p.Rows
	}
	return n
}

func (m *Model) paneTop(k int) int {
	y := m.axisY - m.volumeRows()
	for j := len(m.panes) - 1; j >= k; j-- {
		y -= m.panes[j].Rows
	}
	return y
}

func (m *Model) trimPanes(drop int) {
	for i := range m.panes {
		for j := range m.panes[i].Series {
			s := &m.panes[i].Series[j]
			if len(s.Values) > drop {
				s.Values = s.Values[drop:]
			} else {
				s.Values = s.Values[:0]
			}
		}
	}
}

func (m *Model) clearPanes() {
	for i := range m.panes {
		for j := range m.panes[i].Series {
			m.panes[i].Series[j].Values = m.panes[i].Series[j].Values[:0]
		}
	}
}

func valid(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func (m *Model) paneRange(p *Pane) (float64, float64) {
	if p.Min < p.Max {
		return p.Min, p.Max
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, s := range p.Series {
		for i := m.visStart; i < m.visEnd && i < len(s.Values); i++ {
			if v := s.Values[i]; valid(v) {
				lo, hi = math.Min(lo, v), math.Max(hi, v)
			}
		}
	}
	for _, l := range p.Lines {
		lo, hi = math.Min(lo, l.Value), math.Max(hi, l.Value)
	}
	if math.IsInf(lo, 0) || math.IsInf(hi, 0) {
		return 0, 1
	}
	if hi == lo {
		hi = lo + 1
	}
	pad := (hi - lo) * 0.05
	return lo - pad, hi + pad
}

func (m *Model) drawPanes() {
	for k := range m.panes {
		p := &m.panes[k]
		top := m.paneTop(k)
		rows := p.Rows
		lo, hi := m.paneRange(p)
		span := hi - lo
		rowOf := func(v float64) int {
			f := (hi - v) / span
			r := int(math.Round(f * float64(rows-1)))
			if r < 0 {
				r = 0
			}
			if r > rows-1 {
				r = rows - 1
			}
			return top + r
		}
		format := p.Format
		if format == nil {
			format = m.PriceFormatter
		}
		for _, l := range p.Lines {
			if l.Value < lo || l.Value > hi {
				continue
			}
			y := rowOf(l.Value)
			for x := 0; x < m.axisX; x++ {
				m.Canvas.SetCell(canvas.Point{X: x, Y: y}, canvas.NewCellWithStyle('┈', l.Style))
			}
			m.Canvas.SetStringWithStyle(canvas.Point{X: m.axisX + 1, Y: y}, format(l.Value), m.LabelStyle)
		}
		for _, s := range p.Series {
			switch s.Kind {
			case SeriesBars:
				m.drawPaneBars(p, &s, top, rows, lo, hi)
			default:
				m.drawPaneLine(&s, top, rows, lo, hi)
			}
		}
		label := p.Name
		for _, s := range p.Series {
			if i := m.valueIndex(len(s.Values)); i >= 0 && valid(s.Values[i]) {
				label += fmt.Sprintf("  %s %s", s.Name, format(s.Values[i]))
			}
		}
		m.Canvas.SetStringWithStyle(canvas.Point{X: 1, Y: top}, label, m.LabelStyle)
	}
}

func (m *Model) readoutIndex() int {
	if m.cursor >= 0 && m.cursor < len(m.candles) {
		return m.cursor
	}
	return len(m.candles) - 1
}

func (m *Model) valueIndex(n int) int {
	if n == 0 {
		return -1
	}
	if m.cursor >= 0 {
		if m.cursor < n {
			return m.cursor
		}
		return -1
	}
	return n - 1
}

func (m *Model) drawPaneLine(s *PaneSeries, top, rows int, lo, hi float64) {
	gridH := rows * 4
	grid := runes.NewPatternDotsGrid(m.graphW*2, gridH)
	var prev *canvas.Point
	any := false
	span := hi - lo
	for i := m.visStart; i < m.visEnd && i < len(s.Values); i++ {
		v := s.Values[i]
		if !valid(v) {
			prev = nil
			continue
		}
		gx := m.colOf(i)*2 + m.candleWidth
		if gx >= m.graphW*2 {
			gx = m.graphW*2 - 1
		}
		gy := int(math.Round((hi - v) / span * float64(gridH-1)))
		if gy < 0 {
			gy = 0
		}
		if gy > gridH-1 {
			gy = gridH - 1
		}
		p := canvas.Point{X: gx, Y: gy}
		if prev != nil {
			for _, lp := range graph.GetLinePoints(*prev, p) {
				grid.Set(lp.X, lp.Y)
			}
		} else {
			grid.Set(p.X, p.Y)
		}
		prev = &p
		any = true
	}
	if !any {
		return
	}
	for y, row := range grid.BraillePatterns() {
		for x, r := range row {
			if r == runes.BrailleBlockOffset {
				continue
			}
			pt := canvas.Point{X: x, Y: top + y}
			if cur := m.Canvas.Cell(pt).Rune; runes.IsBraillePattern(cur) {
				r = runes.CombineBraillePatterns(cur, r)
			}
			m.Canvas.SetCell(pt, canvas.NewCellWithStyle(r, s.Style))
		}
	}
}

func (m *Model) drawPaneBars(p *Pane, s *PaneSeries, top, rows int, lo, hi float64) {
	span := hi - lo
	base := 0.0
	if lo > 0 {
		base = lo
	}
	if hi < 0 {
		base = hi
	}
	sub := rows * 2
	subOf := func(v float64) int {
		r := int(math.Round((hi - v) / span * float64(sub-1)))
		if r < 0 {
			r = 0
		}
		if r > sub-1 {
			r = sub - 1
		}
		return r
	}
	baseSub := subOf(base)
	for i := m.visStart; i < m.visEnd && i < len(s.Values); i++ {
		v := s.Values[i]
		if !valid(v) {
			continue
		}
		st := s.Style
		if v < base {
			st = s.Down
		}
		a, b := subOf(v), baseSub
		if a > b {
			a, b = b, a
		}
		x := m.colOf(i)
		for cell := a / 2; cell <= b/2; cell++ {
			uIn := cell*2 >= a && cell*2 <= b
			lIn := cell*2+1 >= a && cell*2+1 <= b
			var r rune
			switch {
			case uIn && lIn:
				r = runes.FullBlock
			case uIn:
				r = upperHalf
			case lIn:
				r = lowerHalf
			default:
				continue
			}
			for k := 0; k < m.candleWidth; k++ {
				m.Canvas.SetCell(canvas.Point{X: x + k, Y: top + cell}, canvas.NewCellWithStyle(r, st))
			}
		}
	}
	_ = p
}
