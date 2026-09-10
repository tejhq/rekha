package candlestick

import (
	"math"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/tejhq/rekha/canvas"
	"github.com/tejhq/rekha/canvas/runes"
)

func mk(i int, o, h, l, c, v float64) Candle {
	return Candle{
		Time: time.Date(2026, 4, 8, 9, 15+i, 0, 0, time.UTC),
		Open: o, High: h, Low: l, Close: c, Volume: v,
	}
}

func series(n int) []Candle {
	cs := make([]Candle, 0, n)
	p := 100.0
	for i := 0; i < n; i++ {
		d := math.Sin(float64(i)/3) * 2
		o := p
		c := p + d
		cs = append(cs, mk(i, o, math.Max(o, c)+1, math.Min(o, c)-1, c, 1000+float64(i*50)))
		p = c
	}
	return cs
}

func hasRune(m *Model, r rune) bool {
	for y := 0; y < m.Height(); y++ {
		for x := 0; x < m.Width(); x++ {
			if m.Cell(canvas.Point{X: x, Y: y}).Rune == r {
				return true
			}
		}
	}
	return false
}

func TestPushAndReplaceSameTime(t *testing.T) {
	m := New(40, 12)
	m.Push(mk(0, 1, 2, 0.5, 1.5, 10))
	m.Push(mk(0, 1, 3, 0.5, 2.5, 20))
	if m.Len() != 1 {
		t.Fatalf("len %d, want 1", m.Len())
	}
	if c, _ := m.Last(); c.High != 3 {
		t.Fatalf("high %v, want 3", c.High)
	}
}

func TestMaxCandlesTrimsOverlays(t *testing.T) {
	m := New(40, 12, WithMaxCandles(5))
	m.SetOverlay("ema", nil, lipgloss.NewStyle())
	for i := 0; i < 8; i++ {
		m.Push(mk(i, 1, 2, 0.5, 1.5, 10))
		m.PushOverlay("ema", float64(i))
	}
	if m.Len() != 5 {
		t.Fatalf("len %d, want 5", m.Len())
	}
	o := m.overlay("ema")
	if len(o.Values) != 5 || o.Values[0] != 3 {
		t.Fatalf("overlay %v", o.Values)
	}
}

func TestDrawCandleBodyAndWick(t *testing.T) {
	m := New(30, 12, WithVolume(0), WithReadout(false))
	m.Push(mk(0, 100, 110, 90, 105, 10))
	m.Push(mk(1, 105, 115, 95, 100, 10))
	m.Draw()
	if !hasRune(&m, runes.FullBlock) {
		t.Fatal("no candle body drawn")
	}
	if !hasRune(&m, runes.LineVertical) {
		t.Fatal("no wick or axis drawn")
	}
	closeRow := m.priceRow(105)
	c := m.Cell(canvas.Point{X: 0, Y: closeRow})
	if c.Rune != runes.FullBlock {
		t.Fatalf("expected body at close row, got %q", c.Rune)
	}
}

func TestPriceLabelsOnRightAxis(t *testing.T) {
	m := New(40, 12, WithVolume(0), WithReadout(false), WithLastPrice(false))
	m.SetCandles(series(10))
	m.Draw()
	found := false
	for y := 0; y < m.Height(); y++ {
		if m.Cell(canvas.Point{X: m.axisX + 1, Y: y}).Rune != runes.Null {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no price labels right of axis")
	}
	if m.Cell(canvas.Point{X: m.axisX, Y: m.axisY}).Rune != runes.LineUpLeft {
		t.Fatal("axis corner missing")
	}
}

func TestVisibleWindowAndScroll(t *testing.T) {
	m := New(30, 12, WithVolume(0), WithReadout(false))
	m.SetCandles(series(100))
	s, e := m.Visible()
	if e != 100 || e-s != m.capacity() {
		t.Fatalf("visible %d..%d cap %d", s, e, m.capacity())
	}
	m.Scroll(10)
	s2, e2 := m.Visible()
	if e2 != 90 || s2 != s-10 {
		t.Fatalf("after scroll visible %d..%d", s2, e2)
	}
	m.Scroll(1000)
	s3, _ := m.Visible()
	if s3 != 0 {
		t.Fatalf("scroll clamp start %d", s3)
	}
	m.ScrollToLatest()
	if _, e4 := m.Visible(); e4 != 100 {
		t.Fatalf("latest end %d", e4)
	}
}

func TestPushWhileScrolledKeepsView(t *testing.T) {
	m := New(30, 12)
	m.SetCandles(series(60))
	m.Scroll(5)
	s, _ := m.Visible()
	m.Push(mk(60, 1, 2, 0.5, 1.5, 1))
	s2, _ := m.Visible()
	if s2 != s {
		t.Fatalf("view moved %d -> %d", s, s2)
	}
}

func TestCursorMovesAndScrolls(t *testing.T) {
	m := New(30, 12)
	m.SetCandles(series(60))
	m.Focus()
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	_, i, ok := m.Cursor()
	if !ok || i != 59 {
		t.Fatalf("cursor %d ok=%v", i, ok)
	}
	for k := 0; k < 40; k++ {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	}
	_, i, _ = m.Cursor()
	s, _ := m.Visible()
	if i != 19 || s > 19 {
		t.Fatalf("cursor %d visStart %d", i, s)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if _, _, ok := m.Cursor(); ok {
		t.Fatal("cursor should clear")
	}
	m.Draw()
	if hasRune(&m, '┊') {
		t.Fatal("crosshair drawn without cursor")
	}
	m.SetCursor(59)
	m.Draw()
	if !hasRune(&m, '┊') {
		t.Fatal("crosshair missing")
	}
}

func TestOverlayDrawn(t *testing.T) {
	st := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	m := New(40, 14, WithVolume(0), WithReadout(false), WithLastPrice(false))
	cs := series(20)
	m.SetCandles(cs)
	vals := make([]float64, len(cs))
	for i := range cs {
		vals[i] = cs[i].Close + 0.5
	}
	m.SetOverlay("ema9", vals, st)
	m.Draw()
	if !hasRune(&m, '╱') && !hasRune(&m, '╲') && !hasRune(&m, runes.LineHorizontal) {
		t.Fatal("overlay not drawn")
	}
}

func TestVolumePane(t *testing.T) {
	m := New(40, 14, WithVolume(3), WithReadout(false))
	m.SetCandles(series(20))
	m.Draw()
	base := m.axisY - 1
	found := false
	for x := 0; x < m.axisX; x++ {
		if r := m.Cell(canvas.Point{X: x, Y: base}).Rune; runes.IsLowerBlockElement(r) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no volume bars")
	}
}

func TestReadoutAndLastPrice(t *testing.T) {
	m := New(60, 14)
	m.SetCandles(series(20))
	m.Draw()
	var row strings.Builder
	for x := 0; x < m.Width(); x++ {
		row.WriteRune(m.Cell(canvas.Point{X: x, Y: 0}).Rune)
	}
	if !strings.Contains(row.String(), "O ") || !strings.Contains(row.String(), "V ") {
		t.Fatalf("readout missing: %q", row.String())
	}
	if !hasRune(&m, '┄') {
		t.Fatal("last price line missing")
	}
}

func TestCandleWidth(t *testing.T) {
	m := New(40, 12, WithCandleWidth(3), WithVolume(0), WithReadout(false))
	m.SetCandles(series(5))
	m.Draw()
	row := m.priceRow(m.candles[0].Close)
	for k := 0; k < 3; k++ {
		if m.Cell(canvas.Point{X: k, Y: row}).Rune != runes.FullBlock {
			t.Fatalf("body col %d not filled", k)
		}
	}
	if m.capacity() != m.graphW/3 {
		t.Fatalf("capacity %d", m.capacity())
	}
}

func TestEmptyDrawDoesNotPanic(t *testing.T) {
	m := New(20, 6)
	_ = m.View()
	m.Resize(3, 2)
	_ = m.View()
}
