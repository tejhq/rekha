package candlestick

import (
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/tejhq/rekha/canvas"
	"github.com/tejhq/rekha/canvas/runes"
)

type Candle struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

func (c Candle) Bull() bool { return c.Close >= c.Open }

type Overlay struct {
	Name   string
	Values []float64
	Style  lipgloss.Style
}

type TimeFormatter func(time.Time) string
type PriceFormatter func(float64) string

var (
	defaultStyle     = lipgloss.NewStyle()
	defaultBull      = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	defaultBear      = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	defaultAxis      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	defaultLabel     = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	defaultCrosshair = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	defaultLast      = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	defaultLastLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("3"))
	defaultReadout   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
)

type Model struct {
	Canvas         canvas.Model
	KeyMap         KeyMap
	Style          lipgloss.Style
	AxisStyle      lipgloss.Style
	LabelStyle     lipgloss.Style
	BullStyle      lipgloss.Style
	BearStyle      lipgloss.Style
	CrosshairStyle lipgloss.Style
	LastStyle      lipgloss.Style
	LastLabelStyle lipgloss.Style
	ReadoutStyle   lipgloss.Style
	TimeFormatter  TimeFormatter
	PriceFormatter PriceFormatter

	candles      []Candle
	overlays     []*Overlay
	levels       []Level
	markers      map[string][]Marker
	maxCandles   int
	candleWidth  int
	gap          int
	volumeHeight int
	yStep        int
	yPad         float64
	showLast     bool
	showReadout  bool
	showVolume   bool

	offset int
	cursor int
	focus  bool
	dirty  bool

	graphW, graphH int
	axisX, axisY   int
	viewMin        float64
	viewMax        float64
	visStart       int
	visEnd         int

	zoneManager *zone.Manager
	zoneID      string
}

func New(w, h int, opts ...Option) Model {
	m := Model{
		Canvas:         canvas.New(w, h),
		KeyMap:         DefaultKeyMap(),
		Style:          defaultStyle,
		AxisStyle:      defaultAxis,
		LabelStyle:     defaultLabel,
		BullStyle:      defaultBull,
		BearStyle:      defaultBear,
		CrosshairStyle: defaultCrosshair,
		LastStyle:      defaultLast,
		LastLabelStyle: defaultLastLabel,
		ReadoutStyle:   defaultReadout,
		PriceFormatter: DefaultPriceFormatter,
		candleWidth:    1,
		yStep:          3,
		yPad:           0.05,
		showLast:       true,
		showReadout:    true,
		showVolume:     true,
		volumeHeight:   3,
		cursor:         -1,
		dirty:          true,
	}
	for _, o := range opts {
		o(&m)
	}
	return m
}

func DefaultPriceFormatter(v float64) string {
	switch {
	case math.Abs(v) >= 10000:
		return fmt.Sprintf("%.0f", v)
	case math.Abs(v) >= 100:
		return fmt.Sprintf("%.1f", v)
	default:
		return fmt.Sprintf("%.2f", v)
	}
}

func (m *Model) Width() int  { return m.Canvas.Width() }
func (m *Model) Height() int { return m.Canvas.Height() }

func (m *Model) Resize(w, h int) {
	m.Canvas.Resize(w, h)
	m.Canvas.ViewWidth = w
	m.Canvas.ViewHeight = h
	m.dirty = true
}

func (m *Model) Candles() []Candle { return m.candles }
func (m *Model) Len() int          { return len(m.candles) }

func (m *Model) Last() (Candle, bool) {
	if len(m.candles) == 0 {
		return Candle{}, false
	}
	return m.candles[len(m.candles)-1], true
}

func (m *Model) SetCandles(cs []Candle) {
	m.candles = append(m.candles[:0], cs...)
	m.trim()
	for _, o := range m.overlays {
		if len(o.Values) > len(m.candles) {
			o.Values = o.Values[len(o.Values)-len(m.candles):]
		}
	}
	m.offset = 0
	m.cursor = -1
	m.dirty = true
}

func (m *Model) Push(c Candle) {
	n := len(m.candles)
	if n > 0 && m.candles[n-1].Time.Equal(c.Time) {
		m.candles[n-1] = c
		m.dirty = true
		return
	}
	m.candles = append(m.candles, c)
	m.trim()
	if m.offset > 0 {
		m.offset++
	}
	m.dirty = true
}

func (m *Model) trim() {
	if m.maxCandles <= 0 || len(m.candles) <= m.maxCandles {
		return
	}
	drop := len(m.candles) - m.maxCandles
	m.candles = m.candles[drop:]
	for _, o := range m.overlays {
		if len(o.Values) > drop {
			o.Values = o.Values[drop:]
		} else {
			o.Values = o.Values[:0]
		}
	}
	if m.cursor >= 0 {
		m.cursor -= drop
		if m.cursor < 0 {
			m.cursor = -1
		}
	}
}

func (m *Model) Clear() {
	m.candles = m.candles[:0]
	for _, o := range m.overlays {
		o.Values = o.Values[:0]
	}
	m.levels = m.levels[:0]
	m.markers = nil
	m.offset = 0
	m.cursor = -1
	m.dirty = true
}

func (m *Model) overlay(name string) *Overlay {
	for _, o := range m.overlays {
		if o.Name == name {
			return o
		}
	}
	return nil
}

func (m *Model) SetOverlay(name string, values []float64, s lipgloss.Style) {
	o := m.overlay(name)
	if o == nil {
		o = &Overlay{Name: name}
		m.overlays = append(m.overlays, o)
	}
	o.Values = append(o.Values[:0], values...)
	o.Style = s
	m.dirty = true
}

func (m *Model) PushOverlay(name string, v float64) {
	o := m.overlay(name)
	if o == nil {
		return
	}
	if len(o.Values) >= len(m.candles) && len(o.Values) > 0 {
		o.Values[len(o.Values)-1] = v
	} else {
		o.Values = append(o.Values, v)
	}
	m.dirty = true
}

func (m *Model) RemoveOverlay(name string) {
	for i, o := range m.overlays {
		if o.Name == name {
			m.overlays = append(m.overlays[:i], m.overlays[i+1:]...)
			m.dirty = true
			return
		}
	}
}

func (m *Model) stride() int { return m.candleWidth + m.gap }

func (m *Model) capacity() int {
	if m.stride() <= 0 || m.graphW <= 0 {
		return 0
	}
	return m.graphW / m.stride()
}

func (m *Model) Offset() int { return m.offset }

func (m *Model) Scroll(n int) {
	m.layout()
	maxOff := len(m.candles) - m.capacity()
	if maxOff < 0 {
		maxOff = 0
	}
	m.offset += n
	if m.offset > maxOff {
		m.offset = maxOff
	}
	if m.offset < 0 {
		m.offset = 0
	}
	m.dirty = true
}

func (m *Model) ScrollToLatest() {
	m.offset = 0
	m.dirty = true
}

func (m *Model) SetCandleWidth(w int) {
	if w < 1 {
		w = 1
	}
	m.candleWidth = w
	m.dirty = true
}

func (m *Model) CandleWidth() int { return m.candleWidth }

func (m *Model) Cursor() (Candle, int, bool) {
	if m.cursor < 0 || m.cursor >= len(m.candles) {
		return Candle{}, -1, false
	}
	return m.candles[m.cursor], m.cursor, true
}

func (m *Model) SetCursor(i int) {
	if i < 0 || i >= len(m.candles) {
		m.cursor = -1
		m.dirty = true
		return
	}
	m.cursor = i
	m.layout()
	if i < m.visStart {
		m.offset += m.visStart - i
	} else if i >= m.visEnd {
		m.offset -= i - m.visEnd + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
	m.dirty = true
}

func (m *Model) ClearCursor() {
	m.cursor = -1
	m.dirty = true
}

func (m *Model) MoveCursor(n int) {
	if len(m.candles) == 0 {
		return
	}
	if m.cursor < 0 {
		m.layout()
		m.SetCursor(m.visEnd - 1)
		if n < 0 {
			return
		}
	}
	i := m.cursor + n
	if i < 0 {
		i = 0
	}
	if i >= len(m.candles) {
		i = len(m.candles) - 1
	}
	m.SetCursor(i)
}

func (m *Model) Visible() (start, end int) {
	m.layout()
	return m.visStart, m.visEnd
}

func (m *Model) ViewRange() (min, max float64) { return m.viewMin, m.viewMax }

func (m *Model) Focused() bool { return m.focus }
func (m *Model) Focus()        { m.focus = true; m.Canvas.Focus() }
func (m *Model) Blur()         { m.focus = false; m.Canvas.Blur() }

func (m *Model) SetZoneManager(zm *zone.Manager) {
	m.zoneManager = zm
	m.zoneID = zm.NewPrefix()
}

func (m *Model) ZoneManager() *zone.Manager { return m.zoneManager }
func (m *Model) ZoneID() string             { return m.zoneID }

func (m *Model) computeVisible() {
	capa := m.capacity()
	n := len(m.candles)
	if n <= capa {
		m.visStart, m.visEnd = 0, n
		return
	}
	end := n - m.offset
	if end > n {
		end = n
	}
	start := end - capa
	if start < 0 {
		start = 0
		end = capa
	}
	m.visStart, m.visEnd = start, end
}

func (m *Model) priceLabelWidth() int {
	w := 0
	for _, v := range []float64{m.viewMin, m.viewMax, (m.viewMin + m.viewMax) / 2} {
		if l := len(m.PriceFormatter(v)); l > w {
			w = l
		}
	}
	for _, lv := range m.levels {
		if l := len([]rune(lv.Label)); l > w {
			w = l
		}
	}
	return w
}

func (m *Model) computeRange(start, end int) {
	if end <= start {
		m.viewMin, m.viewMax = 0, 1
		return
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, c := range m.candles[start:end] {
		lo = math.Min(lo, c.Low)
		hi = math.Max(hi, c.High)
	}
	for _, o := range m.overlays {
		for i := start; i < end && i < len(o.Values); i++ {
			v := o.Values[i]
			if math.IsNaN(v) || v == 0 {
				continue
			}
			lo = math.Min(lo, v)
			hi = math.Max(hi, v)
		}
	}
	for _, lv := range m.levels {
		if lv.Price > 0 {
			lo = math.Min(lo, lv.Price)
			hi = math.Max(hi, lv.Price)
		}
	}
	pad := (hi - lo) * m.yPad
	if pad == 0 {
		pad = math.Max(math.Abs(hi)*0.001, 0.01)
	}
	m.viewMin, m.viewMax = lo-pad, hi+pad
}

func (m *Model) layout() {
	w, h := m.Canvas.Width(), m.Canvas.Height()
	top := 0
	if m.showReadout {
		top = 1
	}
	m.axisY = h - 2
	m.graphH = max(m.axisY-m.volumeRows()-top, 1)
	m.graphW = max(w-8, 1)
	m.computeVisible()
	m.computeRange(m.visStart, m.visEnd)
	m.axisX = max(w-m.priceLabelWidth()-1, 1)
	m.graphW = m.axisX
	m.computeVisible()
	m.computeRange(m.visStart, m.visEnd)
}

func (m *Model) graphTop() int { return m.axisY - m.volumeRows() - m.graphH }

func (m *Model) volumeRows() int {
	if !m.showVolume {
		return 0
	}
	return m.volumeHeight
}

func (m *Model) priceRow(p float64) int {
	span := m.viewMax - m.viewMin
	if span <= 0 {
		return m.graphTop() + m.graphH - 1
	}
	f := (p - m.viewMin) / span
	row := m.graphTop() + m.graphH - 1 - int(math.Round(f*float64(m.graphH-1)))
	return row
}

func (m *Model) rowPrice(row int) float64 {
	if m.graphH <= 1 {
		return m.viewMin
	}
	f := float64(m.graphTop()+m.graphH-1-row) / float64(m.graphH-1)
	return m.viewMin + f*(m.viewMax-m.viewMin)
}

func (m *Model) colOf(i int) int {
	return (i - m.visStart) * m.stride()
}

func (m *Model) Draw() {
	m.dirty = false
	m.Canvas.Clear()
	m.Canvas.SetStyle(m.Style)
	m.layout()
	m.drawAxes()
	m.drawLastPrice()
	m.drawLevels()
	m.drawOverlays()
	m.drawCandles()
	m.drawVolume()
	m.drawMarkers()
	m.drawCrosshair()
	m.drawReadout()
}

func (m *Model) drawAxes() {
	w := m.Canvas.Width()
	top := m.graphTop()
	_ = w
	for y := top; y <= m.axisY; y++ {
		m.Canvas.SetCell(canvas.Point{X: m.axisX, Y: y}, canvas.NewCellWithStyle(runes.LineVertical, m.AxisStyle))
	}
	for x := 0; x < m.axisX; x++ {
		m.Canvas.SetCell(canvas.Point{X: x, Y: m.axisY}, canvas.NewCellWithStyle(runes.LineHorizontal, m.AxisStyle))
	}
	m.Canvas.SetCell(canvas.Point{X: m.axisX, Y: m.axisY}, canvas.NewCellWithStyle(runes.LineUpLeft, m.AxisStyle))

	bottom := top + m.graphH - 1
	for row := bottom; row >= top; row -= m.yStep {
		s := m.PriceFormatter(m.rowPrice(row))
		m.Canvas.SetStringWithStyle(canvas.Point{X: m.axisX + 1, Y: row}, s, m.LabelStyle)
	}

	if m.TimeFormatter == nil {
		m.TimeFormatter = m.autoTimeFormatter()
	}
	lastEnd := -2
	for i := m.visStart; i < m.visEnd; i++ {
		x := m.colOf(i)
		s := m.TimeFormatter(m.candles[i].Time)
		if x <= lastEnd+1 || x+len(s) > w {
			continue
		}
		m.Canvas.SetStringWithStyle(canvas.Point{X: x, Y: m.axisY + 1}, s, m.LabelStyle)
		m.Canvas.SetCell(canvas.Point{X: x, Y: m.axisY}, canvas.NewCellWithStyle(runes.LineHorizontalUp, m.AxisStyle))
		lastEnd = x + len(s)
	}
}

func (m *Model) autoTimeFormatter() TimeFormatter {
	if m.visEnd-m.visStart < 2 {
		return func(t time.Time) string { return t.Format("15:04") }
	}
	a, b := m.candles[m.visStart].Time, m.candles[m.visEnd-1].Time
	if a.YearDay() == b.YearDay() && a.Year() == b.Year() {
		return func(t time.Time) string { return t.Format("15:04") }
	}
	if b.Sub(a) < 400*24*time.Hour {
		return func(t time.Time) string { return t.Format("02 Jan") }
	}
	return func(t time.Time) string { return t.Format("Jan 06") }
}

func (m *Model) drawLastPrice() {
	if !m.showLast || len(m.candles) == 0 {
		return
	}
	last := m.candles[len(m.candles)-1].Close
	if last < m.viewMin || last > m.viewMax {
		return
	}
	row := m.priceRow(last)
	for x := 0; x < m.axisX; x++ {
		m.Canvas.SetCell(canvas.Point{X: x, Y: row}, canvas.NewCellWithStyle('┄', m.LastStyle))
	}
	s := m.PriceFormatter(last)
	m.Canvas.SetStringWithStyle(canvas.Point{X: m.axisX + 1, Y: row}, s, m.LastLabelStyle)
}

func (m *Model) drawOverlays() {
	for _, o := range m.overlays {
		prevRow := -1
		for i := m.visStart; i < m.visEnd; i++ {
			if i >= len(o.Values) {
				break
			}
			v := o.Values[i]
			if math.IsNaN(v) || v == 0 {
				prevRow = -1
				continue
			}
			row := m.priceRow(v)
			x := m.colOf(i)
			r := runes.LineHorizontal
			if i+1 < m.visEnd && i+1 < len(o.Values) && !math.IsNaN(o.Values[i+1]) && o.Values[i+1] != 0 {
				next := m.priceRow(o.Values[i+1])
				switch {
				case next < row:
					r = '╱'
				case next > row:
					r = '╲'
				}
			}
			for k := 0; k < m.candleWidth; k++ {
				m.Canvas.SetCell(canvas.Point{X: x + k, Y: row}, canvas.NewCellWithStyle(r, o.Style))
			}
			if prevRow >= 0 {
				lo, hi := prevRow, row
				if lo > hi {
					lo, hi = hi, lo
				}
				for y := lo + 1; y < hi; y++ {
					m.Canvas.SetCell(canvas.Point{X: x, Y: y}, canvas.NewCellWithStyle(runes.LineVertical, o.Style))
				}
			}
			prevRow = row
		}
	}
}

func (m *Model) drawCandles() {
	for i := m.visStart; i < m.visEnd; i++ {
		c := m.candles[i]
		s := m.BullStyle
		if !c.Bull() {
			s = m.BearStyle
		}
		x := m.colOf(i)
		hiRow := m.priceRow(c.High)
		loRow := m.priceRow(c.Low)
		topRow := m.priceRow(math.Max(c.Open, c.Close))
		botRow := m.priceRow(math.Min(c.Open, c.Close))
		mid := x + m.candleWidth/2
		for y := hiRow; y < topRow; y++ {
			m.Canvas.SetCell(canvas.Point{X: mid, Y: y}, canvas.NewCellWithStyle(runes.LineVertical, s))
		}
		for y := topRow; y <= botRow; y++ {
			for k := 0; k < m.candleWidth; k++ {
				m.Canvas.SetCell(canvas.Point{X: x + k, Y: y}, canvas.NewCellWithStyle(runes.FullBlock, s))
			}
		}
		for y := botRow + 1; y <= loRow; y++ {
			m.Canvas.SetCell(canvas.Point{X: mid, Y: y}, canvas.NewCellWithStyle(runes.LineVertical, s))
		}
	}
}

func (m *Model) drawVolume() {
	rows := m.volumeRows()
	if rows <= 0 || m.visEnd <= m.visStart {
		return
	}
	maxV := 0.0
	for _, c := range m.candles[m.visStart:m.visEnd] {
		maxV = math.Max(maxV, c.Volume)
	}
	if maxV <= 0 {
		return
	}
	base := m.axisY - 1
	for i := m.visStart; i < m.visEnd; i++ {
		c := m.candles[i]
		s := m.BullStyle
		if !c.Bull() {
			s = m.BearStyle
		}
		h := c.Volume / maxV * float64(rows)
		full := int(math.Floor(h))
		frac := h - float64(full)
		x := m.colOf(i)
		for k := 0; k < m.candleWidth; k++ {
			for y := 0; y < full; y++ {
				m.Canvas.SetCell(canvas.Point{X: x + k, Y: base - y}, canvas.NewCellWithStyle(runes.FullBlock, s))
			}
			if r := runes.LowerBlockElementFromFloat64(frac); r != runes.Null && full < rows {
				m.Canvas.SetCell(canvas.Point{X: x + k, Y: base - full}, canvas.NewCellWithStyle(r, s))
			}
		}
	}
}

func (m *Model) drawCrosshair() {
	if m.cursor < m.visStart || m.cursor >= m.visEnd {
		return
	}
	c := m.candles[m.cursor]
	x := m.colOf(m.cursor) + m.candleWidth/2
	row := m.priceRow(c.Close)
	top := m.graphTop()
	for y := top; y < m.axisY; y++ {
		if m.Canvas.Cell(canvas.Point{X: x, Y: y}).Rune == runes.Null {
			m.Canvas.SetCell(canvas.Point{X: x, Y: y}, canvas.NewCellWithStyle('┊', m.CrosshairStyle))
		}
	}
	for xx := 0; xx < m.axisX; xx++ {
		if m.Canvas.Cell(canvas.Point{X: xx, Y: row}).Rune == runes.Null {
			m.Canvas.SetCell(canvas.Point{X: xx, Y: row}, canvas.NewCellWithStyle('┈', m.CrosshairStyle))
		}
	}
	m.Canvas.SetCell(canvas.Point{X: x, Y: m.axisY}, canvas.NewCellWithStyle(runes.LineHorizontalUp, m.CrosshairStyle))
	s := m.PriceFormatter(c.Close)
	m.Canvas.SetStringWithStyle(canvas.Point{X: m.axisX + 1, Y: row}, s, m.LastLabelStyle)
}

func (m *Model) drawReadout() {
	if !m.showReadout {
		return
	}
	var c Candle
	var ok bool
	if m.cursor >= 0 && m.cursor < len(m.candles) {
		c, ok = m.candles[m.cursor], true
	} else {
		c, ok = m.Last()
	}
	if !ok {
		return
	}
	pf := m.PriceFormatter
	chg := 0.0
	if c.Open != 0 {
		chg = (c.Close - c.Open) / c.Open * 100
	}
	when := c.Time.Format("02 Jan 15:04")
	if c.Time.Hour() == 0 && c.Time.Minute() == 0 {
		when = c.Time.Format("02 Jan 2006")
	}
	s := fmt.Sprintf("%s  O %s  H %s  L %s  C %s  %+.2f%%  V %s",
		when, pf(c.Open), pf(c.High), pf(c.Low), pf(c.Close), chg, formatVolume(c.Volume))
	if len(s) > m.Canvas.Width() {
		s = s[:m.Canvas.Width()]
	}
	m.Canvas.SetStringWithStyle(canvas.Point{X: 0, Y: 0}, s, m.ReadoutStyle)
}

func formatVolume(v float64) string {
	switch {
	case v >= 1e7:
		return fmt.Sprintf("%.2fCr", v/1e7)
	case v >= 1e5:
		return fmt.Sprintf("%.2fL", v/1e5)
	case v >= 1e3:
		return fmt.Sprintf("%.1fK", v/1e3)
	default:
		return strings.TrimSuffix(fmt.Sprintf("%.0f", v), ".")
	}
}
