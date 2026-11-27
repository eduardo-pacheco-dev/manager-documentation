package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// NullableStringList is a JSON encoded list that keeps track of SQL NULL,
// which is meaningful for module preferences (NULL means "all enabled").
type NullableStringList struct {
	Valid  bool
	Values []string
}

func (n *NullableStringList) Scan(value any) error {
	if value == nil {
		n.Valid = false
		n.Values = nil
		return nil
	}
	var raw []byte
	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return fmt.Errorf("NullableStringList: unsupported type %T", value)
	}
	n.Valid = true
	if len(raw) == 0 {
		n.Values = []string{}
		return nil
	}
	return json.Unmarshal(raw, &n.Values)
}

func (n NullableStringList) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	if n.Values == nil {
		n.Values = []string{}
	}
	raw, err := json.Marshal(n.Values)
	return string(raw), err
}

func (n NullableStringList) OrEmpty() []string {
	if !n.Valid || n.Values == nil {
		return []string{}
	}
	return n.Values
}

func NewStringList(values []string) NullableStringList {
	if values == nil {
		values = []string{}
	}
	return NullableStringList{Valid: true, Values: values}
}

// NullableStringMap is the JSON encoded equivalent of a map preference.
type NullableStringMap struct {
	Valid  bool
	Values map[string]string
}

func (n *NullableStringMap) Scan(value any) error {
	if value == nil {
		n.Valid = false
		n.Values = nil
		return nil
	}
	var raw []byte
	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return fmt.Errorf("NullableStringMap: unsupported type %T", value)
	}
	n.Valid = true
	if len(raw) == 0 {
		n.Values = map[string]string{}
		return nil
	}
	return json.Unmarshal(raw, &n.Values)
}

func (n NullableStringMap) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	if n.Values == nil {
		n.Values = map[string]string{}
	}
	raw, err := json.Marshal(n.Values)
	return string(raw), err
}

func (n NullableStringMap) OrEmpty() map[string]string {
	if !n.Valid || n.Values == nil {
		return map[string]string{}
	}
	return n.Values
}

func NewStringMap(values map[string]string) NullableStringMap {
	if values == nil {
		values = map[string]string{}
	}
	return NullableStringMap{Valid: true, Values: values}
}

// NullableIntMap stores widget column spans (widget id -> 1..3).
type NullableIntMap struct {
	Valid  bool
	Values map[string]int
}

func (n *NullableIntMap) Scan(value any) error {
	if value == nil {
		n.Valid = false
		n.Values = nil
		return nil
	}
	var raw []byte
	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return fmt.Errorf("NullableIntMap: unsupported type %T", value)
	}
	n.Valid = true
	if len(raw) == 0 {
		n.Values = map[string]int{}
		return nil
	}
	return json.Unmarshal(raw, &n.Values)
}

func (n NullableIntMap) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	if n.Values == nil {
		n.Values = map[string]int{}
	}
	raw, err := json.Marshal(n.Values)
	return string(raw), err
}

func (n NullableIntMap) OrEmpty() map[string]int {
	if !n.Valid || n.Values == nil {
		return map[string]int{}
	}
	return n.Values
}

func NewIntMap(values map[string]int) NullableIntMap {
	if values == nil {
		values = map[string]int{}
	}
	return NullableIntMap{Valid: true, Values: values}
}
