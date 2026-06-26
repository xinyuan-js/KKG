package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type StringList []string

func (s StringList) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (s *StringList) Scan(value interface{}) error {
	if value == nil {
		*s = StringList{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*s = StringList{}
			return nil
		}
		return json.Unmarshal(v, s)
	case string:
		if v == "" {
			*s = StringList{}
			return nil
		}
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("unsupported StringList scan type: %T", value)
	}
}
