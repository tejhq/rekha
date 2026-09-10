# rekha

Terminal charts for [Bubbletea v2](https://github.com/charmbracelet/bubbletea). Built for [tej](https://github.com/tejhq/tej-cli), usable anywhere.

Forked from [ntcharts](https://github.com/NimbleMarkets/ntcharts) by NimbleMarkets. Trimmed to chart primitives, no image or Kitty graphics.

## Packages

| Package | What |
|---|---|
| `candlestick` | OHLC chart. Half-block candles, braille overlays, indicator sub-panes, volume, nice price ticks with grid, last price line, price levels, markers, crosshair, scroll, zoom, auto candle width. |
| `canvas` | Cell grid, cursor, viewport. Base for everything. |
| `canvas/graph` | Line, braille, candlestick drawing on a canvas. |
| `canvas/runes` | Box drawing, braille, arc rune sets. |
| `linechart` | XY line chart with axes, labels, zoom, pan. |
| `linechart/timeserieslinechart` | Line chart with time X axis. |
| `linechart/streamlinechart` | Rolling line for live streams. |
| `barchart` | Vertical and horizontal bars. |
| `sparkline` | Inline mini chart. |

## Install

```
go get github.com/tejhq/rekha
```

## Candlestick

```go
m := candlestick.New(80, 24, candlestick.WithVolume(3))
m.SetCandles(candles)
m.SetOverlay("ema9", ema9, lipgloss.NewStyle().Foreground(lipgloss.Color("4")))
m.Push(latest)
m.PushOverlay("ema9", ema9Now)
m.SetPane(candlestick.Pane{Name: "RSI", Rows: 3, Min: 0, Max: 100,
	Series: []candlestick.PaneSeries{{Name: "rsi", Values: rsi, Style: blueStyle}},
	Lines:  []candlestick.PaneLine{{Value: 30, Style: dim}, {Value: 70, Style: dim}}})
m.SetLevel("stop", 1200, "S 1200", redStyle)
m.AddMarker("fills", candlestick.Marker{Time: fillTime, Price: 1234.5, Rune: '▲', Style: greenStyle})
m.Focus()
```

Keys: `←/→` cursor, `PgUp/PgDn` scroll, `End` latest, `Esc` clear cursor, `+/-` candle width. Mouse click and wheel with a bubblezone manager.

```
08 Apr 11:05  O 24784  H 24796  L 24772  C 24784  -0.00%  V 3.00L
                                                                ┊         │24855
                               ████│                            ┊         │
│││││                        │██││██│                        │││││││      │
██████│┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈██┈┈┈─██┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈│█████████│┈┈┈┈│24784
 ───│███                    ██  ─╱  ██╲                   ███│  ───│███   │
╱     │██╲                 ██│ ╱     ██╲                 ██│  ─╱┊    │██╲ │
       │█│╲               ██│ ╱      │██─╲              │█│ ─╱  ┊     │█│─│24700
▂▅█    ▂▅█    ▂▅█    ▂▅█    ▂▅█    ▂▅█    ▂▅█    ▂▅█    ▂▅█    ▂▅█    ▂▅█ │
███▇██████▇██████▇██████▇██████▇██████▇██████▇██████▇██████▇██████▇██████▇│
┴──────┴──────┴──────┴──────┴──────┴──────┴──────┴──────┴──────┴┴─────┴───┘
10:01  10:08  10:15  10:22  10:29  10:36  10:43  10:50  10:57  11:04  11:11
```

## License

MIT. See [LICENSE](./LICENSE). Original ntcharts code copyright Neomantra Corp.
