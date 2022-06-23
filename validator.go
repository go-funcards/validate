package validate

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
	"sync"
)

var Default = new(Validator)

type Validator struct {
	once     sync.Once
	validate *validator.Validate
}

type (
	Rules              map[string]string
	TypeRules          map[string]Rules
	SliceValidateError []error
)

func (err SliceValidateError) Error() string {
	var errMsgs []string
	for i, e := range err {
		if e == nil {
			continue
		}
		errMsgs = append(errMsgs, fmt.Sprintf("[%d]: %s", i, e.Error()))
	}
	return strings.Join(errMsgs, "\n")
}

func (v *Validator) RegisterStructRules(typeRules TypeRules, types ...any) {
	v.lazyinit()
	for _, t := range types {
		if rules, ok := typeRules[reflect.TypeOf(t).String()]; ok {
			v.validate.RegisterStructValidationMapRules(rules, t)
		}
	}
}

func (v *Validator) ValidateExcept(obj any, fields ...string) error {
	if obj == nil {
		return nil
	}

	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Ptr:
		return v.ValidateStruct(value.Elem().Interface())
	case reflect.Struct:
		return v.validateStruct(obj, fields...)
	case reflect.Slice, reflect.Array:
		count := value.Len()
		validateRet := make(SliceValidateError, 0)
		for i := 0; i < count; i++ {
			if err := v.ValidateStruct(value.Index(i).Interface()); err != nil {
				validateRet = append(validateRet, err)
			}
		}
		if len(validateRet) == 0 {
			return nil
		}
		return validateRet
	default:
		return nil
	}
}

// ValidateStruct receives any kind of type, but only performed struct or pointer to struct type.
func (v *Validator) ValidateStruct(obj any) error {
	return v.ValidateExcept(obj)
}

// validateStruct receives struct type
func (v *Validator) validateStruct(obj any, fields ...string) error {
	v.lazyinit()
	if len(fields) == 0 {
		return v.validate.Struct(obj)
	}
	return v.validate.StructExcept(obj, fields...)
}

// Engine returns the underlying validator engine which powers the default
func (v *Validator) Engine() any {
	v.lazyinit()
	return v.validate
}

func (v *Validator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New()
	})
}
