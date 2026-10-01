# Public compatibility fixtures

`complex-tables.docx` is a synthetic, product-neutral DOCX fixture containing:

- horizontal and vertical merged cells (`gridSpan` and `vMerge`);
- a styled outer table and a nested table;
- nested `FOR` blocks that repeat complete outer and inner table rows;
- an unrelated custom XML part used to verify byte-for-byte preservation.

The fixture and its generator are released under CC0-1.0. They were created
for Namat and are not derived from a customer document or production data.

Regenerate it from this directory with:

```bash
go run generate.go
```

The committed DOCX is the public interoperability input. Generated reports or
client screenshots are test artifacts and should not be committed unless their
client version, platform, date, and provenance are recorded here.

## Verification record

| Date | Client | Platform | Result | Evidence |
| --- | --- | --- | --- | --- |
| 2026-09-30 | LibreOffice 7.4.7.2 | Debian Linux container | Pass | Rendered DOCX converted to a one-page PDF; manual PNG review confirmed the merged outer cells, nested table, repeated detail rows, fills, borders, and text without clipping or overlap |

Microsoft Word and Google Docs remain manual pre-v1 verification gates. No
golden PDF or screenshot is committed yet; the generated PDF used for the
review above was kept outside the repository as temporary QA output.
