# Migration to the strict v1 language

Namat is still pre-v1. The strict language intentionally replaces the earlier
prototype syntax without a legacy mode or automatic detection. Convert every
template before updating the library.

## Command mapping

| Earlier prototype | Strict v1 |
| --- | --- |
| `[[INS value]]`, `[[= value]]` | `[[value]]` |
| `[[EXEC name = value]]`, `[[SET name = value]]` | `[[#let name = value]]` |
| `[[IF condition]]` / `[[ELSE]]` / `[[END-IF]]` | `[[#if condition]]` / `[[#else]]` / `[[/if]]` |
| `[[FOR item IN items]]` / `[[END-FOR item]]` | `[[#each items as item]]` / `[[/each]]` |
| `$item`, `$idx` | `item`, `loop.index` or `loop.number` |
| `[[IMAGE value]]` | `[[@image value]]` |
| `[[LINK value]]` | `[[@link value]]` |
| `[[HTML value]]` | `[[@html value]]` |
| `[[RAW-XML value]]` | `[[@raw-xml value]]` |
| aliases | replace with `#let` or a registered function |
| `QUERY` | load data in the application before calling `Render` |

Structural directives and `#let` must occupy a complete paragraph or table
row. `@html` and `@raw-xml` must occupy a complete paragraph.

## Expression changes

- Conditions accept `bool` only; there is no implicit truthiness.
- `+` adds compatible numeric values or concatenates two strings. Mixed types
  are errors.
- Equality and ordering do not convert unrelated values to text.
- `===`, `!==`, ternary expressions, template literals, and member-method calls
  were removed.
- Use filters such as `name | trim | upper` and
  `customer?.name | default("—")`.
- Missing and null are distinct. Direct insertion of either is an error.
- Integer values retain their width and decimal literals use exact
  `namat.Decimal` arithmetic. Use `namat.DecodeJSON` for JSON input.
- Map and struct fields use exact JSON names; case-insensitive fallback was
  removed.

## Registered functions

`Options.Functions` now accepts `namat.FunctionSpec`. Declare parameter and
return types, and accept the render context:

```go
options := namat.Options{Functions: map[string]namat.FunctionSpec{
    "money": {
        Params:  []namat.ValueType{namat.TypeDecimal, namat.TypeString},
        Returns: namat.TypeString,
        Pure:    true,
        Call: func(ctx context.Context, args ...any) (any, error) {
            return formatMoney(args[0].(namat.Decimal), args[1].(string)), nil
        },
    },
}}
```

Function names and arity are checked during `Compile`; argument types are
checked when rendering. Function values stored in report data are never
callable.

## Validation checklist

1. Convert all commands using the table above.
2. Replace truthy conditions with explicit boolean expressions.
3. Replace string/number coercion with explicit formatting functions.
4. Register every callable helper with a `FunctionSpec`.
5. Decode JSON through `namat.DecodeJSON`.
6. Run `namat lint --data sample.json template.docx`.
7. Render representative large data and inspect every page in each supported
   Word-compatible client.

There is no compatibility flag. A template using prototype syntax fails during
compilation so an outdated template cannot silently produce a logically wrong
report.
