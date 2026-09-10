# rekha

Terminal charts for [Bubbletea v2](https://github.com/charmbracelet/bubbletea). Built for [tej](https://github.com/tejhq/tej-cli), usable anywhere.

Forked from [ntcharts](https://github.com/NimbleMarkets/ntcharts) by NimbleMarkets. Trimmed to chart primitives, no image or Kitty graphics.

## Packages

| Package | What |
|---|---|
| `canvas` | Cell grid, cursor, viewport. Base for everything. |
| `canvas/graph` | Line, braille, candlestick drawing on a canvas. |
| `canvas/runes` | Box drawing, braille, arc rune sets. |
| `linechart` | XY line chart with axes, labels, zoom, pan. |
| `linechart/timeserieslinechart` | Line chart with time X axis. |
| `linechart/streamlinechart` | Rolling line for live streams. |
| `barchart` | Vertical and horizontal bars. |
| `sparkline` | Inline mini chart. |

Planned: `candlestick`, OHLC model with overlays, volume pane, crosshair.

## Install

```
go get github.com/tejhq/rekha
```

## License

MIT. See [LICENSE](./LICENSE). Original ntcharts code copyright Neomantra Corp.
