package qframe_test

import (
	"bytes"
	"fmt"
	"github.com/tobgu/qframe"
	"github.com/tobgu/qframe/aggregation"
	"github.com/tobgu/qframe/filter"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func assertEquals(t *testing.T, expected, actual qframe.QFrame) {
	t.Helper()
	equal, reason := expected.Equals(actual)
	if !equal {
		t.Errorf("QFrames not equal, %s.\nexpected=\n%s\nactual=\n%s", reason, expected, actual)
	}
}

func assertNotErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}

func assertErr(t *testing.T, err error, expectedErr string) {
	t.Helper()
	if err == nil {
		t.Errorf("Expected error, was nil")
		return
	}

	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error to contain: %s, was: %s", expectedErr, err.Error())
	}
}

func assertTrue(t *testing.T, b bool) {
	t.Helper()
	if !b {
		t.Error("Expected true")
	}
}

func TestQFrame_FilterAgainstConstant(t *testing.T) {
	table := []struct {
		name     string
		filters  []filter.Filter
		input    map[string]interface{}
		expected qframe.QFrame
	}{
		{
			"built in greater than",
			[]filter.Filter{{Column: "COL1", Comparator: ">", Arg: 3}},
			map[string]interface{}{"COL1": []int{1, 2, 3, 4, 5}},
			qframe.New(map[string]interface{}{"COL1": []int{4, 5}})},
		{
			"built in 'in' with int",
			[]filter.Filter{{Column: "COL1", Comparator: "in", Arg: []int{3, 5}}},
			map[string]interface{}{"COL1": []int{1, 2, 3, 4, 5}},
			qframe.New(map[string]interface{}{"COL1": []int{3, 5}})},
		{
			"built in 'in' with float (truncated to int)",
			[]filter.Filter{{Column: "COL1", Comparator: "in", Arg: []float64{3.4, 5.1}}},
			map[string]interface{}{"COL1": []int{1, 2, 3, 4, 5}},
			qframe.New(map[string]interface{}{"COL1": []int{3, 5}})},
		{
			"combined with OR",
			[]filter.Filter{
				{Column: "COL1", Comparator: ">", Arg: 4},
				{Column: "COL1", Comparator: "<", Arg: 2}},
			map[string]interface{}{"COL1": []int{1, 2, 3, 4, 5}},
			qframe.New(map[string]interface{}{"COL1": []int{1, 5}})},
		{
			"inverse",
			[]filter.Filter{{Column: "COL1", Comparator: ">", Arg: 4, Inverse: true}},
			map[string]interface{}{"COL1": []int{1, 2, 3, 4, 5}},
			qframe.New(map[string]interface{}{"COL1": []int{1, 2, 3, 4}})},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("Filter %d", i), func(t *testing.T) {
			input := qframe.New(tc.input)
			output := input.Filter(tc.filters...)
			assertEquals(t, tc.expected, output)
		})
	}
}

func TestQFrame_FilterAgainstColumn(t *testing.T) {
	table := []struct {
		name       string
		comparator interface{}
		input      map[string]interface{}
		expected   map[string]interface{}
		configs    []qframe.ConfigFunc
	}{
		{
			name:       "built in int compare",
			comparator: ">",
			input:      map[string]interface{}{"COL1": []int{1, 2, 3}, "COL2": []int{10, 1, 10}},
			expected:   map[string]interface{}{"COL1": []int{1, 3}, "COL2": []int{10, 10}}},
		{
			name:       "custom int compare",
			comparator: func(a, b int) bool { return a > b },
			input:      map[string]interface{}{"COL1": []int{1, 2, 3}, "COL2": []int{10, 1, 10}},
			expected:   map[string]interface{}{"COL1": []int{1, 3}, "COL2": []int{10, 10}}},
		{
			name:       "built in bool compare",
			comparator: "=",
			input:      map[string]interface{}{"COL1": []bool{true, false, false}, "COL2": []bool{true, true, false}},
			expected:   map[string]interface{}{"COL1": []bool{true, false}, "COL2": []bool{true, false}}},
		{
			name:       "custom bool compare",
			comparator: func(a, b bool) bool { return a == b },
			input:      map[string]interface{}{"COL1": []bool{true, false, false}, "COL2": []bool{true, true, false}},
			expected:   map[string]interface{}{"COL1": []bool{true, false}, "COL2": []bool{true, false}}},
		{
			name:       "built in float compare",
			comparator: "<",
			input:      map[string]interface{}{"COL1": []float64{1, 2, 3}, "COL2": []float64{10, 1, 10}},
			expected:   map[string]interface{}{"COL1": []float64{2}, "COL2": []float64{1}}},
		{
			name:       "custon float compare",
			comparator: func(a, b float64) bool { return a < b },
			input:      map[string]interface{}{"COL1": []float64{1, 2, 3}, "COL2": []float64{10, 1, 10}},
			expected:   map[string]interface{}{"COL1": []float64{2}, "COL2": []float64{1}}},
		{
			name:       "built in string compare",
			comparator: "<",
			input:      map[string]interface{}{"COL1": []string{"a", "b", "c"}, "COL2": []string{"o", "a", "q"}},
			expected:   map[string]interface{}{"COL1": []string{"b"}, "COL2": []string{"a"}}},
		{
			name:       "custom string compare",
			comparator: func(a, b *string) bool { return *a < *b },
			input:      map[string]interface{}{"COL1": []string{"a", "b", "c"}, "COL2": []string{"o", "a", "q"}},
			expected:   map[string]interface{}{"COL1": []string{"b"}, "COL2": []string{"a"}}},
		{
			name:       "built in enum compare",
			comparator: "<",
			input:      map[string]interface{}{"COL1": []string{"a", "b", "c"}, "COL2": []string{"o", "a", "q"}},
			expected:   map[string]interface{}{"COL1": []string{"b"}, "COL2": []string{"a"}}},
		{
			name:       "custom enum compare",
			comparator: func(a, b *string) bool { return *a < *b },
			input:      map[string]interface{}{"COL1": []string{"a", "b", "c"}, "COL2": []string{"o", "a", "q"}},
			expected:   map[string]interface{}{"COL1": []string{"b"}, "COL2": []string{"a"}},
			configs: []qframe.ConfigFunc{qframe.Enums(map[string][]string{
				"COL1": {"a", "b", "c", "o", "q"},
				"COL2": {"a", "b", "c", "o", "q"},
			})}},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("Filter %d", i), func(t *testing.T) {
			input := qframe.New(tc.input, tc.configs...)
			output := input.Filter(filter.Filter{Comparator: tc.comparator, Column: "COL2", Arg: filter.ColumnName("COL1")})
			expected := qframe.New(tc.expected, tc.configs...)
			assertEquals(t, expected, output)
		})
	}
}

func TestQFrame_Sort(t *testing.T) {
	a := qframe.New(map[string]interface{}{
		"COL.1": []int{0, 1, 3, 2},
		"COL.2": []int{3, 2, 1, 1},
	})

	table := []struct {
		orders   []qframe.Order
		expected qframe.QFrame
	}{
		{
			[]qframe.Order{{Column: "COL.1"}},
			qframe.New(map[string]interface{}{
				"COL.1": []int{0, 1, 2, 3},
				"COL.2": []int{3, 2, 1, 1}}),
		},
		{
			[]qframe.Order{{Column: "COL.1", Reverse: true}},
			qframe.New(map[string]interface{}{
				"COL.1": []int{3, 2, 1, 0},
				"COL.2": []int{1, 1, 2, 3}}),
		},
		{
			[]qframe.Order{{Column: "COL.2"}, {Column: "COL.1"}},
			qframe.New(map[string]interface{}{
				"COL.1": []int{2, 3, 1, 0},
				"COL.2": []int{1, 1, 2, 3}}),
		},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("Sort %d", i), func(t *testing.T) {
			b := a.Sort(tc.orders...)
			assertEquals(t, tc.expected, b)
		})
	}
}

func TestQFrame_SortNull(t *testing.T) {
	a, b, c := "a", "b", "c"
	stringIn := map[string]interface{}{
		"COL.1": []*string{&b, nil, &a, nil, &c, &a, nil},
	}

	floatIn := map[string]interface{}{
		"COL.1": []float64{1.0, math.NaN(), -1.0, math.NaN()},
	}

	table := []struct {
		in       map[string]interface{}
		orders   []qframe.Order
		expected map[string]interface{}
	}{
		{
			stringIn,
			[]qframe.Order{{Column: "COL.1"}},
			map[string]interface{}{
				"COL.1": []*string{nil, nil, nil, &a, &a, &b, &c},
			},
		},
		{
			stringIn,
			[]qframe.Order{{Column: "COL.1", Reverse: true}},
			map[string]interface{}{
				"COL.1": []*string{&c, &b, &a, &a, nil, nil, nil},
			},
		},
		{
			floatIn,
			[]qframe.Order{{Column: "COL.1"}},
			map[string]interface{}{
				"COL.1": []float64{math.NaN(), math.NaN(), -1.0, 1.0},
			},
		},
		{
			floatIn,
			[]qframe.Order{{Column: "COL.1", Reverse: true}},
			map[string]interface{}{
				"COL.1": []float64{1.0, -1.0, math.NaN(), math.NaN()},
			},
		},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("Sort %d", i), func(t *testing.T) {
			in := qframe.New(tc.in)
			out := in.Sort(tc.orders...)
			assertNotErr(t, out.Err)
			assertEquals(t, qframe.New(tc.expected), out)
		})
	}
}

func TestQFrame_SortStability(t *testing.T) {
	a := qframe.New(map[string]interface{}{
		"COL.1": []int{0, 1, 3, 2},
		"COL.2": []int{1, 1, 1, 1},
	})

	table := []struct {
		orders   []qframe.Order
		expected qframe.QFrame
	}{
		{
			[]qframe.Order{{Column: "COL.2", Reverse: true}, {Column: "COL.1"}},
			qframe.New(map[string]interface{}{
				"COL.1": []int{0, 1, 2, 3},
				"COL.2": []int{1, 1, 1, 1}}),
		},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("Sort %d", i), func(t *testing.T) {
			b := a.Sort(tc.orders...)
			assertEquals(t, tc.expected, b)
		})
	}
}

func TestQFrame_Distinct(t *testing.T) {
	table := []struct {
		input    map[string]interface{}
		expected map[string]interface{}
		columns  []string
	}{
		{
			input: map[string]interface{}{
				"COL.1": []int{0, 1, 0, 1},
				"COL.2": []int{0, 1, 0, 1}},
			expected: map[string]interface{}{
				"COL.1": []int{0, 1},
				"COL.2": []int{0, 1}},
			columns: []string{"COL.1", "COL.2"},
		},
		{
			input: map[string]interface{}{
				"COL.1": []int{},
				"COL.2": []int{}},
			expected: map[string]interface{}{
				"COL.1": []int{},
				"COL.2": []int{}},
			columns: []string{"COL.1", "COL.2"},
		},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("Distinct %d", i), func(t *testing.T) {
			in := qframe.New(tc.input)
			out := in.Distinct()
			assertEquals(t, qframe.New(tc.expected), out)
		})
	}
}

func TestQFrame_GroupByAggregate(t *testing.T) {
	ownSum := func(col []int) int {
		result := 0
		for _, x := range col {
			result += x
		}
		return result
	}

	table := []struct {
		name         string
		input        map[string]interface{}
		expected     map[string]interface{}
		groupColumns []string
		aggregations []aggregation.Aggregation
	}{
		{
			name: "built in aggregation function",
			input: map[string]interface{}{
				"COL.1": []int{0, 0, 1, 2},
				"COL.2": []int{0, 0, 1, 1},
				"COL.3": []int{1, 2, 5, 7}},
			expected: map[string]interface{}{
				"COL.1": []int{0, 1, 2},
				"COL.2": []int{0, 1, 1},
				"COL.3": []int{3, 5, 7}},
			groupColumns: []string{"COL.1", "COL.2"},
			aggregations: []aggregation.Aggregation{aggregation.New("sum", "COL.3")},
		},
		{
			name: "user defined aggregation function",
			input: map[string]interface{}{
				"COL.1": []int{0, 0, 1, 1},
				"COL.2": []int{1, 2, 5, 7}},
			expected: map[string]interface{}{
				"COL.1": []int{0, 1},
				"COL.2": []int{3, 12}},
			groupColumns: []string{"COL.1"},
			aggregations: []aggregation.Aggregation{aggregation.New(ownSum, "COL.2")},
		},
		{
			name: "empty qframe",
			input: map[string]interface{}{
				"COL.1": []int{},
				"COL.2": []int{}},
			expected: map[string]interface{}{
				"COL.1": []int{},
				"COL.2": []int{}},
			groupColumns: []string{"COL.1"},
			aggregations: []aggregation.Aggregation{aggregation.New("sum", "COL.2")},
		},
	}

	for _, tc := range table {
		t.Run(fmt.Sprintf("GroupByAggregate %s", tc.name), func(t *testing.T) {
			in := qframe.New(tc.input)
			out := in.GroupBy(tc.groupColumns...).Aggregate(tc.aggregations...)
			assertEquals(t, qframe.New(tc.expected), out)
		})
	}
}

func TestQFrame_Select(t *testing.T) {
	table := []struct {
		input      map[string]interface{}
		expected   map[string]interface{}
		selectCols []string
	}{
		{
			input: map[string]interface{}{
				"COL.1": []int{0, 1},
				"COL.2": []int{1, 2}},
			expected: map[string]interface{}{
				"COL.1": []int{0, 1}},
			selectCols: []string{"COL.1"},
		},
		{
			input: map[string]interface{}{
				"COL.1": []int{0, 1},
				"COL.2": []int{1, 2}},
			expected:   map[string]interface{}{},
			selectCols: []string{},
		},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("Select %d", i), func(t *testing.T) {
			in := qframe.New(tc.input)
			out := in.Select(tc.selectCols...)
			assertEquals(t, qframe.New(tc.expected), out)
		})
	}
}

func TestQFrame_Slice(t *testing.T) {
	table := []struct {
		input    map[string]interface{}
		expected map[string]interface{}
		start    int
		end      int
	}{
		{
			input: map[string]interface{}{
				"COL.1": []float64{0.0, 1.5, 2.5, 3.5},
				"COL.2": []int{1, 2, 3, 4}},
			expected: map[string]interface{}{
				"COL.1": []float64{1.5, 2.5},
				"COL.2": []int{2, 3}},
			start: 1,
			end:   3,
		},
		{
			input: map[string]interface{}{
				"COL.1": []int{},
				"COL.2": []int{}},
			expected: map[string]interface{}{
				"COL.1": []int{},
				"COL.2": []int{}},
			start: 0,
			end:   0,
		},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("Slice %d", i), func(t *testing.T) {
			in := qframe.New(tc.input)
			out := in.Slice(tc.start, tc.end)
			assertEquals(t, qframe.New(tc.expected), out)
		})
	}
}

func TestQFrame_ReadCsv(t *testing.T) {
	/*
		Pandas reference
		>>> data = """
		... foo,bar,baz,qux
		... ccc,,,www
		... aaa,3.25,7,"""
		>>> pd.read_csv(StringIO(data))
		   foo   bar  baz  qux
		0  ccc   NaN  NaN  www
		1  aaa  3.25  7.0  NaN
	*/
	a, b, c, empty := "a", "b", "c", ""
	table := []struct {
		name         string
		inputHeaders []string
		inputData    string
		emptyNull    bool
		expected     map[string]interface{}
		types        map[string]string
		expectedErr  string
	}{
		{
			name:         "base",
			inputHeaders: []string{"foo", "bar"},
			inputData:    "1,2\n3,4\n",
			expected: map[string]interface{}{
				"foo": []int{1, 3},
				"bar": []int{2, 4}},
		},
		{
			name:         "mixed",
			inputHeaders: []string{"int", "float", "bool", "string"},
			inputData:    "1,2.5,true,hello\n10,20.5,false,\"bye, bye\"",
			expected: map[string]interface{}{
				"int":    []int{1, 10},
				"float":  []float64{2.5, 20.5},
				"bool":   []bool{true, false},
				"string": []string{"hello", "bye, bye"}},
		},
		{
			name:         "null string",
			inputHeaders: []string{"foo", "bar"},
			inputData:    "a,b\n,c",
			emptyNull:    true,
			expected: map[string]interface{}{
				"foo": []*string{&a, nil},
				"bar": []*string{&b, &c}},
		},
		{
			name:         "empty string",
			inputHeaders: []string{"foo", "bar"},
			inputData:    "a,b\n,c",
			emptyNull:    false,
			expected: map[string]interface{}{
				"foo": []*string{&a, &empty},
				"bar": []*string{&b, &c}},
		},
		{
			name:         "NaN float",
			inputHeaders: []string{"foo", "bar"},
			inputData:    "1.5,3.0\n,2.0",
			expected: map[string]interface{}{
				"foo": []float64{1.5, math.NaN()},
				"bar": []float64{3.0, 2.0}},
		},
		{
			name:         "Int to float type success",
			inputHeaders: []string{"foo"},
			inputData:    "3\n2",
			expected:     map[string]interface{}{"foo": []float64{3.0, 2.0}},
			types:        map[string]string{"foo": "float"},
		},
		{
			name:         "Bool to string success",
			inputHeaders: []string{"foo"},
			inputData:    "true\nfalse",
			expected:     map[string]interface{}{"foo": []string{"true", "false"}},
			types:        map[string]string{"foo": "string"},
		},
		{
			name:         "Int to string success",
			inputHeaders: []string{"foo"},
			inputData:    "123\n456",
			expected:     map[string]interface{}{"foo": []string{"123", "456"}},
			types:        map[string]string{"foo": "string"},
		},
		{
			name:         "Float to int failure",
			inputHeaders: []string{"foo"},
			inputData:    "1.23\n4.56",
			expectedErr:  "int",
			types:        map[string]string{"foo": "int"},
		},
		{
			name:         "String to bool failure",
			inputHeaders: []string{"foo"},
			inputData:    "abc\ndef",
			expectedErr:  "bool",
			types:        map[string]string{"foo": "bool"},
		},
		{
			name:         "String to float failure",
			inputHeaders: []string{"foo"},
			inputData:    "abc\ndef",
			expectedErr:  "float",
			types:        map[string]string{"foo": "float"},
		},
		{
			name:         "Enum with null value",
			inputHeaders: []string{"foo"},
			inputData:    "a\n\nc",
			types:        map[string]string{"foo": "enum"},
			emptyNull:    true,
			expected:     map[string]interface{}{"foo": []*string{&a, nil, &c}},
		},
	}

	for _, tc := range table {
		t.Run(fmt.Sprintf("ReadCsv %s", tc.name), func(t *testing.T) {
			input := strings.Join(tc.inputHeaders, ",") + "\n" + tc.inputData
			out := qframe.ReadCsv(strings.NewReader(input), qframe.EmptyNull(tc.emptyNull), qframe.Types(tc.types))
			if tc.expectedErr != "" {
				assertErr(t, out.Err, tc.expectedErr)
			} else {
				assertNotErr(t, out.Err)

				enums := make(map[string][]string)
				for k, v := range tc.types {
					if v == "enum" {
						enums[k] = nil
					}
				}

				assertEquals(t, qframe.New(tc.expected, qframe.ColumnOrder(tc.inputHeaders...), qframe.Enums(enums)), out)
			}
		})
	}
}

func TestQFrame_Enum(t *testing.T) {
	mon, tue, wed, thu, fri, sat, sun := "mon", "tue", "wed", "thu", "fri", "sat", "sun"
	t.Run("Applies specified order", func(t *testing.T) {
		input := `day
tue
mon
sat
wed
sun
thu
mon
thu

`
		out := qframe.ReadCsv(
			strings.NewReader(input),
			qframe.EmptyNull(true),
			qframe.Types(map[string]string{"day": "enum"}),
			qframe.EnumValues(map[string][]string{"day": {mon, tue, wed, thu, fri, sat, sun}}))
		out = out.Sort(qframe.Order{Column: "day"})
		expected := qframe.New(
			map[string]interface{}{"day": []*string{nil, &mon, &mon, &tue, &wed, &thu, &thu, &sat, &sun}},
			qframe.Enums(map[string][]string{"day": {mon, tue, wed, thu, fri, sat, sun}}))

		assertNotErr(t, out.Err)
		assertEquals(t, expected, out)
	})

	t.Run("Wont accept unknown values in strict mode", func(t *testing.T) {
		input := `day
tue
mon
foo
`
		out := qframe.ReadCsv(
			strings.NewReader(input),
			qframe.Types(map[string]string{"day": "enum"}),
			qframe.EnumValues(map[string][]string{"day": {mon, tue, wed, thu, fri, sat, sun}}))

		assertErr(t, out.Err, "unknown enum value")
	})

	t.Run("Fails with too high cardinality column", func(t *testing.T) {
		input := make([]string, 0)
		for i := 0; i < 256; i++ {
			input = append(input, strconv.Itoa(i))
		}

		out := qframe.New(
			map[string]interface{}{"foo": input},
			qframe.Enums(map[string][]string{"foo": nil}))

		assertErr(t, out.Err, "max cardinality")
	})

	t.Run("Fails when enum values specified for non enum column", func(t *testing.T) {
		input := `day
tue
`

		out := qframe.ReadCsv(
			strings.NewReader(input),
			qframe.EnumValues(map[string][]string{"day": {mon, tue, wed, thu, fri, sat, sun}}))

		assertErr(t, out.Err, "specified for non enum column")
	})
}

func TestQFrame_ReadJson(t *testing.T) {
	/*
		>>> pd.DataFrame.from_records([dict(a=1.5), dict(a=None)])
			 a
		0  1.5
		1  NaN
		>>> pd.DataFrame.from_records([dict(a=1), dict(a=None)])
			 a
		0  1.0
		1  NaN
		>>> pd.DataFrame.from_records([dict(a=1), dict(a=2)])
		   a
		0  1
		1  2
		>>> pd.DataFrame.from_records([dict(a='foo'), dict(a=None)])
			  a
		0   foo
		1  None
		>>> pd.DataFrame.from_records([dict(a=1.5), dict(a='N/A')])
			 a
		0  1.5
		1  N/A
		>>> x = pd.DataFrame.from_records([dict(a=1.5), dict(a='N/A')])
		>>> x.ix[0]
		a    1.5
		Name: 0, dtype: object
	*/
	testString := "FOO"
	table := []struct {
		input    string
		expected map[string]interface{}
	}{
		{
			input: `{"STRING1": ["a", "b"], "INT1": [1, 2], "FLOAT1": [1.5, 2.5], "BOOL1": [true, false]}`,
			expected: map[string]interface{}{
				"STRING1": []string{"a", "b"}, "INT1": []int{1, 2}, "FLOAT1": []float64{1.5, 2.5}, "BOOL1": []bool{true, false}},
		},
		{
			input:    `{"STRING1": ["FOO", null]}`,
			expected: map[string]interface{}{"STRING1": []*string{&testString, nil}},
		},
		{
			input: `[
				{"STRING1": "a", "INT1": 1, "FLOAT1": 1.5, "BOOL1": true},
				{"STRING1": "b", "INT1": 2, "FLOAT1": 2.5, "BOOL1": false}]`,
			expected: map[string]interface{}{
				// NOTE: The integers become floats if not explicitly typed
				"STRING1": []string{"a", "b"}, "INT1": []float64{1, 2}, "FLOAT1": []float64{1.5, 2.5}, "BOOL1": []bool{true, false}},
		},
		{
			input: `[{"STRING1": "FOO"}, {"STRING1": null}]`,
			expected: map[string]interface{}{
				"STRING1": []*string{&testString, nil}},
		},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("FromJSON %d", i), func(t *testing.T) {
			out := qframe.ReadJson(strings.NewReader(tc.input))
			assertNotErr(t, out.Err)
			assertEquals(t, qframe.New(tc.expected), out)
		})
	}
}

func TestQFrame_ToCsv(t *testing.T) {
	table := []struct {
		input    map[string]interface{}
		expected string
	}{
		{
			input: map[string]interface{}{
				"STRING1": []string{"a", "b,c"}, "INT1": []int{1, 2}, "FLOAT1": []float64{1.5, 2.5}, "BOOL1": []bool{true, false}},
			expected: `BOOL1,FLOAT1,INT1,STRING1
true,1.5,1,a
false,2.5,2,"b,c"
`,
		},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("ToCsv %d", i), func(t *testing.T) {
			in := qframe.New(tc.input)
			assertNotErr(t, in.Err)

			buf := new(bytes.Buffer)
			err := in.ToCsv(buf)
			assertNotErr(t, err)

			result := buf.String()
			if result != tc.expected {
				t.Errorf("QFrames not equal, %s ||| %s", result, tc.expected)
			}
		})
	}
}

func TestQFrame_ToFromJSON(t *testing.T) {
	config := []qframe.ConfigFunc{qframe.Enums(map[string][]string{"ENUM": {"aa", "bb"}})}
	table := []struct {
		orientation string
		configFuncs []qframe.ConfigFunc
	}{
		{orientation: "records"},
		{orientation: "columns"},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("ToFromJSON %d", i), func(t *testing.T) {
			buf := new(bytes.Buffer)
			data := map[string]interface{}{
				"STRING1": []string{"añ", "bö☺	"}, "FLOAT1": []float64{1.5, 2.5}, "BOOL1": []bool{true, false}, "ENUM": []string{"aa", "bb"}}
			originalDf := qframe.New(data, config...)
			assertNotErr(t, originalDf.Err)

			err := originalDf.ToJson(buf, tc.orientation)
			assertNotErr(t, err)

			jsonDf := qframe.ReadJson(buf, config...)
			assertNotErr(t, jsonDf.Err)
			assertEquals(t, originalDf, jsonDf)
		})
	}
}

func TestQFrame_ToJSONNaN(t *testing.T) {
	table := []struct {
		orientation string
		expected    string
	}{
		{orientation: "records", expected: `[{"FLOAT1":1.5},{"FLOAT1":NaN}]`},
		{orientation: "columns", expected: `{"FLOAT1":[1.5,NaN]}`},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("ToFromJSON %d", i), func(t *testing.T) {
			buf := new(bytes.Buffer)

			// Test the special case NaN, this can currently be encoded but not
			// decoded by the JSON parsers.
			data := map[string]interface{}{"FLOAT1": []float64{1.5, math.NaN()}}
			originalDf := qframe.New(data)
			assertNotErr(t, originalDf.Err)

			err := originalDf.ToJson(buf, tc.orientation)
			assertNotErr(t, err)
			if buf.String() != tc.expected {
				t.Errorf("Not equal: %s ||| %s", buf.String(), tc.expected)
			}
		})
	}
}

func TestQFrame_FilterEnum(t *testing.T) {
	a, b, c, d, e := "a", "b", "c", "d", "e"
	enums := qframe.Enums(map[string][]string{"COL1": {"a", "b", "c", "d", "e"}})
	in := qframe.New(map[string]interface{}{
		"COL1": []*string{&b, &c, &a, nil, &e, &d, nil}}, enums)

	table := []struct {
		filters  []filter.Filter
		expected map[string]interface{}
	}{
		{
			[]filter.Filter{{Column: "COL1", Comparator: ">", Arg: "b"}},
			map[string]interface{}{"COL1": []*string{&c, &e, &d}},
		},
		{
			[]filter.Filter{{Column: "COL1", Comparator: "in", Arg: []string{"a", "b"}}},
			map[string]interface{}{"COL1": []*string{&b, &a}},
		},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("Filter enum %d", i), func(t *testing.T) {
			expected := qframe.New(tc.expected, enums)
			out := in.Filter(tc.filters...)
			assertEquals(t, expected, out)
		})
	}
}

func TestQFrame_FilterString(t *testing.T) {
	a, b, c, d, e := "a", "b", "c", "d", "e"
	withNil := map[string]interface{}{"COL1": []*string{&b, &c, &a, nil, &e, &d, nil}}

	table := []struct {
		input    map[string]interface{}
		filters  []filter.Filter
		expected map[string]interface{}
	}{
		{
			withNil,
			[]filter.Filter{{Column: "COL1", Comparator: ">", Arg: "b"}},
			map[string]interface{}{"COL1": []*string{&c, &e, &d}},
		},
		{
			withNil,
			[]filter.Filter{{Column: "COL1", Comparator: "<", Arg: "b"}},
			map[string]interface{}{"COL1": []*string{&a, nil, nil}},
		},
		{
			withNil,
			[]filter.Filter{{Column: "COL1", Comparator: "like", Arg: "b"}},
			map[string]interface{}{"COL1": []*string{&b}},
		},
		{
			withNil,
			[]filter.Filter{{Column: "COL1", Comparator: "in", Arg: []string{"a", "b"}}},
			map[string]interface{}{"COL1": []*string{&b, &a}},
		},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("Filter string %d", i), func(t *testing.T) {
			in := qframe.New(tc.input)
			expected := qframe.New(tc.expected)
			out := in.Filter(tc.filters...)
			assertEquals(t, expected, out)
		})
	}
}

func TestQFrame_LikeFilterString(t *testing.T) {
	col1 := []string{"ABC", "AbC", "DEF", "ABCDEF", "abcdef", "FFF", "abc$def", "défåäöΦ"}

	// Add a couple of fields to be able to verify functionality for high cardinality enums
	for i := 0; i < 200; i++ {
		col1 = append(col1, fmt.Sprintf("foo%dbar", i))
	}

	data := map[string]interface{}{"COL1": col1}
	for _, enums := range []map[string][]string{{}, {"COL1": nil}} {
		table := []struct {
			comparator string
			arg        string
			expected   []string
		}{
			// like
			{"like", ".*EF.*", []string{"DEF", "ABCDEF"}},
			{"like", "%EF%", []string{"DEF", "ABCDEF"}},
			{"like", "AB%", []string{"ABC", "ABCDEF"}},
			{"like", "%F", []string{"DEF", "ABCDEF", "FFF"}},
			{"like", "ABC", []string{"ABC"}},
			{"like", "défåäöΦ", []string{"défåäöΦ"}},
			{"like", "%éfåäöΦ", []string{"défåäöΦ"}},
			{"like", "défå%", []string{"défåäöΦ"}},
			{"like", "%éfåäö%", []string{"défåäöΦ"}},
			{"like", "abc$def", []string{}},
			{"like", regexp.QuoteMeta("abc$def"), []string{"abc$def"}},
			{"like", "%180%", []string{"foo180bar"}},

			// ilike
			{"ilike", ".*ef.*", []string{"DEF", "ABCDEF", "abcdef", "abc$def"}},
			{"ilike", "ab%", []string{"ABC", "AbC", "ABCDEF", "abcdef", "abc$def"}},
			{"ilike", "%f", []string{"DEF", "ABCDEF", "abcdef", "FFF", "abc$def"}},
			{"ilike", "%ef%", []string{"DEF", "ABCDEF", "abcdef", "abc$def"}},
			{"ilike", "défÅäöΦ", []string{"défåäöΦ"}},
			{"ilike", "%éFåäöΦ", []string{"défåäöΦ"}},
			{"ilike", "défå%", []string{"défåäöΦ"}},
			{"ilike", "%éfåäÖ%", []string{"défåäöΦ"}},
			{"ilike", "ABC$def", []string{}},
			{"ilike", regexp.QuoteMeta("abc$DEF"), []string{"abc$def"}},
			{"ilike", "%180%", []string{"foo180bar"}},
		}

		for _, tc := range table {
			t.Run(fmt.Sprintf("Enum %t, %s %s", len(enums) > 0, tc.comparator, tc.arg), func(t *testing.T) {
				in := qframe.New(data, qframe.Enums(enums))
				expected := qframe.New(map[string]interface{}{"COL1": tc.expected}, qframe.Enums(enums))
				out := in.Filter(filter.Filter{Column: "COL1", Comparator: tc.comparator, Arg: tc.arg})
				assertEquals(t, expected, out)
			})
		}
	}
}

func TestQFrame_String(t *testing.T) {
	a := qframe.New(map[string]interface{}{
		"COLUMN1": []string{"Long content", "a", "b", "c"},
		"COL2":    []int{3, 2, 1, 123456},
	}, qframe.ColumnOrder("COL2", "COLUMN1"))

	expected := ` COL2 COLUMN1
----- -------
    3 Long...
    2       a
    1       b
12...       c`

	if expected != a.String() {
		t.Errorf("\n%s\n != \n%s", expected, a.String())
	}
}

func TestQFrame_ByteSize(t *testing.T) {
	a := qframe.New(map[string]interface{}{
		"COL1": []string{"a", "b"},
		"COL2": []int{3, 2},
		"COL3": []float64{3.5, 2.0},
		"COL4": []bool{true, false},
		"COL5": []string{"1", "2"},
	}, qframe.Enums(map[string][]string{"COL5": nil}))
	totalSize := a.ByteSize()

	// Not so much of a test as lock down on behavior to detect changes
	expectedSize := 740
	if totalSize != expectedSize {
		t.Errorf("Unexpected byte size: %d != %d", totalSize, expectedSize)
	}

	assertTrue(t, a.Select("COL1", "COL2", "COL3", "COL4").ByteSize() < totalSize)
	assertTrue(t, a.Select("COL2", "COL3", "COL4", "COL5").ByteSize() < totalSize)
	assertTrue(t, a.Select("COL1", "COL3", "COL4", "COL5").ByteSize() < totalSize)
	assertTrue(t, a.Select("COL1", "COL2", "COL4", "COL5").ByteSize() < totalSize)
	assertTrue(t, a.Select("COL1", "COL2", "COL3", "COL5").ByteSize() < totalSize)
}

func TestQFrame_CopyColumn(t *testing.T) {
	input := qframe.New(map[string]interface{}{
		"COL1": []string{"a", "b"},
		"COL2": []int{3, 2},
	})

	expectedNew := qframe.New(map[string]interface{}{
		"COL1": []string{"a", "b"},
		"COL2": []int{3, 2},
		"COL3": []int{3, 2},
	})

	expectedReplace := qframe.New(map[string]interface{}{
		"COL1": []int{3, 2},
		"COL2": []int{3, 2},
	})

	assertEquals(t, expectedNew, input.Copy("COL3", "COL2"))
	assertEquals(t, expectedReplace, input.Copy("COL1", "COL2"))
}

func TestQFrame_AssignZeroArg(t *testing.T) {
	a, b := "a", "b"
	table := []struct {
		name     string
		expected interface{}
		fn       interface{}
	}{
		{name: "int fn", expected: []int{2, 2}, fn: func() int { return 2 }},
		{name: "int const", expected: []int{3, 3}, fn: 3},
		{name: "float fn", expected: []float64{2.5, 2.5}, fn: func() float64 { return 2.5 }},
		{name: "float const", expected: []float64{3.5, 3.5}, fn: 3.5},
		{name: "bool fn", expected: []bool{true, true}, fn: func() bool { return true }},
		{name: "bool const", expected: []bool{false, false}, fn: false},
		{name: "string fn", expected: []*string{&a, &a}, fn: func() *string { return &a }},
		{name: "bool const", expected: []*string{&b, &b}, fn: &b},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			input := map[string]interface{}{"COL1": []int{3, 2}}
			in := qframe.New(input)
			input["COL2"] = tc.expected
			expected := qframe.New(input)
			out := in.Assign(qframe.Instruction{Fn: tc.fn, DstCol: "COL2"})
			assertEquals(t, expected, out)
		})
	}
}

func TestQFrame_AssignSingleArgIntToInt(t *testing.T) {
	input := qframe.New(map[string]interface{}{
		"COL1": []int{3, 2},
	})

	expectedNew := qframe.New(map[string]interface{}{
		"COL1": []int{6, 4},
	})

	assertEquals(t, expectedNew, input.Assign(qframe.Instruction{Fn: func(a int) (int, error) { return 2 * a, nil }, DstCol: "COL1", SrcCol1: "COL1"}))
}

func TestQFrame_AssignSingleArgStringToBool(t *testing.T) {
	input := qframe.New(map[string]interface{}{
		"COL1": []string{"a", "aa", "aaa"},
	})

	expectedNew := qframe.New(map[string]interface{}{
		"COL1":    []string{"a", "aa", "aaa"},
		"IS_LONG": []bool{false, false, true},
	})

	assertEquals(t, expectedNew, input.Assign(qframe.Instruction{Fn: func(x *string) (bool, error) { return len(*x) > 2, nil }, DstCol: "IS_LONG", SrcCol1: "COL1"}))
}

func toUpper(x *string) (*string, error) {
	if x == nil {
		return x, nil
	}
	result := strings.ToUpper(*x)
	return &result, nil
}

func TestQFrame_AssignSingleArgString(t *testing.T) {
	a, b := "a", "b"
	A, B := "A", "B"
	input := qframe.New(map[string]interface{}{
		"COL1": []*string{&a, &b, nil},
	})

	expectedNew := qframe.New(map[string]interface{}{
		"COL1": []*string{&A, &B, nil},
	})

	// General function
	assertEquals(t, expectedNew, input.Assign(qframe.Instruction{Fn: toUpper, DstCol: "COL1", SrcCol1: "COL1"}))

	// Built in function
	assertEquals(t, expectedNew, input.Assign(qframe.Instruction{Fn: "ToUpper", DstCol: "COL1", SrcCol1: "COL1"}))
}

func TestQFrame_AssignSingleArgEnum(t *testing.T) {
	a, b := "a", "b"
	A, B := "A", "B"
	input := qframe.New(
		map[string]interface{}{"COL1": []*string{&a, &b, nil, &a}},
		qframe.Enums(map[string][]string{"COL1": nil}))

	expectedData := map[string]interface{}{"COL1": []*string{&A, &B, nil, &A}}
	expectedNewGeneral := qframe.New(expectedData)
	expectedNewBuiltIn := qframe.New(expectedData, qframe.Enums(map[string][]string{"COL1": nil}))

	// General function
	assertEquals(t, expectedNewGeneral, input.Assign(qframe.Instruction{Fn: toUpper, DstCol: "COL1", SrcCol1: "COL1"}))

	// Builtin function
	assertEquals(t, expectedNewBuiltIn, input.Assign(qframe.Instruction{Fn: "ToUpper", DstCol: "COL1", SrcCol1: "COL1"}))
}

func TestQFrame_AssignDoubleArg(t *testing.T) {
	table := []struct {
		name     string
		input    map[string]interface{}
		expected interface{}
		fn       interface{}
		enums    map[string][]string
	}{
		{
			name:     "int",
			input:    map[string]interface{}{"COL1": []int{3, 2}, "COL2": []int{30, 20}},
			expected: []int{33, 22},
			fn:       func(a, b int) (int, error) { return a + b, nil }},
		{
			name:     "string",
			input:    map[string]interface{}{"COL1": []string{"a", "b"}, "COL2": []string{"x", "y"}},
			expected: []string{"ax", "by"},
			fn:       func(a, b *string) (*string, error) { result := *a + *b; return &result, nil }},
		{
			name:     "enum",
			input:    map[string]interface{}{"COL1": []string{"a", "b"}, "COL2": []string{"x", "y"}},
			expected: []string{"ax", "by"},
			fn:       func(a, b *string) (*string, error) { result := *a + *b; return &result, nil },
			enums:    map[string][]string{"COL1": nil, "COL2": nil}},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			in := qframe.New(tc.input, qframe.Enums(tc.enums))
			tc.input["COL3"] = tc.expected
			expected := qframe.New(tc.input, qframe.Enums(tc.enums))
			out := in.Assign(qframe.Instruction{Fn: tc.fn, DstCol: "COL3", SrcCol1: "COL1", SrcCol2: "COL2"})
			assertEquals(t, expected, out)
		})
	}
}

func TestQFrame_FilteredAssign(t *testing.T) {
	plus1 := func(a int) (int, error) { return a + 1, nil }
	table := []struct {
		name         string
		input        map[string]interface{}
		expected     map[string]interface{}
		instructions []qframe.Instruction
		clauses      qframe.FilterClause
	}{
		{
			name:         "null fills for rows that dont match filter when destination column is new",
			input:        map[string]interface{}{"COL1": []int{3, 2, 1}},
			instructions: []qframe.Instruction{{Fn: plus1, DstCol: "COL3", SrcCol1: "COL1"}, {Fn: plus1, DstCol: "COL3", SrcCol1: "COL3"}},
			expected:     map[string]interface{}{"COL1": []int{3, 2, 1}, "COL3": []int{5, 4, 0}},
			clauses:      qframe.Filter{Comparator: ">", Column: "COL1", Arg: 1}},
		{
			// One could question whether this is the desired behaviour or not. The alternative
			// would be to preserve the existing values but that would cause a lot of inconsistencies
			// when the result column type differs from the source column type for example. What would
			// the preserved value be in that case? Preserving the existing behaviour could be achieved
			// by using a temporary column that indexes which columns to modify and not. Perhaps this
			// should be built in at some point.
			name:         "null fills rows that dont match filter when destination column is existing",
			input:        map[string]interface{}{"COL1": []int{3, 2, 1}},
			instructions: []qframe.Instruction{{Fn: plus1, DstCol: "COL1", SrcCol1: "COL1"}},
			expected:     map[string]interface{}{"COL1": []int{4, 3, 0}},
			clauses:      qframe.Filter{Comparator: ">", Column: "COL1", Arg: 1}},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			in := qframe.New(tc.input)
			expected := qframe.New(tc.expected)
			out := in.FilteredAssign(tc.clauses, tc.instructions...)
			assertEquals(t, expected, out)
		})
	}
}

func TestQFrame_AggregateStrings(t *testing.T) {
	table := []struct {
		enums map[string][]string
	}{
		{map[string][]string{"COL2": nil}},
		{map[string][]string{}},
	}

	for _, tc := range table {
		t.Run(fmt.Sprintf("Enum %t", len(tc.enums) > 0), func(t *testing.T) {
			input := qframe.New(map[string]interface{}{
				"COL1": []string{"a", "b", "a", "b", "a"},
				"COL2": []string{"x", "p", "y", "q", "z"},
			}, qframe.Enums(tc.enums))
			expected := qframe.New(map[string]interface{}{"COL1": []string{"a", "b"}, "COL2": []string{"x,y,z", "p,q"}})
			out := input.GroupBy("COL1").Aggregate(aggregation.New(aggregation.StrJoin(","), "COL2"))
			assertEquals(t, expected, out)
		})
	}
}

func TestQFrame_InitWithConstantVal(t *testing.T) {
	a := "a"
	table := []struct {
		name     string
		input    interface{}
		expected interface{}
		enums    map[string][]string
	}{
		{
			name:     "int",
			input:    qframe.ConstInt{Val: 33, Count: 2},
			expected: []int{33, 33}},
		{
			name:     "float",
			input:    qframe.ConstFloat{Val: 33.5, Count: 2},
			expected: []float64{33.5, 33.5}},
		{
			name:     "bool",
			input:    qframe.ConstBool{Val: true, Count: 2},
			expected: []bool{true, true}},
		{
			name:     "string",
			input:    qframe.ConstString{Val: &a, Count: 2},
			expected: []string{"a", "a"}},
		{
			name:     "string null",
			input:    qframe.ConstString{Val: nil, Count: 2},
			expected: []*string{nil, nil}},
		{
			name:     "enum",
			input:    qframe.ConstString{Val: &a, Count: 2},
			expected: []string{"a", "a"},
			enums:    map[string][]string{"COL1": nil}},
		{
			name:     "enum null",
			input:    qframe.ConstString{Val: nil, Count: 2},
			expected: []*string{nil, nil},
			enums:    map[string][]string{"COL1": nil}},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			in := qframe.New(map[string]interface{}{"COL1": tc.input}, qframe.Enums(tc.enums))
			expected := qframe.New(map[string]interface{}{"COL1": tc.expected}, qframe.Enums(tc.enums))
			assertEquals(t, expected, in)
		})
	}
}

func TestQFrame_FloatView(t *testing.T) {
	input := qframe.New(map[string]interface{}{"COL1": []float64{1.5, 0.5, 3.0}})
	input = input.Sort(qframe.Order{Column: "COL1"})
	expected := []float64{0.5, 1.5, 3.0}

	v, err := input.FloatView("COL1")
	assertNotErr(t, err)

	s := v.Slice()
	assertTrue(t, v.Len() == len(expected))
	assertTrue(t, len(s) == len(expected))
	assertTrue(t, (v.ItemAt(0) == s[0]) && (s[0] == expected[0]))
	assertTrue(t, (v.ItemAt(1) == s[1]) && (s[1] == expected[1]))
	assertTrue(t, (v.ItemAt(2) == s[2]) && (s[2] == expected[2]))
}

func TestQFrame_StringView(t *testing.T) {
	a, b := "a", "b"
	input := qframe.New(map[string]interface{}{"COL1": []*string{&a, nil, &b}})
	input = input.Sort(qframe.Order{Column: "COL1"})
	expected := []*string{nil, &a, &b}

	v, err := input.StringView("COL1")
	assertNotErr(t, err)

	s := v.Slice()
	assertTrue(t, v.Len() == len(expected))
	assertTrue(t, len(s) == len(expected))

	// Nil, check pointers
	assertTrue(t, (v.ItemAt(0) == s[0]) && (s[0] == expected[0]))

	// !Nil, check values
	assertTrue(t, (*v.ItemAt(1) == *s[1]) && (*s[1] == *expected[1]))
	assertTrue(t, (*v.ItemAt(2) == *s[2]) && (*s[2] == *expected[2]))
}

func TestQFrame_EnumView(t *testing.T) {
	a, b := "a", "b"
	input := qframe.New(map[string]interface{}{"COL1": []*string{&a, nil, &b}}, qframe.Enums(map[string][]string{"COL1": {"a", "b"}}))
	input = input.Sort(qframe.Order{Column: "COL1"})
	expected := []*string{nil, &a, &b}

	v, err := input.EnumView("COL1")
	assertNotErr(t, err)

	s := v.Slice()
	assertTrue(t, v.Len() == len(expected))
	assertTrue(t, len(s) == len(expected))

	// Nil, check pointers
	assertTrue(t, (v.ItemAt(0) == s[0]) && (s[0] == expected[0]))

	// !Nil, check values
	assertTrue(t, (*v.ItemAt(1) == *s[1]) && (*s[1] == *expected[1]))
	assertTrue(t, (*v.ItemAt(2) == *s[2]) && (*s[2] == *expected[2]))
}
