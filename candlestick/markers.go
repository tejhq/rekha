package candlestick

import (
	"sort"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/tejhq/rekha/canvas"
)

type Level struct {
	Name  string
	Price float64
	Label string
	Style lipgloss.Style
	Rune  rune
}

type Marker struct {
	Time  time.Time
	Price float64
	Rune  rune
	Style lipgloss.Style
	Above bool
}

func (m *Model) SetLevel(name string, price float64, label string, s lipgloss.Style) {
	for i := range m.levels {
		if m.levels[i].Name == name {
			m.levels[i].Price, m.levels[i].Label, m.levels[i].Style = price, label, s
			m.dirty = true
			return
		}
	}
	m.levels = append(m.levels, Level{Name: name, Price: price, Label: label, Style: s, Rune: '╌'})
	m.dirty = true
}

func (m *Model) RemoveLevel(name string) {
	for i := range m.levels {
		if m.levels[i].Name == name {
			m.levels = append(m.levels[:i], m.levels[i+1:]...)
			m.dirty = true
			return
		}
	}
}

func (m *Model) ClearLevels() {
	m.levels = m.levels[:0]
	m.dirty = true
}

func (m *Model) Levels() []Level { return m.levels }

func (m *Model) AddMarker(group string, mk Marker) {
	if m.markers == nil {
		m.markers = map[string][]Marker{}
	}
	m.markers[group] = append(m.markers[group], mk)
	m.dirty = true
}

func (m *Model) SetMarkers(group string, mks []Marker) {
	if m.markers == nil {
		m.markers = map[string][]Marker{}
	}
	m.markers[group] = append([]Marker(nil), mks...)
	m.dirty = true
}

func (m *Model) ClearMarkers(group string) {
	delete(m.markers, group)
	m.dirty = true
}

func (m *Model) ClearAllMarkers() {
	m.markers = nil
	m.dirty = true
}

func (m *Model) Markers(group string) []Marker { return m.markers[group] }

func (m *Model) indexAt(t time.Time) int {
	n := len(m.candles)
	if n == 0 || t.Before(m.candles[0].Time) {
		return -1
	}
	i := sort.Search(n, func(i int) bool { return m.candles[i].Time.After(t) })
	return i - 1
}

func (m *Model) drawLevels() {
	for _, lv := range m.levels {
		if lv.Price < m.viewMin || lv.Price > m.viewMax {
			continue
		}
		row := m.priceRow(lv.Price)
		r := lv.Rune
		if r == 0 {
			r = '╌'
		}
		for x := 0; x < m.axisX; x++ {
			m.Canvas.SetCell(canvas.Point{X: x, Y: row}, canvas.NewCellWithStyle(r, lv.Style))
		}
		label := lv.Label
		if label == "" {
			label = m.PriceFormatter(lv.Price)
		}
		m.Canvas.SetStringWithStyle(canvas.Point{X: m.axisX + 1, Y: row}, label, lv.Style)
	}
}

func (m *Model) drawMarkers() {
	top := m.graphTop()
	bottom := top + m.graphH - 1
	for _, mks := range m.markers {
		for _, mk := range mks {
			i := m.indexAt(mk.Time)
			if i < m.visStart || i >= m.visEnd {
				continue
			}
			c := m.candles[i]
			x := m.colOf(i) + m.candleWidth/2
			var row int
			switch {
			case mk.Price != 0:
				row = m.priceRow(mk.Price)
			case mk.Above:
				row = m.priceRow(c.High) - 1
			default:
				row = m.priceRow(c.Low) + 1
			}
			if row < top {
				row = top
			}
			if row > bottom {
				row = bottom
			}
			r := mk.Rune
			if r == 0 {
				r = '◆'
			}
			m.Canvas.SetCell(canvas.Point{X: x, Y: row}, canvas.NewCellWithStyle(r, mk.Style))
		}
	}
}
