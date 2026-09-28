# Tiraz architecture

Tiraz is a pure Go library. It does not start a helper process, bundle Node.js,
or embed a JavaScript engine.

## Rendering pipeline

1. **Package reader** validates the DOCX ZIP and retains every package part,
   header, compression method, and binary payload.
2. **OOXML parser** builds a prefix-preserving tree for templated `word/*.xml`
   parts.
3. **Word normalizer** rejoins command delimiters split across runs or even
   adjacent paragraphs by Word editing.
4. **Command compiler** validates expressions and balanced structural blocks
   before report data is processed.
5. **Native expression evaluator** resolves Go maps, structs, slices,
   registered host functions, local variables, and bounded operators.
6. **Structural renderer** clones paragraph or table-row ranges for conditions
   and loops, then renders leaf commands.
7. **Package writer** serializes changed parts and copies untouched media and
   relationships into a valid DOCX result.

## Trust boundaries

Template expressions are data, not Go or JavaScript source. They cannot access
the filesystem, network, processes, environment variables, or reflection APIs.
Only functions explicitly registered by the host application are callable.

Rendering is bounded by a context timeout and a maximum aggregate loop count.
The compiled template is immutable; each render receives private package,
variable, and XML copies.

## Compatibility approach

Tiraz aims for behavioral compatibility with established DOCX template
workflows, not an embedded JavaScript compatibility layer. Familiar expression
syntax such as property access, optional chaining, null coalescing, comparisons,
template strings, and `$` loop variables is parsed directly by Go code.

Application-specific JavaScript helpers are migrated to typed Go host
functions. This makes the executable smaller and gives helper logic normal Go
tests, static analysis, and profiling.
