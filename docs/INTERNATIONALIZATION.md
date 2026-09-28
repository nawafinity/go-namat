# Internationalization

Namat treats template expressions and inserted text as UTF-8. It does not
assume an application language, locale, calendar, number format, or writing
direction.

## Text and writing direction

- Unicode values and Unicode identifiers are supported by the expression
  engine.
- Arabic, Latin, CJK, and emoji values are covered by automated tests.
- Namat preserves Word paragraph and run properties, including `w:bidi` and
  `w:rtl`, when replacing text.
- Writing direction, fonts, shaping, and language metadata should be authored
  in the Word template. Namat does not guess direction from inserted text.

## Locale-sensitive formatting

Dates, currencies, percentages, digit shaping, and localized labels are
application concerns. Register explicit Go functions for them:

```go
options := namat.Options{
    Functions: map[string]namat.Function{
        "formatCurrency": formatCurrency,
        "formatDate":     formatDate,
    },
}
```

Keeping locale policy in host functions makes it testable and prevents the
template engine from silently applying the machine's current locale.

## Project language

The public API, source comments, diagnostics, documentation, and committed test
source are English so contributors can work with the project internationally.
Multilingual test values are represented with Unicode escapes where needed.
