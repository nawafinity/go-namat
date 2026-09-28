# Core feature coverage

This matrix defines the supported Namat core. Every listed capability has an
automated behavioral test. Features explicitly listed as unsupported in
`COMPATIBILITY.md` are not counted as part of the core.

## Package and OOXML

| Capability | Automated evidence |
| --- | --- |
| DOCX round trip and untouched binary parts | `TestRenderInsertAcrossWordRuns` |
| DOCM macro preservation | `TestDOCMRoundTripPreservesMacroProject` |
| Prefix, run style, and RTL property preservation | `TestInsertKeepsCommandRunAtTextBoundary`, `TestRTLFormattingPropertiesArePreserved` |
| Headers, footers, and footnotes | `TestListCommandsIncludesHeaders`, `TestAllWordTextPartsAreRendered` |
| Commands split across runs and paragraphs | `TestRenderInsertAcrossWordRuns`, `TestRenderCommandSplitAcrossParagraphs` |
| Core and extended metadata | `TestGetMetadata` |

## Template commands

| Capability | Automated evidence |
| --- | --- |
| Bare, `INS`, `=`, `EXEC`, `!`, and `SET` forms | `TestCommandSyntaxVariantsAndAssignments` |
| Every command type and malformed syntax | `TestParseEveryCommandType`, `TestParseCommandRejectsMalformedSyntax` |
| Nested `IF / ELSE / END-IF` | `TestRenderConditionalWithElse`, `TestNestedConditionsAndLoops` |
| Paragraph and table-row `FOR` with `$idx` | `TestNestedConditionsAndLoops`, `TestRenderLoopRepeatsTableRows` |
| Query resolver | `TestAliasAndQueryResolver`, `TestQueryAndAliasFailureModes` |
| Command aliases | `TestAliasAndQueryResolver`, `TestQueryAndAliasFailureModes` |
| Structure validation | `TestCompileRejectsUnbalancedStructure`, `TestRejectsMismatchedEndForName` |

## Expression engine

| Capability | Automated evidence |
| --- | --- |
| Maps, structs, JSON tags, pointers, arrays, slices, strings, and indexes | `TestExpressionLanguage`, `TestExpressionOperatorsAndBuiltInMembers` |
| Optional access and null coalescing | `TestExpressionLanguage` |
| Arithmetic, comparison, equality, logical, and unary operators | `TestExpressionOperatorsAndBuiltInMembers` |
| Ternary, arrays, objects, and template strings | `TestExpressionLanguage`, `TestExpressionOperatorsAndBuiltInMembers` |
| String and collection helpers | `TestExpressionOperatorsAndBuiltInMembers` |
| Registered and reflected Go functions | `TestCallsVariadicGoFunctionFromData`, rich-content function tests |
| Invalid expressions and evaluation failures | `TestInvalidExpression`, `TestExpressionFailureModes` |
| Multilingual values and Unicode identifiers | `TestExpressionPreservesMultilingualUnicode` |

## Rich content

| Capability | Automated evidence |
| --- | --- |
| PNG, JPEG, GIF, and SVG images | `TestRenderInlineImageAddsMediaRelationshipAndContentType`, `TestImageFormatsAndValidation`, `TestRenderSVGWithFallbackThumbnail` |
| SVG fallback, dimensions, rotation, alt text, captions, and drawing IDs | `TestRenderSVGWithFallbackThumbnail`, `TestRenderImageFromSyntheticObject`, `TestImageDrawingIdentifiersDoNotCollide` |
| Images in headers with part-local relationships | `TestImageInHeaderUsesHeaderRelationshipPart` |
| External links, tooltips, pointers, and scheme allowlists | `TestRenderInlineLinkFromObjectExpression`, `TestLinkSchemesAndPointerValue`, `TestRejectsDisallowedLinkScheme` |
| HTML altChunk and placement restriction | `TestRenderHTMLAltChunk`, `TestHTMLAndRawXMLPlacementRules` |
| Literal OOXML, RAW-XML opt-in, and placement | `TestLiteralXMLAndLineBreaks`, `TestRawXMLRequiresOptIn`, `TestHTMLAndRawXMLPlacementRules` |
| Automatic and disabled line-break conversion | `TestLiteralXMLAndLineBreaks`, `TestObjectResultsAndLineBreakOptions` |

## Reliability, safety, and APIs

| Capability | Automated evidence |
| --- | --- |
| Typed and aggregate errors | `TestTypedNullishAndObjectErrors`, `TestCollectValidationErrors`, `TestStructuredErrorsExposeCategories` |
| Error recovery and abort callbacks | `TestErrorHandlerAndRejectNullish`, `TestErrorHandlerCanAbortRendering` |
| Null and object result policies | `TestTypedNullishAndObjectErrors`, `TestObjectResultsAndLineBreakOptions` |
| Context cancellation and timeout | `TestCancelledContextStopsRender`, `TestTimeoutOption` |
| Input, ZIP, output, part-count, and loop limits | `TestReaderWriterAPIsAndLimits`, `TestPackageLimitsPartsAndUncompressedSize`, `TestAggregateIterationLimit` |
| ZIP traversal, duplicate parts, and invalid package rejection | `TestPackageRejectsTraversalAndDuplicateParts`, `TestPackageRejectsNonZIPAsInvalidTemplate` |
| Reader, writer, command listing, command copies, and API guards | `TestReaderWriterAPIsAndLimits`, `TestCreateReportReaderCommandsAndBuiltins`, `TestPublicAPIGuardsAndDefaults` |
| Concurrent compiled-template rendering | `TestCompiledTemplateRendersConcurrently` and race detection |
| Fuzz resistance | `FuzzParseCommand`, `FuzzReadPackage`, `FuzzRenderCommandText`, `FuzzCompileAndEvaluate` |
| CLI inspect, metadata, render, overwrite safety, help, and version | tests in `cmd/namat/main_test.go` |
| English, product-neutral committed test sources | `TestCommittedTestsUseEnglishAndProductNeutralData` |

The matrix represents 100% behavioral coverage of the currently documented
core feature set. The root library, native expression engine, CLI, and complete
example also maintain 100% Go statement coverage. CI enforces both guarantees;
statement coverage complements, but does not replace, behavioral assertions.
