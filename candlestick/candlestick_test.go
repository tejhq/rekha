package candlestick

import (
	"math"
	"strconv"
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

func isBody(r rune) bool { return r == runes.FullBlock || r == upperHalf || r == lowerHalf }

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
	if !hasRune(&m, runes.FullBlock) && !hasRune(&m, upperHalf) && !hasRune(&m, lowerHalf) {
		t.Fatal("no candle body drawn")
	}
	closeRow := m.priceRow(105)
	c := m.Cell(canvas.Point{X: m.colOf(0), Y: closeRow})
	if !isBody(c.Rune) {
		t.Fatalf("expected body at close row, got %q", c.Rune)
	}
	wick := m.Cell(canvas.Point{X: m.colOf(0), Y: m.priceRow(110)}).Rune
	if wick != runes.LineVertical && wick != wickUp && wick != wickDown && !isBody(wick) {
		t.Fatalf("expected wick at high, got %q", wick)
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
		vals[i] = cs[i].High + 3
	}
	m.SetOverlay("ema9", vals, st)
	m.Draw()
	found := false
	for y := 0; y < m.Height(); y++ {
		for x := 0; x < m.axisX; x++ {
			if runes.IsBraillePattern(m.Cell(canvas.Point{X: x, Y: y}).Rune) {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("overlay not drawn as braille")
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
		if !isBody(m.Cell(canvas.Point{X: m.colOf(0) + k, Y: row}).Rune) {
			t.Fatalf("body col %d not filled", k)
		}
	}
	if m.capacity() != m.graphW/3 {
		t.Fatalf("capacity %d", m.capacity())
	}
}

func TestResizeGrowsView(t *testing.T) {
	m := New(20, 5, WithReadout(false), WithVolume(0))
	m.SetCandles(series(30))
	m.Resize(60, 15)
	v := m.View()
	lines := strings.Split(v, "\n")
	if len(lines) != 15 {
		t.Fatalf("rows %d, want 15", len(lines))
	}
	if w := lipgloss.Width(lines[0]); w != 60 {
		t.Fatalf("width %d, want 60", w)
	}
}

func TestDailyLabels(t *testing.T) {
	m := New(80, 16, WithVolume(0))
	var cs []Candle
	day := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		cs = append(cs, Candle{Time: day.AddDate(0, 0, i*3), Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 1})
	}
	m.SetCandles(cs)
	m.Draw()
	var top, bottom strings.Builder
	for x := 0; x < m.Width(); x++ {
		top.WriteRune(m.Cell(canvas.Point{X: x, Y: 0}).Rune)
		bottom.WriteRune(m.Cell(canvas.Point{X: x, Y: m.Height() - 1}).Rune)
	}
	if !strings.Contains(top.String(), "2026") {
		t.Fatalf("daily readout should show year: %q", top.String())
	}
	if !strings.Contains(bottom.String(), "Mar") || strings.Contains(bottom.String(), "Mar 26") {
		t.Fatalf("labels %q", bottom.String())
	}
}

func TestLevelsDrawnAndExpandRange(t *testing.T) {
	st := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	m := New(50, 16, WithVolume(0), WithReadout(false), WithLastPrice(false))
	m.SetCandles(series(20))
	m.SetLevel("stop", 80, "S 80", st)
	m.Draw()
	lo, _ := m.ViewRange()
	if lo <= 80 {
		t.Fatalf("range should not stretch to level, min %v", lo)
	}
	bottom := m.graphTop() + m.graphH - 1
	var edge strings.Builder
	for x := m.axisX + 1; x < m.Width(); x++ {
		edge.WriteRune(m.Cell(canvas.Point{X: x, Y: bottom}).Rune)
	}
	if !strings.HasPrefix(edge.String(), "▼S 80") {
		t.Fatalf("edge label %q", edge.String())
	}
	m.RemoveLevel("stop")
	mid := (lo + m.viewMax) / 2
	m.SetLevel("stop", mid, "S mid", st)
	m.Draw()
	row := m.priceRow(mid)
	if m.Cell(canvas.Point{X: 0, Y: row}).Rune != '╌' {
		t.Fatal("level line missing")
	}
	var label strings.Builder
	for x := m.axisX + 1; x < m.Width(); x++ {
		label.WriteRune(m.Cell(canvas.Point{X: x, Y: row}).Rune)
	}
	if !strings.HasPrefix(label.String(), "S mid") {
		t.Fatalf("label %q", label.String())
	}
	m.SetLevel("entry", mid, "E 100.25 long", st)
	m.Draw()
	label.Reset()
	for x := m.axisX + 1; x < m.Width(); x++ {
		label.WriteRune(m.Cell(canvas.Point{X: x, Y: m.priceRow(mid)}).Rune)
	}
	if !strings.HasPrefix(label.String(), "E 100.25 long") {
		t.Fatalf("wide label clipped: %q", label.String())
	}
	m.RemoveLevel("entry")
	m.RemoveLevel("stop")
	m.Draw()
	if hasRune(&m, '╌') {
		t.Fatal("level should be removed")
	}
}

func TestMarkersPlacedAtCandle(t *testing.T) {
	st := lipgloss.NewStyle()
	m := New(50, 16, WithVolume(0), WithReadout(false), WithLastPrice(false))
	cs := series(20)
	m.SetCandles(cs)
	m.AddMarker("fills", Marker{Time: cs[5].Time.Add(20 * time.Second), Rune: '▲', Style: st})
	m.AddMarker("alerts", Marker{Time: cs[7].Time, Rune: '▽', Style: st, Above: true})
	m.AddMarker("old", Marker{Time: cs[0].Time.Add(-time.Hour), Rune: 'X', Style: st})
	m.Draw()
	if m.Cell(canvas.Point{X: m.colOf(5), Y: m.priceRow(cs[5].Low) + 1}).Rune != '▲' {
		t.Fatal("buy marker not below candle 5")
	}
	if m.Cell(canvas.Point{X: m.colOf(7), Y: m.priceRow(cs[7].High) - 1}).Rune != '▽' {
		t.Fatal("alert marker not above candle 7")
	}
	if hasRune(&m, 'X') {
		t.Fatal("marker before first candle should be skipped")
	}
	m.ClearMarkers("fills")
	m.Draw()
	if hasRune(&m, '▲') {
		t.Fatal("cleared group still drawn")
	}
	m.Clear()
	if len(m.Markers("alerts")) != 0 {
		t.Fatal("Clear should drop markers")
	}
}

func TestRepeatedTimeLabelsSuppressed(t *testing.T) {
	m := New(80, 14, WithVolume(0), WithReadout(false))
	var cs []Candle
	base := time.Date(2026, 4, 9, 9, 15, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		cs = append(cs, Candle{Time: base.Add(time.Duration(i) * time.Minute), Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 1})
	}
	cs = append(cs, Candle{Time: base.AddDate(0, 5, 0), Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 1})
	m.SetCandles(cs)
	m.Draw()
	var row strings.Builder
	for x := 0; x < m.Width(); x++ {
		row.WriteRune(m.Cell(canvas.Point{X: x, Y: m.Height() - 1}).Rune)
	}
	if strings.Count(row.String(), "09 Apr") != 1 {
		t.Fatalf("labels %q", row.String())
	}
}

func TestFewCandlesRightAligned(t *testing.T) {
	m := New(40, 12, WithVolume(0), WithReadout(false), WithLastPrice(false))
	m.SetCandles(series(5))
	m.Draw()
	lastCol := m.graphW - 1
	if !isBody(m.Cell(canvas.Point{X: lastCol, Y: m.priceRow(m.candles[4].Close)}).Rune) {
		t.Fatal("last candle should sit at the right edge")
	}
	if isBody(m.Cell(canvas.Point{X: 0, Y: m.priceRow(m.candles[0].Close)}).Rune) {
		t.Fatal("first candle should not be at column 0")
	}
	m.SetCursor(4)
	m.Draw()
	found := false
	for y := m.graphTop(); y < m.axisY; y++ {
		if m.Cell(canvas.Point{X: lastCol, Y: y}).Rune == '┊' {
			found = true
		}
	}
	if !found {
		t.Fatal("crosshair should follow padded column")
	}
}

func TestHalfBlockBodies(t *testing.T) {
	m := New(30, 12, WithVolume(0), WithReadout(false), WithLastPrice(false), WithGrid(false))
	m.Push(mk(0, 100, 120, 80, 100.5, 1))
	m.Push(mk(1, 100, 120, 80, 100, 1))
	m.Draw()
	half := hasRune(&m, upperHalf) || hasRune(&m, lowerHalf)
	if !half {
		t.Fatal("tiny bodies should use half blocks")
	}
}

func TestNiceTicks(t *testing.T) {
	for raw, want := range map[float64]float64{0.3: 0.5, 7: 10, 12: 20, 23: 25, 40: 50, 120: 200} {
		if got := niceStep(raw); got != want {
			t.Fatalf("niceStep(%v)=%v want %v", raw, got, want)
		}
	}
	m := New(60, 20, WithVolume(0), WithReadout(false), WithLastPrice(false))
	m.SetCandles(series(30))
	m.Draw()
	var labels []string
	for y := 0; y < m.Height(); y++ {
		var b strings.Builder
		for x := m.axisX + 1; x < m.Width(); x++ {
			b.WriteRune(m.Cell(canvas.Point{X: x, Y: y}).Rune)
		}
		if l := strings.TrimRight(strings.ReplaceAll(b.String(), "\x00", ""), " "); l != "" {
			labels = append(labels, l)
		}
	}
	if len(labels) < 3 {
		t.Fatalf("labels %v", labels)
	}
	for _, l := range labels {
		v, err := strconv.ParseFloat(l, 64)
		if err != nil || math.Mod(v, 0.5) != 0 {
			t.Fatalf("tick %q not on a nice boundary", l)
		}
	}
	if !hasRune(&m, '┈') {
		t.Fatal("grid missing")
	}
}

func TestPanesLayoutAndValues(t *testing.T) {
	m := New(60, 24, WithVolume(2), WithReadout(false), WithLastPrice(false), WithGrid(false))
	cs := series(30)
	m.SetCandles(cs)
	rsi := make([]float64, len(cs))
	hist := make([]float64, len(cs))
	for i := range cs {
		rsi[i] = 30 + float64(i%40)
		hist[i] = math.Sin(float64(i) / 4)
	}
	dim := lipgloss.NewStyle()
	m.SetPane(Pane{Name: "RSI", Rows: 4, Min: 0, Max: 100,
		Series: []PaneSeries{{Name: "rsi", Values: rsi, Style: dim}},
		Lines:  []PaneLine{{Value: 30, Style: dim}, {Value: 70, Style: dim}}})
	m.SetPane(Pane{Name: "MACD", Rows: 4,
		Series: []PaneSeries{{Name: "hist", Kind: SeriesBars, Values: hist, Style: dim, Down: dim}}})
	m.Draw()
	if m.graphH != m.axisY-2-8 {
		t.Fatalf("graphH %d, axisY %d", m.graphH, m.axisY)
	}
	if m.paneTop(0) != m.axisY-2-8 || m.paneTop(1) != m.axisY-2-4 {
		t.Fatalf("pane tops %d %d", m.paneTop(0), m.paneTop(1))
	}
	var label strings.Builder
	for x := 1; x < 20; x++ {
		label.WriteRune(m.Cell(canvas.Point{X: x, Y: m.paneTop(0)}).Rune)
	}
	if !strings.HasPrefix(label.String(), "RSI  rsi ") {
		t.Fatalf("pane label %q", label.String())
	}
	braille, bars := false, false
	for y := m.paneTop(0); y < m.axisY; y++ {
		for x := 0; x < m.axisX; x++ {
			r := m.Cell(canvas.Point{X: x, Y: y}).Rune
			if runes.IsBraillePattern(r) {
				braille = true
			}
			if y >= m.paneTop(1) && isBody(r) {
				bars = true
			}
		}
	}
	if !braille || !bars {
		t.Fatalf("braille %v bars %v", braille, bars)
	}
	m.PushPaneValue("RSI", "rsi", 55)
	if p := m.Pane("RSI"); p.Series[0].Values[len(p.Series[0].Values)-1] != 55 {
		t.Fatal("push should replace value for current candle")
	}
	m.Push(mk(30, 1, 2, 0.5, 1.5, 1))
	m.PushPaneValue("RSI", "rsi", 66)
	if p := m.Pane("RSI"); len(p.Series[0].Values) != 31 {
		t.Fatalf("values %d", len(p.Series[0].Values))
	}
	m.RemovePane("MACD")
	m.Draw()
	if m.graphH != m.axisY-2-4 {
		t.Fatalf("graphH after remove %d", m.graphH)
	}
}

func TestAutoWidth(t *testing.T) {
	m := New(60, 14, WithAutoWidth(true), WithVolume(0), WithReadout(false))
	m.SetCandles(series(5))
	m.Draw()
	if m.CandleWidth() != 3 {
		t.Fatalf("width %d, want 3", m.CandleWidth())
	}
	m.SetCandles(series(20))
	m.Draw()
	if m.CandleWidth() != 1 || m.gap != 1 {
		t.Fatalf("width %d gap %d, want 1 1", m.CandleWidth(), m.gap)
	}
	m.SetCandles(series(50))
	m.Draw()
	if m.CandleWidth() != 1 || m.gap != 0 {
		t.Fatalf("width %d gap %d, want 1 0", m.CandleWidth(), m.gap)
	}
	m.SetCandleWidth(5)
	m.SetCandles(series(5))
	m.Draw()
	if m.CandleWidth() != 5 {
		t.Fatal("manual width should disable auto")
	}
}

func TestReadoutShowsOverlays(t *testing.T) {
	m := New(90, 14, WithVolume(0))
	cs := series(10)
	m.SetCandles(cs)
	vals := make([]float64, len(cs))
	for i := range cs {
		vals[i] = 123.45
	}
	m.SetOverlay("ema9", vals, lipgloss.NewStyle())
	m.Draw()
	var row strings.Builder
	for x := 0; x < m.Width(); x++ {
		row.WriteRune(m.Cell(canvas.Point{X: x, Y: 0}).Rune)
	}
	if !strings.Contains(row.String(), "ema9 123.5") {
		t.Fatalf("readout %q", row.String())
	}
}

func TestFormingCandleKeepsPaneValues(t *testing.T) {
	m := New(80, 20, WithVolume(0))
	cs := series(20)
	m.SetCandles(cs)
	vals := make([]float64, len(cs))
	for i := range vals {
		vals[i] = 50
	}
	m.SetPane(Pane{Name: "RSI", Rows: 3, Min: 0, Max: 100, Series: []PaneSeries{{Name: "rsi", Values: vals, Style: lipgloss.NewStyle()}}})
	m.SetOverlay("ema", vals, lipgloss.NewStyle())
	m.Push(mk(20, 1, 2, 0.5, 1.5, 1))
	m.Draw()
	var row, top strings.Builder
	for x := 0; x < m.Width(); x++ {
		row.WriteRune(m.Cell(canvas.Point{X: x, Y: m.paneTop(0)}).Rune)
		top.WriteRune(m.Cell(canvas.Point{X: x, Y: 0}).Rune)
	}
	if !strings.Contains(row.String(), "rsi 50") {
		t.Fatalf("pane label %q", row.String())
	}
	if !strings.Contains(top.String(), "ema 50") {
		t.Fatalf("readout %q", top.String())
	}
}

func TestMultiDayIntradayLabels(t *testing.T) {
	m := New(120, 16, WithVolume(0), WithReadout(false))
	var cs []Candle
	base := time.Date(2026, 7, 29, 9, 15, 0, 0, time.UTC)
	for d := 0; d < 3; d++ {
		for i := 0; i < 40; i++ {
			cs = append(cs, Candle{Time: base.AddDate(0, 0, d).Add(time.Duration(i*5) * time.Minute), Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 1})
		}
	}
	m.SetCandles(cs)
	m.Draw()
	var row strings.Builder
	for x := 0; x < m.Width(); x++ {
		row.WriteRune(m.Cell(canvas.Point{X: x, Y: m.Height() - 1}).Rune)
	}
	if !strings.Contains(row.String(), "Wed 09:") || !strings.Contains(row.String(), "Thu ") {
		t.Fatalf("labels %q", row.String())
	}
}

func TestEmptyDrawDoesNotPanic(t *testing.T) {
	m := New(20, 6)
	_ = m.View()
	m.Resize(3, 2)
	_ = m.View()
}
