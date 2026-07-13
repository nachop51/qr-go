# v0.2.0

v0.2 is a breaking correctness and security release. It fixes 18-bit version
metadata placement, validates every public option, prevents SVG paint/XML
injection, makes file output atomic, bounds image allocations, and aligns the
CLI, library, and browser behavior.

## Migration

| v0.1 | v0.2 |
| --- | --- |
| `code.Version` | `code.Version()` |
| `code.Mask` | `code.Mask()` |
| `code.ErrorCorrectionLevel` | `code.CorrectionLevel()` |
| `code.IsECI` | `code.UsesECI()` |
| `code.Segments` | `code.Segments()` (copy-returning) |
| mutable `CorrectionLevel` structs | `uint8`-backed named constants |
| global `render.Warnf` | `png.New().WarningHandler(fn)` / `svg.New().WarningHandler(fn)` |

`Segment.Bytes()` returns a copy for binary inspection; `Segment.Data()` is
retained for text-oriented inspection. `--info` now always writes to stderr.
PNG and SVG share the strict color grammar documented in the README. Logo
sizing is a conservative recommendation, not a guarantee. Zone-less CLI event
date-times use the machine local timezone, while explicit RFC3339 offsets win.
