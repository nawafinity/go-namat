<div align="center" dir="rtl">

<h1>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/brand/namat-logo-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="assets/brand/namat-logo-light.svg">
    <img src="assets/brand/namat-logo-light.svg" width="360" alt="نَمَط · Namat">
  </picture>
</h1>

**محرك قوالب أصيل بلغة Go لمستندات Microsoft Word**

صمّم التقرير في Word، وأنشئه باستخدام Go، ووزّعه ضمن تطبيق أصيل واحد.

[![Go](https://img.shields.io/badge/Go-1.23%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/nawafinity/go-namat.svg)](https://pkg.go.dev/github.com/nawafinity/go-namat)
[![CI](https://github.com/nawafinity/go-namat/actions/workflows/ci.yml/badge.svg)](https://github.com/nawafinity/go-namat/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/statement%20coverage-100%25-brightgreen)](docs/TESTING.md)
[![License](https://img.shields.io/badge/license-MIT-2ea44f)](LICENSE)

[English](README.md) · **العربية**

[لماذا نَمَط؟](#why-namat) · [البداية السريعة](#quick-start) · [لغة القوالب](#template-language) · [التوثيق](#documentation)

</div>

<div dir="rtl">

يحوّل نَمَط ملفات DOCX وDOCM إلى تقارير مبنية على البيانات، مع الحفاظ على
البنية التي صُممت في Word: الأنماط، والصور، والرؤوس، والتذييلات، ووحدات
الماكرو، وأجزاء OOXML التي لا تحتاج إلى تعديل.

المكتبة مكتوبة بالكامل بلغة Go، ولا تعتمد على أي حزمة Go خارجية، ولذلك تندمج
بسلاسة داخل الخدمات وأدوات سطر الأوامر وتطبيقات سطح المكتب.

> [!IMPORTANT]
> نَمَط حاليًا في مرحلة **ما قبل الإصدار 1.0**. الواجهة الأساسية ولغة القوالب
> قابلة للاستخدام ومغطاة باختبارات دقيقة، وتبدأ ضمانات التوافق وفق الإصدارات
> الدلالية مع `v1.0`.

<a id="why-namat" name="why-namat"></a>

## ✨ لماذا نَمَط؟

<table>
  <tr>
    <td width="50%" dir="rtl">
      <img src="assets/icons/native.svg" width="30" alt=""><br>
      <strong>Go أصيل</strong><br>
      ضمّن مكتبة مركزة واحدة، ووزّع تطبيقًا أصيلًا واحدًا.
    </td>
    <td width="50%" dir="rtl">
      <img src="assets/icons/word-first.svg" width="30" alt=""><br>
      <strong>Word أولًا</strong><br>
      أنشئ التخطيط والأنماط في Word بدل إعادة بنائها برمجيًا.
    </td>
  </tr>
  <tr>
    <td width="50%" dir="rtl">
      <img src="assets/icons/safe.svg" width="30" alt=""><br>
      <strong>آمن منذ التصميم</strong><br>
      استخدم تعبيرات محدودة وأخطاء مصنفة ومهلات وحدود موارد صريحة.
    </td>
    <td width="50%" dir="rtl">
      <img src="assets/icons/structure.svg" width="30" alt=""><br>
      <strong>إدراك لبنية المستند</strong><br>
      طبّق الشروط والحلقات المتداخلة على الفقرات وصفوف الجداول الكاملة.
    </td>
  </tr>
  <tr>
    <td width="50%" dir="rtl">
      <img src="assets/icons/rich-content.svg" width="30" alt=""><br>
      <strong>محتوى غني</strong><br>
      أنشئ النصوص والصور والروابط ومقاطع HTML وفواصل الأسطر وOOXML الموثوق.
    </td>
    <td width="50%" dir="rtl">
      <img src="assets/icons/concurrent.svg" width="30" alt=""><br>
      <strong>إعادة استخدام متزامنة</strong><br>
      حضّر القالب مرة واحدة، ثم أنشئ منه التقارير بأمان عبر عدة goroutines.
    </td>
  </tr>
</table>

<a id="quick-start" name="quick-start"></a>

## 🚀 البداية السريعة

### التثبيت

```bash
go get github.com/nawafinity/go-namat
```

### إنشاء قالب Word

أنشئ ملف DOCX أو DOCM في Word، ثم ضع الأوامر مباشرة داخل المستند:

```text
فاتورة العميل: [[customer.name]]

[[#if invoice.total > 0]]
الإجمالي: [[invoice.total]]
[[#else]]
لا يوجد رصيد مستحق.
[[/if]]

[[#each invoice.items as item]]
[[loop.number]]. [[item.name]]
[[/each]]
```

### تحضير القالب مرة واحدة وإنشاء تقارير متعددة

```go
package main

import (
	"context"
	"os"

	"github.com/nawafinity/go-namat"
)

func main() {
	source, err := os.ReadFile("invoice-template.docx")
	if err != nil {
		panic(err)
	}

	tmpl, err := namat.Compile(source, namat.Options{})
	if err != nil {
		panic(err)
	}

	report, err := tmpl.Render(context.Background(), map[string]any{
		"customer": map[string]any{"name": "Acme"},
		"invoice": map[string]any{
			"total": 250.00,
			"items": []map[string]any{
				{"name": "Assessment"},
				{"name": "Implementation"},
			},
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

يقدّم المسار [`examples/complete`](examples/complete) مثالًا مستقلًا ينشئ تقريرًا
عربيًا يتضمن جدولًا وصورة ورابطًا وشروطًا وحلقات.
وللاختبار المكثف، يضع [`examples/advanced`](examples/advanced) جداول مهام داخل
صفوف مشاريع متكررة، ويستدعي الدوال المسجلة من مستويي الحلقات، بما في ذلك دوال
تعيد روابط ورسومًا مولّدة.

<a id="template-language" name="template-language"></a>

## 🧩 لغة القوالب

تستخدم الأوامر المحددين `[[` و`]]` افتراضيًا، ويمكن تغييرهما من خلال
`Options`. العقد الوحيد المدعوم هو `v1`، ويمكن للتطبيق تثبيته صراحة عبر
`LanguageVersion: namat.LanguageVersionV1`. لا يكتشف نَمَط صيغة قديمة تلقائيًا
ولا يعود إليها.

| الأمر | الغرض |
| --- | --- |
| `[[expression]]` | إدراج قيمة بسيطة |
| `[[#let name = expression]]` | تعريف قيمة ثابتة ضمن النطاق دون إخراج |
| `[[#if expression]]` / `[[#else]]` / `[[/if]]` | إظهار كتلة شرطية من الفقرات أو صفوف الجداول |
| `[[#each values as item]]` / `[[/each]]` | تكرار فقرات أو صفوف جداول كاملة |
| `[[@image expression]]` | إدراج صورة مضمنة |
| `[[@link expression]]` | إدراج رابط خارجي |
| `[[@html expression]]` | إدراج Word HTML altChunk |
| `[[@raw-xml expression]]` | إدراج OOXML موثوق عند تفعيله صراحة |

يجب أن تكون الأوامر البنيوية و`#let` في فقرة مستقلة أو صف جدول مستقل، وأن يكون
`@html` و`@raw-xml` في فقرة مستقلة.

قد يقسم Word الأمر الواحد بين عدة XML runs، وأحيانًا بين فقرات متجاورة.
يعيد نَمَط تجميع الأوامر المجزأة المدعومة قبل تحضير القالب.

### التعبيرات

صُممت لغة التعبيرات الأصلية للوصول إلى البيانات وتنفيذ حسابات محدودة، وليست
بيئة لتشغيل شفرات عامة. وهي تدعم:

- الخرائط والبنى (`struct`) وأسماء حقول JSON والمؤشرات والمصفوفات والشرائح
  (`slice`) والنصوص والفهارس؛
- الوصول إلى الخصائص والوصول الاختياري مثل `customer?.name`؛
- عمليات حسابية ومقارنات ومساواة ومنطق ذات أنواع صارمة؛
- المصفوفات والكائنات: `[1, 2]` و`{ url: url, label: name }`؛
- المرشحات مثل `name | trim | upper` و`customer?.name | default("—")`؛
- أعداد صحيحة موقعة وغير موقعة وقيم `namat.Decimal` الدقيقة؛
- دوال Go مسجلة صراحة مع وصف لأنواع المدخلات والناتج.

لمتغير الحلقة اسم عادي، وتوجد معلوماتها في `loop.index` و`loop.number`
و`loop.first` و`loop.last` و`loop.parent`. تقبل الشروط `bool` فقط. ويعد الحقل
المفقود و`null` عند الإدراج وخلط الأنواع في `+` والمقارنة بين أنواع غير متوافقة
أخطاءً ما لم تعالج صراحة. ولا يمكن استدعاء قيم الدوال الموجودة في البيانات؛
بل تسجل الدوال المقصودة عبر `Options.Functions` بوصف `FunctionSpec`.

## 🖼️ المحتوى الغني

### الصور

يقبل الأمر `@image` قيمة `namat.Image` أو كائنًا بالحقول المكافئة. يدعم نَمَط
PNG وJPEG وGIF وSVG كصور مضمنة، مع تحديد الأبعاد بالسنتيمتر والدوران والنص
البديل والتعليق الاختياري.

```go
namat.Image{
	Data:      pngBytes,
	Extension: "png",
	Width:     8,
	Height:    4,
	Alt:       "Quarterly chart",
}
```

يمكن لصورة SVG أن تتضمن صورة مصغرة بصيغة PNG أو JPEG أو GIF لدعم إصدارات Word
القديمة ومعاينات المستندات.

### الروابط وHTML وOOXML

- يقبل `@link` قيمة `namat.Link` أو تعبير كائن، ويسمح افتراضيًا ببروتوكولات
  `http` و`https` و`mailto` فقط.
- يستخدم `@html` آلية OOXML `altChunk`. يدعمها Microsoft Word، بينما قد يختلف
  استيرادها في LibreOffice وGoogle Docs. تُضمّن حمولات HTML وSVG دون تنقية؛
  استخدم محتوى موثوقًا أو منقّى داخل التطبيق.
- يكون `@raw-xml` معطلًا افتراضيًا لأنه يتجاوز escaping. لا تفعّله إلا عندما
  يكون القالب والقيم المدرجة موثوقة.

```text
[[@link ({ url: project.url, label: project.name })]]
```

```go
options := namat.Options{AllowRawXML: true}
```

## 🧰 واجهة المكتبة

### القراءة والكتابة

```go
tmpl, err := namat.CompileReader(reader, options)
err = tmpl.RenderTo(ctx, writer, data)
```

يتعامل `RenderTo` مع الكتابة بوصفها عملية ذرية: يبني نَمَط حزمة DOCX كاملة في
الذاكرة قبل الكتابة، ولذلك لا يترك فشل الإنشاء مستندًا جزئيًا.

### الفحص والبيانات الوصفية

```go
commands, err := namat.ListCommands(templateBytes, options)
metadata, err := namat.GetMetadata(documentBytes)
```

يعيد `GetMetadata` خصائص Word المخزنة، ولا يدّعي إعادة حساب عدد الصفحات أو
الكلمات المعتمد على التخطيط. كما يحجب فاحص CLI تعبيرات الأوامر افتراضيًا للحد
من كشف البيانات دون قصد.

### أرقام JSON الدقيقة

استخدم `namat.DecodeJSON` لبيانات JSON. فهو يحافظ على `int64` و`uint64`
والقيم العشرية الدقيقة بدل تحويل كل الأرقام إلى `float64`.

## 💻 واجهة سطر الأوامر

تُبنى أداة CLI الاختيارية باستخدام الواجهة العامة نفسها:

```text
namat inspect template.docx
namat inspect --json template.docx
namat lint --data data.json template.docx
namat metadata document.docx
namat render --data data.json --out report.docx template.docx
```

يرفض `render` استبدال ملف موجود ما لم يُمرر `--force` صراحة. ويكتب ملفًا
مؤقتًا داخل مجلد الوجهة ويزامنه، ثم يستخدم hard link ذرية لا تسمح بالاستبدال
بحيث لا يستبدل ملفًا أنشئ بالتزامن. ومع `--force`، يحاول أولًا استخدام استبدال
ذري تدعمه المنصة. وإذا رفضت المنصة ذلك، يستخدم نسخة احتياطية جانبية قابلة
للاستعادة أثناء تثبيت الملف الجديد؛ وهذه الآلية البديلة ليست عملية استبدال
ذرية واحدة.

```bash
go build -trimpath -ldflags="-s -w" ./cmd/namat
```

## 🛡️ نموذج الأمان

تصل القوالب إلى البيانات الممررة إلى `Render` ودوال Go التي يسجلها التطبيق
صراحة. ولا يمكنها استدعاء قيم الدوال الموجودة في بيانات التقرير، كما لا تحصل
تلقائيًا على صلاحيات لنظام الملفات أو العمليات أو متغيرات البيئة أو واجهات
الانعكاس أو الشبكة. ومع القوالب غير الموثوقة، لا تعرض إلا دوال راجعها التطبيق
عمدًا.

القوالب المحضّرة غير قابلة للتغيير وآمنة للاستخدام المتزامن. لكن هذا الضمان
لا يجعل استدعاءات التطبيق أو الحالة التي تلتقطها آمنة تلقائيًا؛ إذ يجب أن
تدعم `Functions` و`ErrorHandler` مستوى التزامن الذي يستخدمه
التطبيق.

توفّر `Options` حدودًا لكل من:

- حجم القالب المضغوط؛
- حجم كل جزء داخل ZIP؛
- الحجم الإجمالي غير المضغوط للحزمة؛
- عدد أجزاء الحزمة؛
- حجم الملف الناتج؛
- إجمالي تكرارات الحلقات؛
- طول التعبير وعدد رموزه وعمق AST وإجمالي خطوات التقييم؛
- مدة إنشاء التقرير.

يعمل سياق إنشاء التقرير و`Timeout` بصورة تعاونية؛ إذ يفحصهما نَمَط بين
عمليات الإنشاء، لكنه لا يستطيع إيقاف استدعاء تابع للتطبيق وهو عالق. ينبغي
تستقبل الدوال المسجلة سياق التصيير، ويجب أن تعود الدوال ومعالجات الأخطاء بسرعة
وأن تطبق حدودها الخاصة على العمليات التابعة. راجع [سياسة الأمان](SECURITY.md)
للقيم الافتراضية ونموذج الثقة الكامل.

يرفض قارئ الحزمة مسارات traversal والمدخلات المكررة والأجزاء المتجاوزة للحدود
والحزم غير الصالحة. راجع [سياسة الأمان](SECURITY.md) قبل قبول قوالب من مستخدمين
غير موثوقين.

## 🔄 التوافق

| القدرة | الحالة | الملاحظات |
| --- | --- | --- |
| معالجة DOCX | مدعومة | يحافظ على الأجزاء والوسائط غير المعدلة |
| معالجة DOCM | مدعومة | ينسخ أجزاء VBA دون تعديل |
| الأوامر المقسمة بين Word runs | مدعومة | تشمل حالات الانقسام المدعومة بين الفقرات المتجاورة |
| الشروط والحلقات المتداخلة | مدعومة | داخل الفقرات وصفوف الجداول الكاملة |
| الرؤوس والتذييلات والملاحظات | مدعومة | يعالج أجزاء `word/*.xml` ذات الصلة |
| PNG وJPEG وGIF وSVG | مدعومة | صور مضمنة مع صورة بديلة اختيارية لـSVG |
| الروابط الخارجية | مدعومة | مع قائمة سماح للبروتوكولات |
| HTML altChunk | مدعومة | داخل المستند الرئيسي فقط |
| OOXML الخام الصريح | اختياري | عبر `@raw-xml` وللمدخلات الموثوقة فقط |
| JavaScript العام | غير مدعوم | يُستبدل بدوال Go مسجلة |
| الصور العائمة | لا ينشئها المحرك | الصور المضمنة أكثر قابلية للنقل |

راجع [مصفوفة التوافق الكاملة](docs/COMPATIBILITY.md) للاطلاع على السلوك الدقيق
والقيود المعروفة.

## 📊 الجودة والأداء

| بوابة الجودة | الضمان الحالي |
| --- | --- |
| تغطية العبارات البرمجية | **100%** بصورة مستقلة لكل حزمة Go موزعة، وتفرضها CI |
| التغطية السلوكية | ترتبط الميزات الأساسية المتحقق منها بأدلة آلية، بينما يبقى التوافق البصري داخل العملاء بوابة غير مكتملة قبل v1 |
| المنصات | يعمل CI على Linux وWindows وmacOS باستخدام Go 1.23 وخط إصدار Go الحالي |
| التزامن | كاشف سباقات البيانات واختبارات إنشاء التقارير المتزامن |
| المتانة | يجري CI اختبارات fuzz للأوامر وحزم ZIP والتقارير والتعبيرات، ويشترط نجاح LibreOffice في تحويل fixture العامة |
| بيانات الاختبار | بيانات مولّدة واصطناعية ومحايدة وغير مرتبطة بأي منتج، مع ترخيص صريح للـfixtures العامة |

```bash
go test ./...
go test -race ./...
go test -shuffle=on -count=3 ./...
go vet ./...
go test -run '^$' -bench . -benchmem ./...
```

لأحمال العمل المرتفعة، حضّر كل قالب مرة واحدة وأعد استخدام `Template` غير
القابل للتغيير عبر عدة مسارات تنفيذ. راجع [دليل الأداء](docs/PERFORMANCE.md)
و[القياس المرجعي المؤرخ](docs/BENCHMARKS.md).

## 🏗️ بنية المستودع

```text
go-namat/
├── assets/             project mark and feature icons
├── cmd/namat/          optional native CLI
├── docs/               architecture, compatibility, testing, and performance
├── examples/complete/  complete synthetic report example
├── examples/advanced/  nested tables and functions inside loops
├── internal/engine/    private DOCX compiler, renderer, and focused tests
├── internal/expr/      native expression lexer, parser, and evaluator
├── namat.go             stable public package facade
├── README_AR.md         complete Arabic documentation
├── types.go             public options, values, commands, and errors
└── *_test.go            public API examples and repository policy checks
```

يمنع حد `internal` مستخدمي المكتبة من الاعتماد على تفاصيل التنفيذ، مع الحفاظ
على مسار استيراد واضح ومختصر:

```go
import "github.com/nawafinity/go-namat"
```

<a id="documentation" name="documentation"></a>

## 📚 التوثيق

| المستند | المحتوى |
| --- | --- |
| [Architecture](docs/ARCHITECTURE.md) | مسار إنشاء التقرير وحدود المكونات |
| [Compatibility](docs/COMPATIBILITY.md) | السلوك المدعوم والقيود المعروفة |
| [Core feature coverage](docs/FEATURE_COVERAGE.md) | الدليل الآلي لكل ميزة أساسية |
| [Testing](docs/TESTING.md) | تنظيم الاختبارات وأوامر الجودة وسياسة عزل البيانات |
| [Internationalization](docs/INTERNATIONALIZATION.md) | Unicode وRTL ومسؤوليات التنسيق المحلي |
| [Performance](docs/PERFORMANCE.md) | نموذج الأداء وإرشادات القياس |
| [Roadmap](docs/ROADMAP.md) | الأعمال المتبقية قبل الإصدار 1.0 |
| [Migration](docs/MIGRATION.md) | مسودة عقد الانتقال من مرحلة ما قبل v1 وقائمة تحقق للمستخدمين |
| [Releasing](docs/RELEASING.md) | بوابات الإصدار والتحقق داخل العملاء وتوقيع الملفات |
| [Security](SECURITY.md) | نموذج الثقة والإبلاغ عن الثغرات |
| [Code of Conduct](CODE_OF_CONDUCT.md) | معايير المجتمع وآلية الإبلاغ |
| [Changelog](CHANGELOG.md) | التغييرات المهمة في المشروع |

## 🗺️ خارطة الطريق

اكتملت الأسس المتعلقة بمحرك النصوص الأصلي والمحتوى الغني وتوافق التأليف
وتقوية الاستخدام الإنتاجي. تركز الأعمال المتبقية قبل الإصدار 1.0 على عينات
عامة للتوافق البصري، ودليل ترحيل مرتبط بالإصدارات الدلالية، وإصدارات موقعة
لأداة CLI.

تبقى التكاملات وعمليات الترحيل الخاصة بأي منتج خارج هذا المستودع المستقل.
راجع [خارطة الطريق التفصيلية](docs/ROADMAP.md).

## 🤝 المساهمة

نرحب بالمساهمات التي تحافظ على معمارية Go المركزة، وسياسة التبعيات الصريحة،
واستقرار الواجهة العامة. ويجب أن تستخدم الاختبارات مستندات وقيمًا اصطناعية
فقط، دون قوالب عملاء أو بيانات إنتاج أو بيانات اعتماد أو معلومات سرية.

اقرأ [دليل المساهمة](CONTRIBUTING.md) و[مدونة السلوك](CODE_OF_CONDUCT.md) قبل
إرسال أي تغيير.

## 💡 الإلهام

استُلهم نَمَط من أسلوب التأليف الطبيعي القائم على Word في مشروع
[docx-templates](https://github.com/guigrpa/docx-templates)، ثم أعاد تصور هذا
الأسلوب بما يلائم نظام الأنواع والتزامن وسهولة التوزيع في Go، مع محرك تعبيرات
ومسار OOXML وواجهة برمجية وضوابط أمان خاصة به.

## 🏷️ الاسم

**نَمَط** كلمة عربية تعني الأسلوب أو القالب أو الهيئة المتكررة؛ وهو اسم قصير
يصف المكتبة مباشرة ويظل واضحًا داخل عبارات الاستيراد في Go.

يجمع الشعار بين مستند مطوي وأشكال هندسية متداخلة؛ في إشارة بصرية إلى تحوّل
القوالب المنظمة إلى تقارير مكتملة.

## 📄 الترخيص

تتوفر نَمَط بموجب [رخصة MIT](LICENSE).

<div align="center">

**صمّم في Word. أنشئ باستخدام Go.**

</div>

</div>
