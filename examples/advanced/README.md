# Advanced nested-loop example

This example exercises a three-level data model: departments, projects, and
tasks. The DOCX contains a project table whose repeated rows contain nested task
tables. Registered Go functions are called from inside both loop levels.

هذا المثال مصمم كاختبار ضغط: حلقة أقسام تحتوي جدول مشاريع، وكل صف مشروع يحتوي
جدول مهام متداخل وحلقة أخرى، مع استدعاء دوال Go من داخل المستويين.

From the repository root:

```powershell
go run ./examples/advanced --out dist/manual/advanced-report.docx
```

To force a multi-page stress test, repeat the complete department dataset:

```powershell
go run ./examples/advanced --repeat 12 --out dist/manual/advanced-stress.docx
```

`--repeat` accepts values from 1 to 50. A value of 12 produces 24 departments,
36 projects, 96 tasks, 36 generated charts, and 36 hyperlinks.

The template demonstrates:

- nested `#each` blocks and a nested Word table;
- `#if / #else`, immutable `#let`, and `loop` metadata at both loop depths;
- text-returning functions: `upper`, `money`, `statusLabel`, `taskBadge`, and
  `hours`;
- aggregate functions over loop data: `count`, `sumBudgets`, and `sumHours`;
- a function returning `namat.Link` inside a project loop;
- a function returning a generated `namat.Image` chart inside that loop;
- RTL paragraphs, merged cells, fixed table layouts, and generated
  part-local relationships for images and hyperlinks.

Edit `data.json` to add departments, projects, tasks, budgets, or progress
values. Edit `template.docx` in Word, then rerun the command.

للتجربة: عدّل البيانات في `data.json` أو أضف مشروعًا ومهامًا جديدة، ثم نفّذ
الأمر نفسه وافتح الملف الناتج `dist/manual/advanced-report.docx` في Word أو
LibreOffice.

ولفحص تعدد الصفحات والصفحات الفارغة استخدم `--repeat 12`؛ ينتج ملف الاختبار
الحالي 13 صفحة في LibreOffice دون صفحات فارغة بالكامل، ويمكن رفع القيمة حتى
50 لاختبارات الضغط الأكبر.
