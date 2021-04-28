package main

import (
	"errors"
	"fmt"
	"reflect"
)

func i2s(data interface{}, out interface{}) error {
	// todo
	if reflect.ValueOf(out).Kind() != reflect.Ptr {
		return errors.New("No pointer error")
	}
	structValue := reflect.ValueOf(out).Elem()
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Map {
		if structValue.Kind() == reflect.Slice {
			return errors.New("Struct error")
		}
		for _, key := range v.MapKeys() {
			strct := v.MapIndex(key)
			i := 0
			for i = 0; i < structValue.NumField(); i++ {
				fieldName := structValue.Type().Field(i).Name
				// fmt.Println("fieldName is", fieldName)
				if fieldName == key.String() {
					break
				}
			}
			structFieldValue := structValue.Field(i)
			if !structFieldValue.IsValid() {
				fmt.Println("error invalid key")
				return fmt.Errorf("No such field: %s in obj", strct.String())
			}

			if !structFieldValue.CanSet() {
				fmt.Println("error cant set")
				return fmt.Errorf("Cannot set %s field value", strct.String())
			}

			res := structFieldValue.Addr().Interface()
			err := i2s(strct.Interface(), res)
			if err != nil {
				return err
			}
			structFieldValue.Set(reflect.ValueOf(res).Elem())

		}
	} else if v.Kind() == reflect.Slice {
		if structValue.Kind() != v.Kind() {
			return errors.New("types mismatch error")
		}
		slice := reflect.MakeSlice(structValue.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			structFieldValue := slice.Index(i)
			res := structFieldValue.Addr().Interface()

			err := i2s(v.Index(i).Interface(), res)
			if err != nil {
				return err
			}
			structFieldValue.Set(reflect.ValueOf(res).Elem())
		}
		structValue.Set(slice)
	} else if v.Kind() == reflect.Float64 {
		if structValue.Kind() != v.Kind() && structValue.Kind() != reflect.Int {
			return errors.New("types mismatch error")
		}
		fieldType := structValue.Type()
		structValue.Set(v.Convert(fieldType))

	} else if v.Kind() == reflect.String {
		if structValue.Kind() != v.Kind() {
			return errors.New("types mismatch error")
		}
		fieldType := structValue.Type()
		structValue.Set(v.Convert(fieldType))

	} else if v.Kind() == reflect.Bool {
		if structValue.Kind() != v.Kind() {
			return errors.New("error")
		}
		fieldType := structValue.Type()
		structValue.Set(v.Convert(fieldType))

	} else {
		return errors.New("error")
	}
	out = structValue
	return nil
}
