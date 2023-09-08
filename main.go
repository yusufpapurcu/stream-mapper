package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"
)

type MappingFunc func(input []any) (any, error)
type ValidationFunc func(input []any) (bool, error)

type Definition struct {
	Type    string `json:"type"`
	Index   int    `json:"index"`
	Mapping struct {
		Func       string   `json:"func"`
		FuncParams []string `json:"funcParams"`
	} `json:"mapping"`
	Validation struct {
		Func       string   `json:"func"`
		FuncParams []string `json:"funcParams"`
	} `json:"validation"`
}

type StreamMapper struct {
	outputSchema    []byte
	output          map[string]Definition
	mappingFuncs    map[string]MappingFunc
	validationFuncs map[string]ValidationFunc
}

func NewStreamMapper(outputSchema []byte) (*StreamMapper, error) {
	var s StreamMapper

	s.output = make(map[string]Definition)
	if err := json.NewDecoder(bytes.NewBuffer(outputSchema)).Decode(&s.output); err != nil {
		return nil, err
	}

	s.mappingFuncs = make(map[string]MappingFunc)
	s.validationFuncs = make(map[string]ValidationFunc)
	return &s, nil
}

func (s *StreamMapper) Process(input map[string]map[string]any) map[string]any {
	var outputResult = make(map[string]any)

	// instructions == map[string]any{}
	for outputKey, instructions := range s.output {
		mappingFunc := instructions.Mapping.Func
		var mappingInput []any
		for _, key := range instructions.Mapping.FuncParams {
			fields := strings.Split(key, ".")
			if len(fields) < 2 {
				panic("sg")
			}

			var source map[string]any
			if fields[0] == "self" {
				source = outputResult
			} else {
				source = input[fields[0]]
			}

			for i := 2; i < len(fields)-1; i++ {
				source = source[fields[i]].(map[string]any)
			}

			mappingInput = append(mappingInput, source[fields[len(fields)-1]])
		}

		fieldVal, err := s.mappingFuncs[mappingFunc](mappingInput)
		if err != nil {
			panic(fmt.Sprintf("mapping failed for %s: %s", mappingFunc, err.Error()))
		}

		outputResult[outputKey] = fieldVal

		validationFunc := instructions.Validation.Func

		var validationInput []any
		for _, key := range instructions.Validation.FuncParams {
			fields := strings.Split(key, ".")
			if len(fields) != 2 {
				panic("sg")
			}

			var source map[string]any
			if fields[0] == "self" {
				source = outputResult
			} else {
				source = input[fields[0]]
			}

			for i := 2; i < len(fields)-1; i++ {
				source = source[fields[i]].(map[string]any)
			}

			validationInput = append(validationInput, source[fields[len(fields)-1]])
		}

		if isValid, err := s.validationFuncs[validationFunc](validationInput); err != nil {
			panic(fmt.Sprintf("validation failed for %s: %s", validationFunc, err.Error()))
		} else if !isValid {
			panic("notValid")
		}
	}

	return outputResult
}

func (s *StreamMapper) AddMappingFunc(name string, function MappingFunc) error {
	if _, ok := s.mappingFuncs[name]; ok {
		return fmt.Errorf("duplicate func names")
	}

	s.mappingFuncs[name] = function
	return nil
}

func (s *StreamMapper) AddValidateFunc(name string, function ValidationFunc) error {
	if _, ok := s.validationFuncs[name]; ok {
		return fmt.Errorf("duplicate func names")
	}

	s.validationFuncs[name] = function
	return nil
}

func Decode(data io.Reader, v interface{}) error {
	decoder := json.NewDecoder(data)
	decoder.UseNumber()
	return decoder.Decode(v)
}

func main() {
	f, err := os.Open("output.json")
	if err != nil {
		panic("file")
	}

	output, err := io.ReadAll(f)
	if err != nil {
		panic("file read")
	}
	err = f.Close()
	if err != nil {
		panic("file close")
	}

	streamMapper := GetStream(output)
	f, err = os.Open("input_hostEnum.json")
	if err != nil {
		panic("file")
	}

	hostEnum := make(map[string]any)
	err = Decode(f, &hostEnum)
	if err != nil {
		panic("json")
	}
	err = f.Close()
	if err != nil {
		panic("file close")
	}

	f, err = os.Open("input_state.json")
	if err != nil {
		panic("file")
	}

	state := make(map[string]any)
	err = Decode(f, &state)
	if err != nil {
		panic("json")
	}
	err = f.Close()
	if err != nil {
		panic("file close")
	}

	res := streamMapper.Process(map[string]map[string]any{
		"host_enum": hostEnum,
		"state":     state,
	})

	resJson, err := json.Marshal(res)
	if err != nil {
		panic("file close")
	}
	fmt.Println(string(resJson))
}

func GetStream(output []byte) *StreamMapper {
	streamMapper, err := NewStreamMapper(output)
	if err != nil {
		panic("new bum")
	}

	err = streamMapper.AddMappingFunc("StringToString", func(input []any) (any, error) {
		if len(input) != 1 {
			panic("StringToString")
		}

		switch input[0].(type) {
		case string:
			return input[0], nil
		default:
			return nil, fmt.Errorf("type invalid")
		}
	})
	err = streamMapper.AddMappingFunc("IntToString", func(input []any) (any, error) {
		if len(input) != 1 {
			panic("IntToString")
		}
		switch input[0].(type) {
		case json.Number:
			return input[0].(json.Number).String(), nil
		case int64, int:
			return fmt.Sprintf("%d", input[0]), nil
		default:
			return nil, fmt.Errorf("invalid type")
		}
	})
	err = streamMapper.AddValidateFunc("ValidateStringZero", func(input []any) (bool, error) {
		if len(input) != 1 {
			panic("ValidateStringZero")
		}
		switch input[0].(type) {
		case string:
			return input[0].(string) != "0", nil
		default:
			k := reflect.TypeOf(input[0])
			return false, fmt.Errorf("invalid type %s", k)
		}
	})
	err = streamMapper.AddValidateFunc("ValidateHostEnum", func(input []any) (bool, error) {
		if len(input) != 1 {
			panic("ValidateHostEnum")
		}
		switch input[0].(type) {
		case string:
			return input[0].(string) == "Camino4", nil
		default:
			return false, fmt.Errorf("invalid type")
		}
	})
	err = streamMapper.AddValidateFunc("ValidateStringEmpty", func(input []any) (bool, error) {
		if len(input) != 1 {
			panic("ValidateStringEmpty")
		}
		switch input[0].(type) {
		case string:
			return input[0].(string) != "", nil
		default:
			return false, fmt.Errorf("invalid type")
		}
	})
	err = streamMapper.AddValidateFunc("TimeRFC3339", func(input []any) (bool, error) {
		if len(input) != 1 {
			panic("TimeRFC3339")
		}
		switch input[0].(type) {
		case string:
			_, err := time.Parse(time.RFC3339, input[0].(string))
			if err != nil {
				return false, fmt.Errorf("error on banking date(%s): %w", input[0].(string), err)
			}

			return true, nil
		default:
			return false, fmt.Errorf("invalid type")
		}
	})

	return streamMapper
}
