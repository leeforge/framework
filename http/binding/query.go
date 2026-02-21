package binding

import (
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	validatorV10 "github.com/go-playground/validator/v10"
)

// QueryUnmarshaler 自定义类型可以实现此接口来自定义 query 参数解析
type QueryUnmarshaler interface {
	UnmarshalQuery(string) error
}

// ArrayStrategy 数组解析策略
type ArrayStrategy int

const (
	// ArrayStrategyMultiple 多次传参：?tags=go&tags=rust
	ArrayStrategyMultiple ArrayStrategy = iota
	// ArrayStrategyComma 逗号分隔：?tags=go,rust
	ArrayStrategyComma
	// ArrayStrategyBoth 两种都支持，优先多次传参
	ArrayStrategyBoth
)

// QueryParser 查询参数解析器
type QueryParser struct {
	tagName       string
	defaultTag    string
	arrayStrategy ArrayStrategy
}

// NewQueryParser 创建新的查询参数解析器
func NewQueryParser() *QueryParser {
	return &QueryParser{
		tagName:       "query",
		defaultTag:    "default",
		arrayStrategy: ArrayStrategyBoth,
	}
}

// SetArrayStrategy 设置数组解析策略
func (qp *QueryParser) SetArrayStrategy(strategy ArrayStrategy) {
	qp.arrayStrategy = strategy
}

// Parse 解析查询参数到结构体
func (qp *QueryParser) Parse(values url.Values, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &BindError{
			Type:    "bind_error",
			Message: "v must be a non-nil pointer",
		}
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return &BindError{
			Type:    "bind_error",
			Message: "v must be a pointer to struct",
		}
	}

	return qp.parseStruct(values, rv, "")
}

// parseStruct 解析结构体
func (qp *QueryParser) parseStruct(values url.Values, rv reflect.Value, prefix string) error {
	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		// 跳过未导出的字段
		if !field.CanSet() {
			continue
		}

		// 获取字段的查询参数名
		queryName := qp.getQueryName(fieldType, prefix)
		if queryName == "-" {
			continue
		}

		// 处理嵌套结构体
		if field.Kind() == reflect.Struct && fieldType.Type.Name() != "" {
			// 检查是否实现了 QueryUnmarshaler 接口
			if field.Addr().Type().Implements(reflect.TypeOf((*QueryUnmarshaler)(nil)).Elem()) {
				if err := qp.setFieldValue(field, values, queryName, fieldType.Name); err != nil {
					return err
				}
				continue
			}

			// 递归处理嵌套结构体
			if err := qp.parseStruct(values, field, queryName+"."); err != nil {
				return err
			}
			continue
		}

		// 处理指针类型的嵌套结构体
		if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			// 检查是否实现了 QueryUnmarshaler 接口
			if field.Type().Implements(reflect.TypeOf((*QueryUnmarshaler)(nil)).Elem()) {
				if err := qp.setFieldValue(field, values, queryName, fieldType.Name); err != nil {
					return err
				}
				continue
			}

			// 如果有相关的查询参数，初始化指针
			if qp.hasNestedParams(values, queryName+".") {
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}
				if err := qp.parseStruct(values, field.Elem(), queryName+"."); err != nil {
					return err
				}
			}
			continue
		}

		// 设置字段值
		if err := qp.setFieldValue(field, values, queryName, fieldType.Name); err != nil {
			return err
		}
	}

	return nil
}

// hasNestedParams 检查是否有嵌套参数
func (qp *QueryParser) hasNestedParams(values url.Values, prefix string) bool {
	for key := range values {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// getQueryName 获取字段对应的查询参数名
func (qp *QueryParser) getQueryName(fieldType reflect.StructField, prefix string) string {
	// 优先使用 query 标签
	tagName := fieldType.Tag.Get(qp.tagName)
	if tagName != "" {
		name := strings.Split(tagName, ",")[0]
		if name == "-" {
			return "-"
		}
		return prefix + name
	}

	// 其次使用 json 标签
	tagName = fieldType.Tag.Get("json")
	if tagName != "" {
		name := strings.Split(tagName, ",")[0]
		if name == "-" {
			return "-"
		}
		return prefix + name
	}

	// 最后使用字段名的小写形式
	return prefix + strings.ToLower(fieldType.Name)
}

// setFieldValue 设置字段值
func (qp *QueryParser) setFieldValue(field reflect.Value, values url.Values, queryName string, fieldName string) error {
	// 获取查询参数值
	queryValues, exists := values[queryName]

	// 如果不存在，尝试应用默认值
	if !exists || len(queryValues) == 0 {
		return qp.applyDefaultValue(field, fieldName)
	}

	// 处理指针类型
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return qp.setFieldValue(field.Elem(), values, queryName, fieldName)
	}

	// 检查是否实现了 QueryUnmarshaler 接口
	if field.CanAddr() && field.Addr().Type().Implements(reflect.TypeOf((*QueryUnmarshaler)(nil)).Elem()) {
		unmarshaler := field.Addr().Interface().(QueryUnmarshaler)
		if err := unmarshaler.UnmarshalQuery(queryValues[0]); err != nil {
			return &BindError{
				Type:    "bind_error",
				Field:   fieldName,
				Message: "failed to unmarshal query: " + err.Error(),
			}
		}
		return nil
	}

	// 根据字段类型设置值
	return qp.setField(field, queryValues, fieldName)
}

// applyDefaultValue 应用默认值
func (qp *QueryParser) applyDefaultValue(field reflect.Value, fieldName string) error {
	// 通过反射获取字段的 StructField 信息
	// 注意：这里我们需要从父结构体获取，所以这个方法需要改进
	// 暂时通过零值检查来应用默认值
	if !field.IsZero() {
		return nil
	}

	// 由于无法直接获取 StructField，默认值将在 parseStruct 中处理
	return nil
}

// setField 根据字段类型设置值
func (qp *QueryParser) setField(field reflect.Value, values []string, fieldName string) error {
	if len(values) == 0 {
		return nil
	}

	kind := field.Kind()
	firstValue := values[0]

	switch kind {
	case reflect.String:
		field.SetString(firstValue)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(firstValue, 10, 64)
		if err != nil {
			return &BindError{
				Type:    "bind_error",
				Field:   fieldName,
				Message: "invalid integer value: " + err.Error(),
			}
		}
		field.SetInt(intVal)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(firstValue, 10, 64)
		if err != nil {
			return &BindError{
				Type:    "bind_error",
				Field:   fieldName,
				Message: "invalid unsigned integer value: " + err.Error(),
			}
		}
		field.SetUint(uintVal)

	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(firstValue, 64)
		if err != nil {
			return &BindError{
				Type:    "bind_error",
				Field:   fieldName,
				Message: "invalid float value: " + err.Error(),
			}
		}
		field.SetFloat(floatVal)

	case reflect.Bool:
		boolVal, err := strconv.ParseBool(firstValue)
		if err != nil {
			return &BindError{
				Type:    "bind_error",
				Field:   fieldName,
				Message: "invalid boolean value: " + err.Error(),
			}
		}
		field.SetBool(boolVal)

	case reflect.Slice:
		return qp.setSliceField(field, values, fieldName)

	default:
		return &BindError{
			Type:    "bind_error",
			Field:   fieldName,
			Message: "unsupported field type: " + kind.String(),
		}
	}

	return nil
}

// setSliceField 设置切片类型字段
func (qp *QueryParser) setSliceField(field reflect.Value, values []string, fieldName string) error {
	elemType := field.Type().Elem()

	// 处理数组解析策略
	var actualValues []string
	switch qp.arrayStrategy {
	case ArrayStrategyMultiple:
		actualValues = values
	case ArrayStrategyComma:
		// 使用逗号分隔
		if len(values) > 0 {
			actualValues = strings.Split(values[0], ",")
		}
	case ArrayStrategyBoth:
		// 如果只有一个值且包含逗号，则按逗号分隔
		if len(values) == 1 && strings.Contains(values[0], ",") {
			actualValues = strings.Split(values[0], ",")
		} else {
			actualValues = values
		}
	}

	// 创建切片
	slice := reflect.MakeSlice(field.Type(), len(actualValues), len(actualValues))

	// 填充切片元素
	for i, val := range actualValues {
		val = strings.TrimSpace(val)
		elem := slice.Index(i)

		// 根据元素类型设置值
		switch elemType.Kind() {
		case reflect.String:
			elem.SetString(val)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return &BindError{
					Type:    "bind_error",
					Field:   fieldName,
					Message: "invalid integer value in array: " + err.Error(),
				}
			}
			elem.SetInt(intVal)

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			uintVal, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return &BindError{
					Type:    "bind_error",
					Field:   fieldName,
					Message: "invalid unsigned integer value in array: " + err.Error(),
				}
			}
			elem.SetUint(uintVal)

		case reflect.Float32, reflect.Float64:
			floatVal, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return &BindError{
					Type:    "bind_error",
					Field:   fieldName,
					Message: "invalid float value in array: " + err.Error(),
				}
			}
			elem.SetFloat(floatVal)

		case reflect.Bool:
			boolVal, err := strconv.ParseBool(val)
			if err != nil {
				return &BindError{
					Type:    "bind_error",
					Field:   fieldName,
					Message: "invalid boolean value in array: " + err.Error(),
				}
			}
			elem.SetBool(boolVal)

		default:
			return &BindError{
				Type:    "bind_error",
				Field:   fieldName,
				Message: "unsupported slice element type: " + elemType.Kind().String(),
			}
		}
	}

	field.Set(slice)
	return nil
}

// parseQueryWithDefaults 解析查询参数并应用默认值
func (qp *QueryParser) parseQueryWithDefaults(values url.Values, rv reflect.Value, prefix string) error {
	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		if !field.CanSet() {
			continue
		}

		queryName := qp.getQueryName(fieldType, prefix)
		if queryName == "-" {
			continue
		}

		// 处理嵌套结构体
		if field.Kind() == reflect.Struct && fieldType.Type.Name() != "" {
			if field.Addr().Type().Implements(reflect.TypeOf((*QueryUnmarshaler)(nil)).Elem()) {
				if err := qp.setFieldValueWithDefault(field, values, queryName, fieldType); err != nil {
					return err
				}
				continue
			}
			if err := qp.parseQueryWithDefaults(values, field, queryName+"."); err != nil {
				return err
			}
			continue
		}

		// 处理指针类型的嵌套结构体
		if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			if field.Type().Implements(reflect.TypeOf((*QueryUnmarshaler)(nil)).Elem()) {
				if err := qp.setFieldValueWithDefault(field, values, queryName, fieldType); err != nil {
					return err
				}
				continue
			}
			if qp.hasNestedParams(values, queryName+".") || fieldType.Tag.Get(qp.defaultTag) != "" {
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}
				if err := qp.parseQueryWithDefaults(values, field.Elem(), queryName+"."); err != nil {
					return err
				}
			}
			continue
		}

		if err := qp.setFieldValueWithDefault(field, values, queryName, fieldType); err != nil {
			return err
		}
	}

	return nil
}

// setFieldValueWithDefault 设置字段值（包含默认值处理）
func (qp *QueryParser) setFieldValueWithDefault(field reflect.Value, values url.Values, queryName string, fieldType reflect.StructField) error {
	queryValues, exists := values[queryName]

	if exists && len(queryValues) > 0 {
		// 使用查询参数的值
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			return qp.setFieldValueWithDefault(field.Elem(), values, queryName, fieldType)
		}

		if field.CanAddr() && field.Addr().Type().Implements(reflect.TypeOf((*QueryUnmarshaler)(nil)).Elem()) {
			unmarshaler := field.Addr().Interface().(QueryUnmarshaler)
			if err := unmarshaler.UnmarshalQuery(queryValues[0]); err != nil {
				return &BindError{
					Type:    "bind_error",
					Field:   fieldType.Name,
					Message: "failed to unmarshal query: " + err.Error(),
				}
			}
			return nil
		}

		return qp.setField(field, queryValues, fieldType.Name)
	}

	// 应用默认值
	defaultValue := fieldType.Tag.Get(qp.defaultTag)
	if defaultValue != "" {
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			return qp.setFieldValueWithDefault(field.Elem(), url.Values{queryName: []string{defaultValue}}, queryName, fieldType)
		}

		return qp.setField(field, []string{defaultValue}, fieldType.Name)
	}

	return nil
}

// QueryWithParser 使用自定义解析器解析查询参数
func QueryWithParser(r *http.Request, v any, parser *QueryParser) error {
	queryValues := r.URL.Query()

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &BindError{
			Type:    "bind_error",
			Message: "v must be a non-nil pointer",
		}
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return &BindError{
			Type:    "bind_error",
			Message: "v must be a pointer to struct",
		}
	}

	if err := parser.parseQueryWithDefaults(queryValues, rv, ""); err != nil {
		return err
	}

	if err := validator.Struct(v); err != nil {
		if validationErrors, ok := err.(validatorV10.ValidationErrors); ok {
			var bindErrors ValidationErrors
			for _, ve := range validationErrors {
				bindErrors = append(bindErrors, BindError{
					Type:    "validation_error",
					Field:   ve.Field(),
					Message: getValidationMessage(ve),
				})
			}
			return bindErrors
		}
		return &BindError{
			Type:    "validation_error",
			Message: err.Error(),
		}
	}

	return nil
}
