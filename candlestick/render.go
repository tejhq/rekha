package candlestick

import (
	"math"

	"github.com/tejhq/rekha/canvas"
	"github.com/tejhq/rekha/canvas/graph"
	"github.com/tejhq/rekha/canvas/runes"
)

const (
	upperHalf = '▀'
	lowerHalf = '▄'
	wickUp    = '╵'
	wickDown  = '╷'
)

const (
	subNone = iota
	subWick
	subBody
)

func (m *Model) subRows() int { return m.graphH * 2 }

func (m *Model) subRow(p float64) int {
	span := m.viewMax - m.viewMin
	n := m.subRows()
	if span <= 0 || n <= 1 {
		return n - 1
	}
	f := (m.viewMax - p) / span
	r := int(math.Round(f * float64(n-1)))
	if r < 0 {
		r = 0
	}
	if r > n-1 {
		r = n - 1
	}
	return r
}

func candleRune(upper, lower int) rune {
	switch {
	case upper == subBody && lower == subBody:
		return runes.FullBlock
	case upper == subBody:
		return upperHalf
	case lower == subBody:
		return lowerHalf
	case upper == subWick && lower == subWick:
		return runes.LineVertical
	case upper == subWick:
		return wickUp
	case lower == subWick:
		return wickDown
	}
	return runes.Null
}

func (m *Model) drawCandles() {
	top := m.graphTop()
	for i := m.visStart; i < m.visEnd; i++ {
		c := m.candles[i]
		s := m.BullStyle
		if !c.Bull() {
			s = m.BearStyle
		}
		x := m.colOf(i)
		mid := x + m.candleWidth/2
		hi := m.subRow(c.High)
		lo := m.subRow(c.Low)
		bTop := m.subRow(math.Max(c.Open, c.Close))
		bBot := m.subRow(math.Min(c.Open, c.Close))
		kind := func(sub int) int {
			switch {
			case sub >= bTop && sub <= bBot:
				return subBody
			case sub >= hi && sub <= lo:
				return subWick
			}
			return subNone
		}
		for cell := hi / 2; cell <= lo/2; cell++ {
			u, l := kind(cell*2), kind(cell*2+1)
			r := candleRune(u, l)
			if r == runes.Null {
				continue
			}
			y := top + cell
			if u == subBody || l == subBody {
				for k := 0; k < m.candleWidth; k++ {
					m.Canvas.SetCell(canvas.Point{X: x + k, Y: y}, canvas.NewCellWithStyle(r, s))
				}
			} else {
				m.Canvas.SetCell(canvas.Point{X: mid, Y: y}, canvas.NewCellWithStyle(r, s))
			}
		}
	}
}

func (m *Model) drawOverlays() {
	if m.graphW <= 0 || m.graphH <= 0 {
		return
	}
	top := m.graphTop()
	gridH := m.graphH * 4
	span := m.viewMax - m.viewMin
	for _, o := range m.overlays {
		grid := runes.NewPatternDotsGrid(m.graphW*2, gridH)
		var prev *canvas.Point
		any := false
		for i := m.visStart; i < m.visEnd; i++ {
			if i >= len(o.Values) {
				break
			}
			v := o.Values[i]
			if math.IsNaN(v) || v == 0 || span <= 0 {
				prev = nil
				continue
			}
			gx := m.colOf(i)*2 + m.candleWidth
			if gx >= m.graphW*2 {
				gx = m.graphW*2 - 1
			}
			gy := int(math.Round((m.viewMax - v) / span * float64(gridH-1)))
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
			continue
		}
		for y, row := range grid.BraillePatterns() {
			for x, r := range row {
				if r == runes.BrailleBlockOffset {
					continue
				}
				p := canvas.Point{X: x, Y: top + y}
				if cur := m.Canvas.Cell(p).Rune; runes.IsBraillePattern(cur) {
					r = runes.CombineBraillePatterns(cur, r)
				}
				m.Canvas.SetCell(p, canvas.NewCellWithStyle(r, o.Style))
			}
		}
	}
}

func niceStep(raw float64) float64 {
	if raw <= 0 {
		return 1
	}
	exp := math.Floor(math.Log10(raw))
	base := math.Pow(10, exp)
	for _, f := range []float64{1, 2, 2.5, 5, 10} {
		if f*base >= raw {
			return f * base
		}
	}
	return 10 * base
}

func (m *Model) ticks() []float64 {
	span := m.viewMax - m.viewMin
	if span <= 0 || m.graphH <= 0 {
		return nil
	}
	rows := float64(m.graphH - 1)
	if rows < 1 {
		rows = 1
	}
	step := niceStep(span * float64(m.yStep) / rows)
	var out []float64
	for v := math.Ceil(m.viewMin/step) * step; v <= m.viewMax; v += step {
		out = append(out, v)
	}
	return out
}

func (m *Model) drawGridAndTicks() {
	lastRow := -10
	lastPriceRow := -10
	if m.showLast {
		if c, ok := m.Last(); ok && c.Close >= m.viewMin && c.Close <= m.viewMax {
			lastPriceRow = m.priceRow(c.Close)
		}
	}
	for _, v := range m.ticks() {
		row := m.priceRow(v)
		if lastRow >= 0 && lastRow-row < 2 {
			continue
		}
		if d := row - lastPriceRow; d > -2 && d < 2 {
			if m.showGrid {
				for x := 0; x < m.axisX; x++ {
					m.Canvas.SetCell(canvas.Point{X: x, Y: row}, canvas.NewCellWithStyle('┈', m.GridStyle))
				}
			}
			lastRow = row
			continue
		}
		if m.showGrid {
			for x := 0; x < m.axisX; x++ {
				m.Canvas.SetCell(canvas.Point{X: x, Y: row}, canvas.NewCellWithStyle('┈', m.GridStyle))
			}
		}
		m.Canvas.SetStringWithStyle(canvas.Point{X: m.axisX + 1, Y: row}, m.PriceFormatter(v), m.LabelStyle)
		lastRow = row
	}
}
