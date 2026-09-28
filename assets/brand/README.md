# Namat identity

Namat means “pattern” or “template” in Arabic. Its symbol is an abstract **ن**
(noon), built from two mirrored corner modules and a diamond-shaped dot.
The repeated modules express structure and assembly; the custom lowercase
wordmark gives the identity a clear, approachable international voice.

## Assets

| File | Use |
| --- | --- |
| `namat-logo-light.svg` | Full lockup on light backgrounds |
| `namat-logo-dark.svg` | Full lockup on dark backgrounds |
| `namat-mark-light.svg` | Standalone symbol on light backgrounds |
| `namat-mark-dark.svg` | Standalone symbol on dark backgrounds |
| `og-image.svg` | Editable, self-contained social-preview source |
| `og-image.png` | 1280 × 640 GitHub Social Preview upload |

## Color

| Color | Value | Role |
| --- | --- | --- |
| Deep teal | `#143C3C` | Light-theme symbol and wordmark |
| Porcelain | `#EEF8F5` | Dark-theme symbol and wordmark |
| Jade | `#18B99A` | Diamond accent in both themes |
| Night teal | `#102F30` | Social-preview background |

Light and dark assets share identical geometry. Only the primary ink changes;
the jade accent stays the same. Choose the asset for its background rather
than applying an inversion filter. Keep the original proportions and the
clear space included in each SVG's viewBox.

## Construction and use

- Every visible element is vector geometry, including the custom wordmark
  and the outlined social-preview lettering. No fonts or images are loaded.
- The standalone symbol is designed to remain recognizable at 16 px;
  24 px or larger is preferred when space allows. The full lockup is best
  used at 180 px wide or larger.
- The symbol's two modules are reflections across its vertical axis. Keep
  the central gap open and retain the diamond's position and proportions.
- SVG titles and descriptions provide accessible names. When embedding an
  SVG through an HTML image element, also provide appropriate `alt` text.
- The social preview is one fixed composition. Edit its labeled vector
  groups, then export an opaque sRGB PNG at exactly 1280 × 640. Keep that
  PNG below 1 MB. It does not switch themes with a README's picture element.

All assets were constructed and rendered locally. The earlier raster logo
and mark exports are superseded by these SVGs and remain in Git history.
