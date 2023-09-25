package proton

import (
	"encoding/json"
	"errors"
	"gitlab.com/c0b/go-ordered-json"
)

var ErrBadHeader = errors.New("bad header")

type Headers struct {
	Values map[string][]string
	Order  []string
}

func (h *Headers) UnmarshalJSON(b []byte) error {
	type rawHeaders map[string]any

	raw := make(rawHeaders)

	// Need to use a different type to deserialize, because there still is no official way for json to decode an object
	// with the fields in order https://github.com/golang/go/issues/27179.
	orderedMap := ordered.NewOrderedMap()
	if err := orderedMap.UnmarshalJSON(b); err != nil {
		return err
	}

	header := Headers{
		Values: make(map[string][]string, len(raw)),
		Order:  make([]string, 0, len(raw)),
	}

	iter := orderedMap.EntriesIter()

	for {
		entry, ok := iter()
		if !ok {
			break
		}

		switch val := entry.Value.(type) {
		case string:
			header.Values[entry.Key] = []string{val}

		case []any:
			for _, val := range val {
				switch val := val.(type) {
				case string:
					header.Values[entry.Key] = append(header.Values[entry.Key], val)

				default:
					return ErrBadHeader
				}
			}

		default:
			return ErrBadHeader
		}

		header.Order = append(header.Order, entry.Key)
	}

	*h = header

	return nil
}

func (h Headers) MarshalJSON() ([]byte, error) {
	// Manually Serialize to preserve oder
	if len(h.Values) == 0 {
		return []byte{'{', '}'}, nil
	}

	out := make([]byte, 0, 64)

	out = append(out, '{')

	for _, k := range h.Order {
		v := h.Values[k]

		if len(v) == 0 {
			continue
		}

		key, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}

		var val []byte
		if len(v) == 1 {
			val, err = json.Marshal(v[0])
		} else {
			val, err = json.Marshal(v)
		}
		if err != nil {
			return nil, err
		}

		out = append(out, key...)
		out = append(out, ':')
		out = append(out, val...)
		out = append(out, ',')
	}

	out[len(out)-1] = '}'

	return out, nil
}
