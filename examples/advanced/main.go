// Command advanced renders a deeply nested DOCX template with registered Go
// functions called from inside nested loops.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/nawafinity/go-namat"
)

var exit = os.Exit
var encodePNG = func(writer io.Writer, source image.Image) error { return png.Encode(writer, source) }

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "advanced example:", err)
		exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("advanced", flag.ContinueOnError)
	templatePath := flags.String("template", "examples/advanced/template.docx", "DOCX template path")
	dataPath := flags.String("data", "examples/advanced/data.json", "JSON data path")
	outputPath := flags.String("out", "advanced-report.docx", "rendered DOCX path")
	repeat := flags.Int("repeat", 1, "repeat the department dataset 1-50 times for pagination stress tests")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if *repeat < 1 || *repeat > 50 {
		return errors.New("repeat must be between 1 and 50")
	}

	template, err := os.ReadFile(*templatePath)
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}
	encodedData, err := os.ReadFile(*dataPath)
	if err != nil {
		return fmt.Errorf("read data: %w", err)
	}
	data, err := namat.DecodeJSON(bytes.NewReader(encodedData))
	if err != nil {
		return fmt.Errorf("decode data: %w", err)
	}
	data, err = repeatReportData(data, *repeat)
	if err != nil {
		return fmt.Errorf("repeat data: %w", err)
	}

	report, err := namat.CreateReport(context.Background(), template, data, namat.Options{
		Functions: advancedFunctions(),
	})
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}
	if err := os.WriteFile(*outputPath, report, 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	fmt.Println(*outputPath)
	return nil
}

func repeatReportData(data any, repeat int) (any, error) {
	if repeat == 1 {
		return data, nil
	}
	root, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("report data must be an object, got %T", data)
	}
	departments, ok := root["departments"].([]any)
	if !ok {
		return nil, fmt.Errorf("departments must be an array, got %T", root["departments"])
	}

	expanded := make([]any, 0, len(departments)*repeat)
	for batch := 1; batch <= repeat; batch++ {
		for _, department := range departments {
			clone := cloneJSONValue(department)
			object, ok := clone.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("department must be an object, got %T", clone)
			}
			suffix := fmt.Sprintf(" #%02d", batch)
			object["name"] = fmt.Sprint(object["name"]) + suffix
			projects, ok := object["projects"].([]any)
			if !ok {
				return nil, fmt.Errorf("department projects must be an array, got %T", object["projects"])
			}
			for projectIndex, project := range projects {
				projectObject, ok := project.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("project %d must be an object, got %T", projectIndex, project)
				}
				projectObject["name"] = fmt.Sprint(projectObject["name"]) + suffix
				projectObject["url"] = fmt.Sprintf("%s?batch=%d", projectObject["url"], batch)
				tasks, ok := projectObject["tasks"].([]any)
				if !ok {
					return nil, fmt.Errorf("project %d tasks must be an array, got %T", projectIndex, projectObject["tasks"])
				}
				for _, task := range tasks {
					taskObject, ok := task.(map[string]any)
					if !ok {
						return nil, fmt.Errorf("project %d task must be an object, got %T", projectIndex, task)
					}
					taskObject["name"] = fmt.Sprint(taskObject["name"]) + suffix
				}
			}
			expanded = append(expanded, object)
		}
	}
	root["departments"] = expanded
	return root, nil
}

func cloneJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		clone := make(map[string]any, len(typed))
		for key, item := range typed {
			clone[key] = cloneJSONValue(item)
		}
		return clone
	case []any:
		clone := make([]any, len(typed))
		for index, item := range typed {
			clone[index] = cloneJSONValue(item)
		}
		return clone
	default:
		return value
	}
}

func advancedFunctions() map[string]namat.FunctionSpec {
	return map[string]namat.FunctionSpec{
		"upper": pureFunction([]namat.ValueType{namat.TypeString}, namat.TypeString, func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, errors.New("upper expects one argument")
			}
			return strings.ToUpper(fmt.Sprint(args[0])), nil
		}),
		"count": pureFunction([]namat.ValueType{namat.TypeAny}, namat.TypeInt, func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, errors.New("count expects one argument")
			}
			value := reflect.ValueOf(args[0])
			if !value.IsValid() || (value.Kind() != reflect.Array && value.Kind() != reflect.Slice && value.Kind() != reflect.Map && value.Kind() != reflect.String) {
				return nil, fmt.Errorf("count does not support %T", args[0])
			}
			return value.Len(), nil
		}),
		"money": pureFunction([]namat.ValueType{namat.TypeAny, namat.TypeString}, namat.TypeString, func(args ...any) (any, error) {
			if len(args) != 2 {
				return nil, errors.New("money expects amount and currency")
			}
			amount, ok := decimalNumber(args[0])
			if !ok {
				return nil, fmt.Errorf("money amount must be numeric, got %T", args[0])
			}
			return fmt.Sprintf("%s %s", amount.Rat().FloatString(2), fmt.Sprint(args[1])), nil
		}),
		"sumBudgets": pureFunction([]namat.ValueType{namat.TypeList}, namat.TypeDecimal, func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, errors.New("sumBudgets expects projects")
			}
			return sumField(args[0], "budget")
		}),
		"sumHours": pureFunction([]namat.ValueType{namat.TypeList}, namat.TypeDecimal, func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, errors.New("sumHours expects tasks")
			}
			return sumField(args[0], "hours")
		}),
		"statusLabel": pureFunction([]namat.ValueType{namat.TypeString}, namat.TypeString, func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, errors.New("statusLabel expects one argument")
			}
			labels := map[string]string{"green": "مستقر", "amber": "يحتاج متابعة", "red": "متعثر"}
			value := fmt.Sprint(args[0])
			if label, ok := labels[value]; ok {
				return label, nil
			}
			return value, nil
		}),
		"taskBadge": pureFunction([]namat.ValueType{namat.TypeString}, namat.TypeString, func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, errors.New("taskBadge expects one argument")
			}
			badges := map[string]string{"done": "✓ مكتمل", "doing": "◐ جارٍ", "blocked": "! متوقف"}
			value := fmt.Sprint(args[0])
			if badge, ok := badges[value]; ok {
				return badge, nil
			}
			return value, nil
		}),
		"hours": pureFunction([]namat.ValueType{namat.TypeAny}, namat.TypeString, func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, errors.New("hours expects one argument")
			}
			value, ok := decimalNumber(args[0])
			if !ok {
				return nil, fmt.Errorf("hours value must be numeric, got %T", args[0])
			}
			return value.String() + " ساعة", nil
		}),
		"projectLink": pureFunction([]namat.ValueType{namat.TypeObject}, namat.TypeRich, func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, errors.New("projectLink expects one project")
			}
			project, ok := args[0].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("projectLink expects an object, got %T", args[0])
			}
			return namat.Link{URL: fmt.Sprint(project["url"]), Label: fmt.Sprint(project["name"]), Tooltip: "فتح صفحة المشروع"}, nil
		}),
		"sparkline": pureFunction([]namat.ValueType{namat.TypeList, namat.TypeString}, namat.TypeRich, sparkline),
	}
}

func pureFunction(params []namat.ValueType, returns namat.ValueType, call func(...any) (any, error)) namat.FunctionSpec {
	return namat.FunctionSpec{
		Params:  params,
		Returns: returns,
		Pure:    true,
		Call: func(_ context.Context, args ...any) (any, error) {
			return call(args...)
		},
	}
}

func sumField(collection any, field string) (namat.Decimal, error) {
	items, ok := collection.([]any)
	if !ok {
		return namat.Decimal{}, fmt.Errorf("sum %s expects an array, got %T", field, collection)
	}
	total, _ := namat.ParseDecimal("0")
	for index, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return namat.Decimal{}, fmt.Errorf("sum %s item %d is %T", field, index, item)
		}
		value, ok := decimalNumber(object[field])
		if !ok {
			return namat.Decimal{}, fmt.Errorf("sum %s item %d is not numeric", field, index)
		}
		total = total.Add(value)
	}
	return total, nil
}

func sparkline(args ...any) (any, error) {
	if len(args) != 2 {
		return nil, errors.New("sparkline expects values and label")
	}
	values, ok := args[0].([]any)
	if !ok || len(values) == 0 {
		return nil, errors.New("sparkline values must be a non-empty array")
	}
	numbers := make([]float64, len(values))
	for index, value := range values {
		numbers[index], ok = numeric(value)
		if !ok {
			return nil, fmt.Errorf("sparkline value %d is not numeric", index)
		}
	}

	const width, height = 360, 90
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	background := color.RGBA{R: 245, G: 248, B: 252, A: 255}
	bar := color.RGBA{R: 31, G: 111, B: 139, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			canvas.SetRGBA(x, y, background)
		}
	}
	barWidth := width / len(numbers)
	for index, value := range numbers {
		if value < 0 {
			value = 0
		}
		if value > 100 {
			value = 100
		}
		barHeight := int(value * float64(height-10) / 100)
		for y := height - 5 - barHeight; y < height-5; y++ {
			for x := index*barWidth + 3; x < (index+1)*barWidth-3; x++ {
				canvas.SetRGBA(x, y, bar)
			}
		}
	}
	var encoded bytes.Buffer
	if err := encodePNG(&encoded, canvas); err != nil {
		return nil, err
	}
	return namat.Image{Data: encoded.Bytes(), Extension: "png", Width: 5.4, Height: 1.35, Alt: "Progress chart for " + fmt.Sprint(args[1])}, nil
}

func numeric(value any) (float64, bool) {
	switch number := value.(type) {
	case namat.Decimal:
		result, _ := number.Rat().Float64()
		return result, true
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case int32:
		return float64(number), true
	case uint:
		return float64(number), true
	case uint64:
		return float64(number), true
	case uint32:
		return float64(number), true
	default:
		return 0, false
	}
}

func decimalNumber(value any) (namat.Decimal, bool) {
	switch number := value.(type) {
	case namat.Decimal:
		return number, true
	case int:
		result, err := namat.ParseDecimal(strconv.Itoa(number))
		return result, err == nil
	case int64:
		result, err := namat.ParseDecimal(strconv.FormatInt(number, 10))
		return result, err == nil
	case int32:
		result, err := namat.ParseDecimal(strconv.FormatInt(int64(number), 10))
		return result, err == nil
	case uint:
		result, err := namat.ParseDecimal(strconv.FormatUint(uint64(number), 10))
		return result, err == nil
	case uint64:
		result, err := namat.ParseDecimal(strconv.FormatUint(number, 10))
		return result, err == nil
	case uint32:
		result, err := namat.ParseDecimal(strconv.FormatUint(uint64(number), 10))
		return result, err == nil
	case float64:
		result, err := namat.ParseDecimal(strconv.FormatFloat(number, 'g', -1, 64))
		return result, err == nil
	case float32:
		result, err := namat.ParseDecimal(strconv.FormatFloat(float64(number), 'g', -1, 32))
		return result, err == nil
	default:
		return namat.Decimal{}, false
	}
}
