package proton_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/ProtonMail/go-proton-api"
	"pgregory.net/rapid"
)

// canonicalJSON normalizes a JSON blob into a byte-deterministic form: object
// keys sorted, insignificant whitespace removed, numbers re-rendered. Two
// blobs are semantically equal iff their canonical forms are byte-equal.
func canonicalJSON(t *rapid.T, b []byte) string {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("canonicalize: unmarshal %q: %v", b, err)
	}
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("canonicalize: marshal: %v", err)
	}
	return string(out)
}

// genJSONValue draws an arbitrary JSON value (scalar, array, or object up to
// the given nesting depth) as raw JSON bytes. It is the section-value
// generator for the round-trip property: the fork must preserve any value an
// unmodeled section carries.
func genJSONValue(t *rapid.T, depth int) json.RawMessage {
	kinds := []string{"string", "int", "float", "bool", "null"}
	if depth > 0 {
		kinds = append(kinds, "array", "object")
	}

	switch rapid.SampledFrom(kinds).Draw(t, "kind") {
	case "string":
		b, _ := json.Marshal(rapid.String().Draw(t, "str"))
		return b
	case "int":
		b, _ := json.Marshal(rapid.Int64().Draw(t, "int"))
		return b
	case "float":
		b, _ := json.Marshal(rapid.Float64Range(-1e9, 1e9).Draw(t, "float"))
		return b
	case "bool":
		b, _ := json.Marshal(rapid.Bool().Draw(t, "bool"))
		return b
	case "array":
		n := rapid.IntRange(0, 4).Draw(t, "arrlen")
		arr := make([]json.RawMessage, n)
		for i := range arr {
			arr[i] = genJSONValue(t, depth-1)
		}
		b, _ := json.Marshal(arr)
		return b
	case "object":
		n := rapid.IntRange(0, 4).Draw(t, "objlen")
		m := make(map[string]json.RawMessage, n)
		for i := 0; i < n; i++ {
			k := rapid.StringMatching(`[A-Za-z0-9_]{1,10}`).Draw(t, "objkey")
			m[k] = genJSONValue(t, depth-1)
		}
		b, _ := json.Marshal(m)
		return b
	default: // "null"
		return json.RawMessage("null")
	}
}

// genCommon draws an arbitrary but schema-valid RevisionXAttrCommon (only the
// modeled fields, so the typed decode→encode of Common is faithful).
func genCommon(t *rapid.T) proton.RevisionXAttrCommon {
	var blockSizes []int64
	if rapid.Bool().Draw(t, "hasBlocks") {
		n := rapid.IntRange(0, 4).Draw(t, "nBlocks")
		blockSizes = make([]int64, n)
		for i := range blockSizes {
			blockSizes[i] = rapid.Int64().Draw(t, "blockSize")
		}
	}

	var digests map[string]string
	if rapid.Bool().Draw(t, "hasDigests") {
		digests = make(map[string]string)
		n := rapid.IntRange(0, 3).Draw(t, "nDigests")
		for i := 0; i < n; i++ {
			k := rapid.StringMatching(`[A-Za-z0-9]{1,8}`).Draw(t, "digestKey")
			digests[k] = rapid.String().Draw(t, "digestVal")
		}
	}

	return proton.RevisionXAttrCommon{
		ModificationTime: rapid.String().Draw(t, "mtime"),
		Size:             rapid.Int64().Draw(t, "size"),
		BlockSizes:       blockSizes,
		Digests:          digests,
	}
}

// TestProperty1_UnknownSectionsRoundTrip is the headline round-trip property:
// a blob with a valid Common plus an arbitrary set of unmodeled top-level
// sections survives UnmarshalJSON→MarshalJSON byte-equivalently under
// canonicalized JSON, with every unmodeled section preserved.
//
// Property 1: Unknown sections survive round-trip byte-equivalently.
// **Validates: Requirements 1.4**
func TestProperty1_UnknownSectionsRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		commonBytes, err := json.Marshal(genCommon(t))
		if err != nil {
			t.Fatalf("marshal Common: %v", err)
		}

		input := map[string]json.RawMessage{"Common": commonBytes}

		// Key generator biased toward the real-world sections other Proton
		// clients write, plus arbitrary keys to fuzz the space.
		keyGen := rapid.OneOf(
			rapid.SampledFrom([]string{"Media", "Camera", "Location", "POSIX"}),
			rapid.StringMatching(`[A-Za-z0-9_]{1,12}`),
		)

		nExtra := rapid.IntRange(0, 5).Draw(t, "nExtra")
		for i := 0; i < nExtra; i++ {
			k := keyGen.Draw(t, "sectionKey")
			if k == "Common" {
				continue // Common is the one typed section; never an Extra.
			}
			input[k] = genJSONValue(t, 3)
		}

		inputBytes, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("marshal input blob: %v", err)
		}

		var x proton.RevisionXAttr
		if err := json.Unmarshal(inputBytes, &x); err != nil {
			t.Fatalf("UnmarshalJSON: %v", err)
		}

		outBytes, err := json.Marshal(x)
		if err != nil {
			t.Fatalf("MarshalJSON: %v", err)
		}

		if got, want := canonicalJSON(t, outBytes), canonicalJSON(t, inputBytes); got != want {
			t.Fatalf("round-trip not byte-equivalent:\n input: %s\noutput: %s", want, got)
		}

		// Every unmodeled section must be preserved verbatim in Extra.
		for k, v := range input {
			if k == "Common" {
				continue
			}
			raw, ok := x.Extra[k]
			if !ok {
				t.Fatalf("section %q dropped from Extra", k)
			}
			if got, want := canonicalJSON(t, raw), canonicalJSON(t, v); got != want {
				t.Fatalf("section %q not preserved:\n want: %s\n got:  %s", k, want, got)
			}
		}
	})
}

// TestRevisionXAttr_StrayExtraCommon covers Requirement 1.6: a stray
// Extra["Common"] alongside the typed Common is discarded on encode — the
// typed Common wins — while sibling sections survive.
func TestRevisionXAttr_StrayExtraCommon(t *testing.T) {
	typed := proton.RevisionXAttrCommon{ModificationTime: "2021-09-16T07:40:54+0000", Size: 1234}
	typedBytes, err := json.Marshal(typed)
	if err != nil {
		t.Fatalf("marshal typed Common: %v", err)
	}

	x := proton.RevisionXAttr{
		Common: typed,
		Extra: map[string]json.RawMessage{
			"Common": json.RawMessage(`{"Size":9999,"Bogus":true}`),
			"Media":  json.RawMessage(`{"Width":4032,"Height":3024}`),
		},
	}

	out, err := json.Marshal(x)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if string(decoded["Common"]) != string(typedBytes) {
		t.Fatalf("stray Extra[\"Common\"] not discarded: got %s, want %s", decoded["Common"], typedBytes)
	}
	if _, ok := decoded["Media"]; !ok {
		t.Fatal("sibling Media section dropped")
	}
}

// TestRevisionXAttr_NoCommon covers Requirement 1.7: a blob with no Common
// section decodes to a zero Common with every section placed into Extra.
func TestRevisionXAttr_NoCommon(t *testing.T) {
	input := []byte(`{"Media":{"Width":100},"POSIX":{"Mode":493}}`)

	var x proton.RevisionXAttr
	if err := json.Unmarshal(input, &x); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if !reflect.DeepEqual(x.Common, proton.RevisionXAttrCommon{}) {
		t.Fatalf("expected zero Common, got %+v", x.Common)
	}
	for _, k := range []string{"Media", "POSIX"} {
		if _, ok := x.Extra[k]; !ok {
			t.Fatalf("section %q not placed into Extra", k)
		}
	}
	if _, ok := x.Extra["Common"]; ok {
		t.Fatal("Common must never appear in Extra")
	}
}

// TestRevisionXAttr_EmptyObject covers Requirement 1.8: an empty JSON object
// decodes to a zero Common and leaves Extra nil.
func TestRevisionXAttr_EmptyObject(t *testing.T) {
	var x proton.RevisionXAttr
	if err := json.Unmarshal([]byte(`{}`), &x); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if !reflect.DeepEqual(x.Common, proton.RevisionXAttrCommon{}) {
		t.Fatalf("expected zero Common, got %+v", x.Common)
	}
	if x.Extra != nil {
		t.Fatalf("expected nil Extra, got %v", x.Extra)
	}
}

// TestProperty1_MarshalDeterminism is the determinism facet of Property 1:
// repeated MarshalJSON of the same RevisionXAttr yields byte-identical output,
// and the top-level keys of that output are emitted in ascending lexicographic
// order. Deterministic, sorted output is what makes the round-trip byte
// comparison in TestProperty1_UnknownSectionsRoundTrip well-defined.
//
// Property 1 (determinism facet): top-level keys emitted in ascending
// lexicographic order.
// **Validates: Requirements 1.3, 1.4**
func TestProperty1_MarshalDeterminism(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		x := proton.RevisionXAttr{Common: genCommon(t)}

		// Draw an arbitrary set of unmodeled sections into Extra.
		keyGen := rapid.OneOf(
			rapid.SampledFrom([]string{"Media", "Camera", "Location", "POSIX"}),
			rapid.StringMatching(`[A-Za-z0-9_]{1,12}`),
		)
		nExtra := rapid.IntRange(0, 6).Draw(t, "nExtra")
		for i := 0; i < nExtra; i++ {
			k := keyGen.Draw(t, "sectionKey")
			if k == "Common" {
				continue // Common is the typed section, never an Extra.
			}
			if x.Extra == nil {
				x.Extra = make(map[string]json.RawMessage)
			}
			x.Extra[k] = genJSONValue(t, 3)
		}

		first, err := json.Marshal(x)
		if err != nil {
			t.Fatalf("MarshalJSON: %v", err)
		}

		// Repeated marshaling of the same section set is byte-identical.
		for i := 0; i < 4; i++ {
			again, err := json.Marshal(x)
			if err != nil {
				t.Fatalf("MarshalJSON (repeat %d): %v", i, err)
			}
			if !bytes.Equal(first, again) {
				t.Fatalf("MarshalJSON not deterministic:\n first: %s\nrepeat: %s", first, again)
			}
		}

		// Top-level keys must appear in ascending lexicographic order.
		keys := topLevelKeyOrder(t, first)
		if !sort.StringsAreSorted(keys) {
			t.Fatalf("top-level keys not in ascending order: %v (blob: %s)", keys, first)
		}
	})
}

// topLevelKeyOrder returns the top-level object keys of a JSON blob in the
// order they appear in the byte stream, using a streaming decoder (the
// order-preserving read encoding/json's map marshaling does not otherwise
// expose).
func topLevelKeyOrder(t *rapid.T, b []byte) []string {
	dec := json.NewDecoder(bytes.NewReader(b))

	tok, err := dec.Token()
	if err != nil {
		t.Fatalf("decode open: %v", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		t.Fatalf("expected top-level object, got %v", tok)
	}

	var keys []string
	depth := 0
	for dec.More() || depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			t.Fatalf("decode token: %v", err)
		}
		switch v := tok.(type) {
		case json.Delim:
			switch v {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		case string:
			if depth == 0 {
				keys = append(keys, v)
				// Consume this key's value so the next string we see at
				// depth 0 is the next key, not a nested string.
				if err := skipValue(dec); err != nil {
					t.Fatalf("skip value: %v", err)
				}
			}
		}
	}
	return keys
}

// skipValue consumes exactly one JSON value from dec, descending through
// nested objects and arrays so the caller resumes at the following key.
func skipValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	d, ok := tok.(json.Delim)
	if !ok {
		return nil // scalar value, already consumed
	}
	depth := 1
	if d != '{' && d != '[' {
		return nil
	}
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		}
	}
	return nil
}

// TestRevisionXAttr_CommonMediaPosixLossless covers the lossless unit case: a
// blob carrying Common, Media, and POSIX sections decodes then encodes back to
// the same canonical JSON, with the typed Common preserved and both unmodeled
// sections (Media, POSIX) round-tripped verbatim through Extra.
func TestRevisionXAttr_CommonMediaPosixLossless(t *testing.T) {
	input := []byte(`{` +
		`"Common":{"ModificationTime":"2021-09-16T07:40:54+0000","Size":13283,"BlockSizes":[1234,5678],"Digests":{"SHA1":"abc123"}},` +
		`"Media":{"Width":4032,"Height":3024,"Duration":0},` +
		`"POSIX":{"Mode":493}` +
		`}`)

	var x proton.RevisionXAttr
	if err := json.Unmarshal(input, &x); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	// Typed Common decoded faithfully.
	wantCommon := proton.RevisionXAttrCommon{
		ModificationTime: "2021-09-16T07:40:54+0000",
		Size:             13283,
		BlockSizes:       []int64{1234, 5678},
		Digests:          map[string]string{"SHA1": "abc123"},
	}
	if !reflect.DeepEqual(x.Common, wantCommon) {
		t.Fatalf("Common decoded wrong:\n want: %+v\n got:  %+v", wantCommon, x.Common)
	}

	// Media and POSIX preserved verbatim in Extra; Common never in Extra.
	for _, k := range []string{"Media", "POSIX"} {
		if _, ok := x.Extra[k]; !ok {
			t.Fatalf("section %q not placed into Extra", k)
		}
	}
	if _, ok := x.Extra["Common"]; ok {
		t.Fatal("Common must never appear in Extra")
	}

	// Encode losslessly: output equals input under canonicalized JSON.
	out, err := json.Marshal(x)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if got, want := canonicalBytes(t, out), canonicalBytes(t, input); got != want {
		t.Fatalf("encode not lossless:\n input: %s\noutput: %s", want, got)
	}
}

// canonicalBytes is the non-rapid twin of canonicalJSON: it normalizes a JSON
// blob (keys sorted, whitespace removed) for use in plain unit tests.
func canonicalBytes(t *testing.T, b []byte) string {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("canonicalize: unmarshal %q: %v", b, err)
	}
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("canonicalize: marshal: %v", err)
	}
	return string(out)
}
