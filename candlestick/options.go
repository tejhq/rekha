package candlestick

import (
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

type Option func(*Model)

func WithStyle(s lipgloss.Style) Option      { return func(m *Model) { m.Style = s } }
func WithAxisStyle(s lipgloss.Style) Option  { return func(m *Model) { m.AxisStyle = s } }
func WithLabelStyle(s lipgloss.Style) Option { return func(m *Model) { m.LabelStyle = s } }
func WithBullStyle(s lipgloss.Style) Option  { return func(m *Model) { m.BullStyle = s } }
func WithBearStyle(s lipgloss.Style) Option  { return func(m *Model) { m.BearStyle = s } }
func WithLastStyle(line, label lipgloss.Style) Option {
	return func(m *Model) { m.LastStyle = line; m.LastLabelStyle = label }
}
func WithCrosshairStyle(s lipgloss.Style) Option { return func(m *Model) { m.CrosshairStyle = s } }
func WithReadoutStyle(s lipgloss.Style) Option   { return func(m *Model) { m.ReadoutStyle = s } }

func WithTimeFormatter(f TimeFormatter) Option   { return func(m *Model) { m.TimeFormatter = f } }
func WithPriceFormatter(f PriceFormatter) Option { return func(m *Model) { m.PriceFormatter = f } }

func WithCandleWidth(w int) Option {
	return func(m *Model) {
		if w < 1 {
			w = 1
		}
		m.candleWidth = w
	}
}

func WithGap(g int) Option {
	return func(m *Model) {
		if g < 0 {
			g = 0
		}
		m.gap = g
	}
}

func WithMaxCandles(n int) Option { return func(m *Model) { m.maxCandles = n } }

func WithVolume(rows int) Option {
	return func(m *Model) {
		m.showVolume = rows > 0
		m.volumeHeight = rows
	}
}

func WithYStep(n int) Option {
	return func(m *Model) {
		if n < 1 {
			n = 1
		}
		m.yStep = n
	}
}

func WithYPad(f float64) Option    { return func(m *Model) { m.yPad = f } }
func WithLastPrice(on bool) Option { return func(m *Model) { m.showLast = on } }
func WithReadout(on bool) Option   { return func(m *Model) { m.showReadout = on } }
func WithKeyMap(k KeyMap) Option   { return func(m *Model) { m.KeyMap = k } }
func WithZoneManager(zm *zone.Manager) Option {
	return func(m *Model) { m.SetZoneManager(zm) }
}
