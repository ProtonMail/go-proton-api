package proton

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHeaders_MarshalInOrder(t *testing.T) {
	jsonBytes := []byte(`{"zz":"v1","foo":["a","b"],"bar":"30"}`)

	var h Headers

	err := json.Unmarshal(jsonBytes, &h)
	require.NoError(t, err)

	expectedKeyOrder := []string{"zz", "foo", "bar"}

	require.Equal(t, expectedKeyOrder, h.Order)

	serializedJson, err := json.Marshal(h)
	require.NoError(t, err)
	require.Equal(t, jsonBytes, serializedJson)
}
