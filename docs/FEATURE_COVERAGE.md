# Core feature coverage

This matrix traces the supported Namat core to automated evidence. Each verified
row names at least one automated test, and a repository policy test checks that
the referenced test names exist. Client-rendered visual compatibility and items
marked as pending in `COMPATIBILITY.md` are outside this automated claim.

## Package and OOXML

| Capability | Automated evidence |
| --- | --- |
| DOCX round trip and untouched binary parts | `TestRenderInsertAcrossWordRuns` |
| DOCM macro preservation | `TestDOCMRoundTripPreservesMacroProject` |
| Prefix, run style, and RTL property preservation | `TestInsertKeepsCommandRunAtTextBoundary`, `TestRTLFormattingPropertiesArePreserved` |
| Header/footer loops, part-local links, footnotes, and endnotes | `TestListCommandsIncludesHeaders`, `TestAllWordTextPartsAreRendered`, `TestHeaderFooterLoopsLinksAndEndnotes` |
| Commands split across runs and paragraphs | `TestRenderInsertAcrossWordRuns`, `TestRenderCommandSplitAcrossParagraphs` |
| Core and extended metadata | `TestGetMetadata` |
| Public complex-table compatibility fixture | `TestPublicComplexTableFixture` |

## Template commands

| Capability | Automated evidence |
| --- | --- |
| Expression insertion and lexical `#let` | `TestCommandSyntaxVariantsAndAssignments`, `TestLexicalLet` |
| Every v1 command type and malformed syntax | `TestParseEveryV1CommandType`, `TestParseCommandRejectsMalformedV1Syntax` |
| Nested `#if / #else / /if` | `TestRenderConditionalWithElse`, `TestNestedConditionsAndLoops` |
| Paragraph and table-row `#each` with `loop` metadata | `TestNestedConditionsAndLoops`, `TestRenderLoopRepeatsTableRows` |
| Reserved words remain available as data names | `TestReservedWordsRemainOrdinaryDataNames` |
| Structure validation | `TestCompileRejectsUnbalancedStructure`, `TestRejectsArgumentsOnEndEach` |

## Expression engine

| Capability | Automated evidence |
| --- | --- |
| Maps, structs, exact JSON names, pointers, arrays, slices, strings, and indexes | `TestV1ExpressionLanguage`, `TestExactStructAndMapNames` |
| Optional access and explicit defaults | `TestOptionalAccessPreservesMissing` |
| Strict arithmetic, comparison, equality, logical, and unary operators | `TestV1ExpressionLanguage`, `TestStrictExpressionFailures` |
| Exact integers and decimals | `TestV1ExpressionLanguage` |
| Filters and string/list helpers | `TestV1ExpressionLanguage` |
| Typed registered functions and compile-time arity checks | `TestFunctionSignatureValidation`, rich-content function tests |
| Invalid expressions, type failures, and complexity limits | `TestStrictExpressionFailures`, `TestCompileAndEvaluationLimits` |
| Multilingual values and Unicode identifiers | `TestExpressionPreservesMultilingualUnicode` |

## Rich content

| Capability | Automated evidence |
| --- | --- |
| PNG, JPEG, GIF, and SVG images | `TestRenderInlineImageAddsMediaRelationshipAndContentType`, `TestImageFormatsAndValidation`, `TestRenderSVGWithFallbackThumbnail` |
| SVG fallback, dimensions, rotation, alt text, captions, and drawing IDs | `TestRenderSVGWithFallbackThumbnail`, `TestRenderImageFromSyntheticObject`, `TestImageDrawingIdentifiersDoNotCollide` |
| Images in headers with part-local relationships | `TestImageInHeaderUsesHeaderRelationshipPart` |
| External links, tooltips, pointers, and scheme allowlists | `TestRenderInlineLinkFromObjectExpression`, `TestLinkSchemesAndPointerValue`, `TestRejectsDisallowedLinkScheme` |
| HTML altChunk and placement restriction | `TestRenderHTMLAltChunk`, `TestHTMLAndRawXMLPlacementRules` |
| Explicit RAW-XML opt-in and placement | `TestRawXMLRequiresOptIn`, `TestHTMLAndRawXMLPlacementRules` |
| Automatic and disabled line-break conversion without implicit XML | `TestLineBreaksDoNotInterpretLiteralXML`, `TestStrictObjectResultsAndLineBreakOptions` |

## Reliability, safety, and APIs

| Capability | Automated evidence |
| --- | --- |
| Typed and aggregate errors | `TestTypedNullishAndObjectErrors`, `TestCollectValidationErrors`, `TestStructuredErrorsExposeCategories` |
| Error recovery and abort callbacks | `TestErrorHandlerAndStrictNullish`, `TestErrorHandlerCanAbortRendering` |
| Strict null and object insertion failures | `TestTypedNullishAndObjectErrors`, `TestStrictObjectResultsAndLineBreakOptions` |
| Context cancellation and timeout | `TestCancelledContextStopsRender`, `TestTimeoutOption` |
| Input, ZIP, output, part-count, and loop limits | `TestReaderWriterAPIsAndLimits`, `TestPackageLimitsPartsAndUncompressedSize`, `TestAggregateIterationLimit` |
| ZIP traversal, duplicate parts, and invalid package rejection | `TestPackageRejectsTraversalAndDuplicateParts`, `TestPackageRejectsNonZIPAsInvalidTemplate` |
| Reader, writer, command listing, command copies, and API guards | `TestReaderWriterAPIsAndLimits`, `TestCreateReportReaderCommandsAndBuiltins`, `TestPublicAPIGuardsAndDefaults` |
| Concurrent compiled-template rendering | `TestCompiledTemplateRendersConcurrently`, `TestCompiledTemplateRichContentRendersConcurrently`, and race detection |
| Fuzz resistance | `FuzzParseCommand`, `FuzzReadPackage`, `FuzzRenderCommandText`, `FuzzRenderLimits`, `FuzzCompileAndEvaluate` |
| CLI inspect, metadata, render, overwrite safety, help, and version | tests in `cmd/namat/main_test.go` |
| English, product-neutral committed test sources | `TestCommittedTestsUseEnglishAndProductNeutralData` |

`TestLibreOfficeConvertsCompatibilityFixture` is a required Ubuntu CI smoke
test and an optional local test when LibreOffice is installed. It confirms that
LibreOffice can convert the generated document to a non-empty PDF; it is not a
reviewed visual golden comparison.

The matrix is the traceability index for the currently verified automated core;
it does not claim client-rendered visual coverage. CI enforces 100% statement
coverage independently for every shipped Go package. Statement coverage
complements, but does not replace, behavioral and client-compatibility evidence.
