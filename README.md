<div align="center">

# نَمَط · Namat

**محرك قوالب Microsoft Word أصيل بلغة Go**<br>
**A native Go template engine for Microsoft Word**

حوّل ملفات DOCX وDOCM المصممة في Word إلى تقارير غنية بالبيانات، دون JavaScript أو Node.js أو برامج مساعدة.<br>
Turn Word-authored DOCX and DOCM files into data-driven reports—without JavaScript, Node.js, or helper processes.

[![Go](https://img.shields.io/badge/Go-1.23%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/nawafinity/go-namat.svg)](https://pkg.go.dev/github.com/nawafinity/go-namat)
[![CI](https://github.com/nawafinity/go-namat/actions/workflows/ci.yml/badge.svg)](https://github.com/nawafinity/go-namat/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/statement%20coverage-100%25-brightgreen)](docs/TESTING.md)
[![License](https://img.shields.io/badge/license-MIT-2ea44f)](LICENSE)

[لماذا نَمَط؟](#why-namat) · [البداية السريعة](#quick-start) · [لغة القالب](#template-language) · [التوثيق](#documentation) · [English](#english-overview)

</div>

> [!IMPORTANT]
> نَمَط حاليًا في مرحلة **ما قبل الإصدار 1.0**. الواجهة الأساسية ولغة القوالب قابلة للاستخدام ومغطاة بالاختبارات، لكن ضمانات التوافق الدلالي تبدأ مع الإصدار `v1.0`.<br>
> Namat is currently **pre-v1**. Its core API and template language are usable and tested, while semantic-version compatibility guarantees begin with `v1.0`.

<a id="why-namat"></a>

## لماذا نَمَط؟ · Why Namat?

نَمَط مكتبة مستقلة لإنشاء تقارير Word من قوالب يصممها المستخدم داخل Word نفسه. تحافظ المكتبة على الأنماط والصور والرؤوس والتذييلات والماكرو وأجزاء OOXML التي لا تحتاج إلى تعديل، وتغيّر فقط المواضع التي تحتوي على أوامر القالب.

Namat is a standalone library for generating Word reports from templates authored directly in Word. It preserves styles, media, headers, footers, macros, and untouched OOXML parts while changing only the areas that contain template commands.

| | العربية | English |
| --- | --- | --- |
| **Go أصيل · Pure Go** | مكتبة واحدة داخل تطبيقك، دون runtime جانبي أو ملف تنفيذي إضافي. | One embeddable library with no sidecar runtime or extra executable. |
| **Word أولًا · Word-first** | أنشئ التخطيط والتنسيق في Word بدل إعادة بنائهما برمجيًا. | Author layout and styling in Word instead of rebuilding them in code. |
| **تعبيرات آمنة · Safe expressions** | محرك محدود لا يتيح الملفات أو العمليات أو الشبكة تلقائيًا. | A bounded engine with no implicit filesystem, process, or network access. |
| **محتوى غني · Rich content** | نصوص، شروط، حلقات، جداول، صور، روابط، HTML وOOXML اختياري. | Text, conditions, loops, tables, images, links, HTML, and opt-in OOXML. |
| **جاهز للإنتاج · Production controls** | مهلات وحدود موارد وأخطاء مصنفة وإلغاء عبر `context`. | Timeouts, resource limits, typed errors, and `context` cancellation. |
| **إعادة استخدام متزامنة · Concurrent reuse** | اترجم القالب مرة واحدة واستخدمه بأمان من عدة goroutines. | Compile once and render safely from multiple goroutines. |

<a id="english-overview"></a>

### English overview

Namat brings the familiar Word-template workflow to Go without embedding a JavaScript engine. It provides a native expression language, structural document commands, rich content, package-preserving OOXML updates, and explicit safety limits. Applications remain responsible for their data model and may expose narrowly scoped Go functions when templates need domain-specific formatting or calculations.

### نبذة عربية

يوفّر نَمَط أسلوب قوالب Word المألوف داخل Go دون تضمين محرك JavaScript. ويجمع بين لغة تعبيرات أصلية، وأوامر بنيوية للمستند، ومحتوى غني، وتعديل محافظ لحزمة OOXML، وحدود أمان صريحة. يبقى نموذج البيانات داخل التطبيق، ويمكنه تسجيل دوال Go محددة للتنسيق أو الحسابات الخاصة بمجاله.

<a id="quick-start"></a>

## البداية السريعة · Quick start

### 1. التثبيت · Install

```bash
go get github.com/nawafinity/go-namat
```

لا تعتمد المكتبة على أي حزمة Go خارجية.<br>
The library has no third-party Go dependencies.

### 2. صمّم القالب في Word · Author the template in Word

اكتب الأوامر مباشرة داخل ملف DOCX أو DOCM:<br>
Type commands directly into a DOCX or DOCM file:

```text
Report for [[customer.name]]

[[IF invoice.total > 0]]
Total: [[money(invoice.total)]]
[[ELSE]]
No balance is due.
[[END-IF]]

[[FOR item IN invoice.items]]
[[$idx + 1]]. [[$item.name]]
[[END-FOR item]]
```

### 3. ترجم مرة وأنشئ تقارير متعددة · Compile once, render many

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nawafinity/go-namat"
)

func main() {
	source, err := os.ReadFile("invoice-template.docx")
	if err != nil {
		panic(err)
	}

	tmpl, err := namat.Compile(source, namat.Options{
		Functions: map[string]namat.Function{
			"money": func(args ...any) (any, error) {
				return "$" + fmt.Sprint(args[0]), nil
			},
		},
	})
	if err != nil {
		panic(err)
	}

	report, err := tmpl.Render(context.Background(), map[string]any{
		"customer": map[string]any{"name": "Nawaf"},
		"invoice": map[string]any{
			"total": 250,
			"items": []map[string]any{{"name": "Assessment"}},
		},
	})
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile("invoice.docx", report, 0o644); err != nil {
		panic(err)
	}
}
```

المثال الكامل ينشئ تقريرًا عربيًا اصطناعيًا يحتوي على جدول وصورة ورابط وشروط وحلقات: [`examples/complete`](examples/complete).<br>
The complete example generates a synthetic Arabic report with a table, image, link, conditions, and loops: [`examples/complete`](examples/complete).

<a id="template-language"></a>

## لغة القالب · Template language

تستخدم الأوامر المحددات `[[` و`]]` افتراضيًا، ويمكن تغييرها من `Options`.<br>
Commands use `[[` and `]]` by default; both delimiters are configurable through `Options`.

| الأمر · Command | الغرض | Purpose |
| --- | --- | --- |
| `[[value]]`, `[[INS value]]`, `[[= value]]` | إدراج نص | Insert text |
| `[[EXEC name = expression]]`, `[[! name = expression]]` | تعيين قيمة محلية دون إخراج | Assign a local value without output |
| `[[SET name = expression]]` | صيغة تعيين صريحة | Explicit assignment alias |
| `[[IF expression]]` / `[[ELSE]]` / `[[END-IF]]` | شرط على فقرة أو صف جدول | Conditional paragraph or table row |
| `[[FOR item IN values]]` / `[[END-FOR item]]` | تكرار فقرات أو صفوف جداول | Repeat paragraphs or table rows |
| `[[IMAGE expression]]` | إدراج صورة مضمنة | Insert an inline image |
| `[[LINK expression]]` | إدراج رابط خارجي | Insert an external hyperlink |
| `[[HTML expression]]` | إدراج Word HTML altChunk | Insert a Word HTML altChunk |
| `[[RAW-XML expression]]` | إدراج OOXML موثوق عند تفعيله | Insert trusted OOXML when enabled |
| `[[QUERY query text]]` | طلب البيانات عبر callback في Go | Resolve data through a Go callback |
| `[[ALIAS name INS expression]]`, `[[*name]]` | تعريف أمر كامل وإعادة استخدامه | Define and reuse a complete command |

ينبغي أن تكون علامات `IF` و`FOR` البنيوية في فقرة مستقلة أو صف جدول مستقل. ويجب أن يكون `HTML` و`RAW-XML` في فقرة مستقلة. يعالج نَمَط الأوامر التي يقسمها Word بين عدة XML runs، كما يعالج الانقسام المدعوم بين الفقرات المتجاورة.

Structural `IF` and `FOR` markers should occupy their own paragraph or table row. `HTML` and `RAW-XML` must occupy their own paragraph. Namat normalizes commands split by Word across multiple XML runs and supported adjacent paragraphs.

### التعبيرات · Expressions

لغة التعبيرات مخصصة للوصول إلى البيانات والحسابات المحدودة، وليست بيئة لتنفيذ شفرات عامة.<br>
The expression language is designed for data access and bounded calculations—not arbitrary code execution.

- Maps, structs, JSON field names, pointers, arrays, slices, strings, and indexes.
- Property access, optional chaining, and null coalescing with `??`.
- Arithmetic, comparison, equality, logical, and unary operators.
- Ternary expressions: `condition ? yes : no`.
- Array and object literals: `[1, 2]`, `{ url: url, label: name }`.
- Template strings: `` `Score: ${score}` ``.
- String helpers such as `.slice()`, `.trim()`, `.toUpperCase()`, `.contains()`, and `.startsWith()`.
- Collection helpers such as `.join()`, `.includes()`, and `.length`.
- Explicitly registered Go functions.

متغيرات الحلقة تبدأ بـ `$`، ويمثل `$idx` الفهرس الذي يبدأ من الصفر. تحل دوال Go المسجلة محل JavaScript helpers، ويمكن اختبارها وقياسها ومراجعتها كأي شفرة Go عادية.<br>
Loop variables use the `$` prefix, and `$idx` is zero-based. Registered Go functions replace JavaScript helpers and can be tested, profiled, and audited like ordinary Go code.

## المحتوى الغني · Rich content

### الصور · Images

يدعم نَمَط PNG وJPEG وGIF وSVG كصور inline، مع أبعاد بالسنتيمتر، ودوران، ونص بديل، وتعليق اختياري. ويمكن لصورة SVG توفير thumbnail بصيغة PNG أو JPEG أو GIF للتوافق مع الإصدارات القديمة من Word والمعاينات.<br>
Namat supports PNG, JPEG, GIF, and SVG as inline drawings, with centimeter dimensions, rotation, alt text, and optional captions. SVG values may include a PNG, JPEG, or GIF thumbnail for older Word versions and previews.

```go
namat.Image{
	Data:      pngBytes,
	Extension: "png",
	Width:     8,
	Height:    4,
	Alt:       "Quarterly chart",
}
```

### الروابط وHTML وOOXML · Links, HTML, and OOXML

- يقبل `LINK` قيمة `namat.Link` أو object expression، ويسمح افتراضيًا بـ `http` و`https` و`mailto` فقط.<br>
  `LINK` accepts `namat.Link` or an object expression and allows only `http`, `https`, and `mailto` by default.
- يستخدم `HTML` آلية OOXML `altChunk` التي يدعمها Microsoft Word، وقد يختلف استيرادها في LibreOffice وGoogle Docs.<br>
  `HTML` uses OOXML `altChunk`, supported by Microsoft Word but imported inconsistently by LibreOffice and Google Docs.
- يكون `RAW-XML` معطّلًا افتراضيًا لأنه يتجاوز escaping، ولا ينبغي تفعيله إلا للقوالب والقيم الموثوقة.<br>
  `RAW-XML` is disabled by default because it bypasses escaping and should be enabled only for trusted templates and values.

```text
[[LINK ({ url: project.url, label: project.name })]]
```

```go
options := namat.Options{AllowRawXML: true}
```

## واجهة المكتبة · Library API

### Readers and writers

```go
tmpl, err := namat.CompileReader(reader, options)
err = tmpl.RenderTo(ctx, writer, data)
```

يبقى `RenderTo` ذريًا من منظور الكاتب: تُبنى حزمة DOCX في الذاكرة أولًا، فلا يترك الخطأ مستندًا جزئيًا.<br>
`RenderTo` remains transactional from the writer's perspective: the DOCX package is assembled in memory first, so an error does not leave a partial document.

### الفحص والبيانات الوصفية · Inspection and metadata

```go
commands, err := namat.ListCommands(templateBytes, options)
metadata, err := namat.GetMetadata(documentBytes)
```

يعيد `GetMetadata` خصائص Word المخزنة؛ ولا يدّعي إعادة حساب عدد الصفحات أو الكلمات المعتمد على التخطيط. ويحجب أمر CLI تعبيرات القالب افتراضيًا عند الفحص حفاظًا على الخصوصية.<br>
`GetMetadata` returns cached Word properties; it does not claim to recalculate layout-dependent page or word counts. The CLI inspector hides template expressions by default for privacy.

### Query resolver

يمكن للقالب إعلان `QUERY` واحد يمرره نَمَط دون تعديل إلى callback يملكه التطبيق. لا تفسر المكتبة SQL أو GraphQL ولا تفتح اتصالًا شبكيًا بنفسها.<br>
A template may declare one `QUERY`, passed unchanged to an application-owned callback. Namat does not interpret SQL or GraphQL and never opens a network connection itself.

```go
options := namat.Options{
	QueryResolver: func(ctx context.Context, query string) (any, error) {
		return database.ReportData(ctx, query)
	},
}
```

## أداة سطر الأوامر · CLI

أداة CLI اختيارية وتستخدم المكتبة نفسها:<br>
The optional CLI uses the same library:

```text
namat inspect template.docx
namat inspect --json template.docx
namat metadata document.docx
namat render --data data.json --out report.docx template.docx
```

يرفض `render` استبدال ملف موجود ما لم يُمرر `--force` صراحة، ويكتب النتيجة عبر إعادة تسمية ذرية لملف مؤقت.<br>
`render` refuses to overwrite an existing file unless `--force` is explicit and writes through an atomic temporary-file rename.

```bash
go build -trimpath -ldflags="-s -w" ./cmd/namat
```

## الأمان والاعتمادية · Safety and reliability

لا تمنح قوالب نَمَط وصولًا تلقائيًا إلى نظام الملفات أو العمليات أو متغيرات البيئة أو reflection أو الشبكة. ولا يصل القالب إلا إلى البيانات الممررة إلى `Render` والدوال التي يسجلها التطبيق صراحة.

Namat templates receive no implicit access to the filesystem, processes, environment variables, reflection, or the network. A template can access only the data passed to `Render` and functions explicitly registered by the host application.

تشمل `Options` حدودًا لحجم القالب، وحجم كل ZIP part، وإجمالي المحتوى غير المضغوط، وعدد الأجزاء، وحجم الخرج، وإجمالي تكرارات الحلقات، ومدة الإنشاء. ويرفض القارئ مسارات traversal والأجزاء المكررة والحزم غير الصالحة.<br>
`Options` provides limits for template bytes, each ZIP part, total uncompressed content, part count, output bytes, aggregate loop iterations, and render duration. The reader rejects traversal paths, duplicate parts, and invalid packages.

راجع [سياسة الأمان · Security policy](SECURITY.md) قبل قبول قوالب من مستخدمين غير موثوقين.

## التوافق · Compatibility

| Capability | Status | Notes · ملاحظات |
| --- | --- | --- |
| DOCX round trip | Supported | يحافظ على الأجزاء والوسائط غير المعدلة · Preserves untouched parts and media |
| DOCM round trip | Supported | ينسخ أجزاء VBA دون تعديل · Copies VBA parts unchanged |
| Split Word runs | Supported | أوامر موزعة بين runs وفقرات مدعومة · Commands split across runs and supported paragraphs |
| Nested conditions and loops | Supported | فقرات وصفوف جداول · Paragraphs and table rows |
| Headers, footers, notes | Supported | معالجة أجزاء `word/*.xml` ذات الصلة · Processes relevant `word/*.xml` parts |
| PNG, JPEG, GIF, SVG | Supported | صور inline مع SVG fallback اختياري · Inline images with optional SVG fallback |
| Hyperlinks | Supported | علاقات خارجية مع allowlist للبروتوكولات · External relationships with a scheme allowlist |
| HTML altChunk | Supported | في المستند الرئيسي فقط · Main document only |
| Literal OOXML | Opt-in | للمدخلات الموثوقة فقط · Trusted input only |
| Arbitrary JavaScript | Not supported | يُستبدل بدوال Go مسجلة · Replaced by registered Go functions |
| Floating images | Not generated | صور inline أكثر قابلية للنقل · Inline drawings are more portable |

للتفاصيل الدقيقة وحدود السلوك، راجع [مصفوفة التوافق الكاملة](docs/COMPATIBILITY.md).<br>
For exact behavior and limitations, see the [full compatibility matrix](docs/COMPATIBILITY.md).

## الجودة والأداء · Quality and performance

| Quality gate | Current guarantee |
| --- | --- |
| Statement coverage | **100%** independently for the root library, expression engine, CLI, and complete example |
| Behavioral coverage | Every documented core capability maps to an automated test |
| Platforms | CI runs on Linux, Windows, and macOS |
| Concurrency | Race detector plus concurrent rendering tests |
| Robustness | Fuzz targets for commands, ZIP packages, reports, and expressions |
| Test data | Generated, synthetic, English, and product-neutral fixtures |

```bash
go test ./...
go test -race ./...
go test -shuffle=on -count=3 ./...
go vet ./...
go test -run '^$' -bench . -benchmem ./...
```

يفصل نَمَط الترجمة عن الإنشاء: ترجم كل قالب مرة واحدة وأعد استخدام `Template` غير القابل للتغيير عبر goroutines. يسجل المشروع baseline مؤرخًا للقياس، لكنه لا يقدمه كضمان صالح لكل الأجهزة.<br>
Namat separates compilation from rendering: compile each template once and reuse the immutable `Template` across goroutines. The project records a dated benchmark baseline, but does not present it as a cross-machine guarantee.

راجع [منهجية الأداء](docs/PERFORMANCE.md) و[آخر baseline](docs/BENCHMARKS.md).

## ما الذي يتضمنه المستودع؟ · What's included

```text
go-namat/
├── cmd/namat/          optional native CLI
├── docs/               architecture, compatibility, testing, and performance
├── examples/complete/  complete synthetic report example
├── internal/expr/      native expression lexer, parser, and evaluator
├── testdata/            minimized synthetic fuzz regressions
└── *.go                 public API and OOXML rendering engine
```

<a id="documentation"></a>

## التوثيق · Documentation

| Document | العربية · English |
| --- | --- |
| [Architecture](docs/ARCHITECTURE.md) | بنية المحرك وحدود المكونات · Engine structure and component boundaries |
| [Compatibility](docs/COMPATIBILITY.md) | ما تدعمه المكتبة وحدوده · Supported behavior and limitations |
| [Core feature coverage](docs/FEATURE_COVERAGE.md) | ربط كل ميزة أساسية باختبار آلي · Mapping every core feature to automated evidence |
| [Testing](docs/TESTING.md) | تنظيم الاختبارات وسياسة عزل البيانات · Test layout and data-isolation policy |
| [Internationalization](docs/INTERNATIONALIZATION.md) | Unicode وRTL ومسؤوليات locale · Unicode, RTL, and locale responsibilities |
| [Performance](docs/PERFORMANCE.md) | نموذج الأداء وكيفية القياس · Performance model and benchmarking guidance |
| [Roadmap](docs/ROADMAP.md) | الطريق إلى الإصدار 1.0 · Path to v1.0 |
| [Security](SECURITY.md) | نموذج الثقة والإبلاغ عن الثغرات · Trust model and vulnerability reporting |
| [Changelog](CHANGELOG.md) | التغييرات الملحوظة · Notable changes |

## خارطة الطريق · Roadmap

اكتملت المراحل الأساسية للمحرك النصي والمحتوى الغني وتوافق التأليف وتقوية الإنتاج. تركز الأعمال المتبقية قبل `v1.0` على fixtures عامة للتوافق البصري، ودليل ترحيل بإصدارات دلالية، وإصدارات موقعة لأداة CLI. تبقى أي تكاملات أو عمليات ترحيل خاصة بمنتج معين خارج هذا المستودع المستقل.

The native text engine, rich-content, authoring-compatibility, and production-hardening foundations are complete. Remaining pre-`v1.0` work focuses on public visual-compatibility fixtures, a semantic-versioned migration guide, and signed CLI releases. Product-specific integrations and migrations remain outside this standalone repository.

See the detailed [roadmap](docs/ROADMAP.md).

## المساهمة · Contributing

نرحب بالإصلاحات والتحسينات التي تحافظ على كون المشروع Go أصيلًا، دون JavaScript runtime أو برامج مساعدة أو اعتماد شبكي خفي. يجب أن تستخدم الاختبارات مستندات وبيانات اصطناعية فقط، وألا تتضمن قوالب عملاء أو بيانات إنتاج أو معلومات سرية.

Fixes and improvements are welcome when they preserve the pure-Go design, with no JavaScript runtime, helper executable, or hidden network dependency. Tests must use synthetic documents and values only—never customer templates, production data, or confidential information.

اقرأ [دليل المساهمة](CONTRIBUTING.md) قبل إرسال التغييرات.<br>
Read the [contribution guide](CONTRIBUTING.md) before submitting changes.

## الاسم · The name

**نَمَط** كلمة عربية تعني pattern أو mode أو template؛ اسم قصير يصف المكتبة مباشرة ويظل واضحًا داخل Go imports.<br>
**Namat (نَمَط)** is the Arabic word for a pattern, mode, or template—a short name that describes the library and stays clear in Go imports.

## الترخيص · License

مرخّص بموجب [MIT](LICENSE).<br>
Licensed under the [MIT License](LICENSE).

<div align="center">

**صمّم في Word. أنشئ باستخدام Go. · Author in Word. Render with Go.**

</div>
