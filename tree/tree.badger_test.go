package tree

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"redwood.dev/types"
)

func TestVersionedDBTree_Value_MapWithRange(t *testing.T) {
	tests := []struct {
		start, end int64
		expected   interface{}
	}{
		{0, 1, M{
			"asdf": S{"1234", float64(987.2), uint64(333)}},
		},
		{0, 2, M{
			"asdf": S{"1234", float64(987.2), uint64(333)},
			"flo":  float64(321),
		}},
		{1, 2, M{
			"flo": float64(321),
		}},
		{1, 3, M{
			"flo": float64(321),
			"flox": S{
				uint64(65),
				M{"yup": "yes", "hey": uint64(321)},
				"jkjkjkj",
			},
		}},
		{0, 5, M{
			"asdf": S{"1234", float64(987.2), uint64(333)},
			"flo":  float64(321),
			"flox": S{
				uint64(65),
				M{"yup": "yes", "hey": uint64(321)},
				"jkjkjkj",
			},
			"floxxx": "asdf123",
			"hello": M{
				"xyzzy": uint64(33),
			},
		}},
		{0, 0, M{}},
		{5, 5, ErrInvalidRange},
		{6, 6, ErrInvalidRange},
		{-2, 0, M{
			"floxxx": "asdf123",
			"hello": M{
				"xyzzy": uint64(33),
			},
		}},
	}

	rootKeypaths := []Keypath{Keypath(nil)}

	for _, rootKeypath := range rootKeypaths {
		for _, test := range tests {
			test := test
			rootKeypath := rootKeypath
			name := fmt.Sprintf("%v[%v:%v]", rootKeypath, test.start, test.end)

			t.Run(name, func(t *testing.T) {
				db := setupVersionedDBTreeWithValue(t, rootKeypath, fixture1.input)
				defer db.DeleteDB()

				state := db.StateAtVersion(nil, false)

				val, exists, err := state.Value(rootKeypath, &Range{test.start, test.end})
				switch exp := test.expected.(type) {
				case error:
					require.True(t, errors.Cause(exp) == test.expected)
				default:
					require.NoError(t, err)
					require.True(t, exists)
					require.Equal(t, exp, val)
				}
			})
		}
	}
}

func TestVersionedDBTree_Value_SliceWithRange(t *testing.T) {
	tests := []struct {
		start, end int64
		expected   interface{}
	}{
		{0, 1, S{
			uint64(8383),
		}},
		{0, 2, S{
			uint64(8383),
			M{"9999": "hi", "vvvv": "yeah"},
		}},
		{1, 2, S{
			M{"9999": "hi", "vvvv": "yeah"},
		}},
		{1, 3, S{
			M{"9999": "hi", "vvvv": "yeah"},
			float64(321.23),
		}},
		{0, 3, S{
			uint64(8383),
			M{"9999": "hi", "vvvv": "yeah"},
			float64(321.23),
		}},
		{0, 0, S{}},
		{4, 4, ErrInvalidRange},
		{-2, 0, S{
			float64(321.23),
			"hello",
		}},
		{-2, -1, S{
			float64(321.23),
		}},
	}

	for _, test := range tests {
		test := test
		name := fmt.Sprintf("[%v : %v]", test.start, test.end)
		t.Run(name, func(t *testing.T) {
			db := setupVersionedDBTreeWithValue(t, nil, fixture3.input)
			defer db.DeleteDB()

			state := db.StateAtVersion(nil, false)

			val, exists, err := state.Value(Keypath(nil), &Range{test.start, test.end})
			switch exp := test.expected.(type) {
			case error:
				require.True(t, errors.Cause(exp) == test.expected)
			default:
				require.NoError(t, err)
				require.True(t, exists)
				require.Equal(t, exp, val)
			}
		})
	}
}

func TestDBNode_Set_NoRange(t *testing.T) {
	t.Run("slice", func(t *testing.T) {
		db := setupVersionedDBTreeWithValue(t, Keypath("data"), fixture1.input)
		defer db.DeleteDB()

		state := db.StateAtVersion(nil, true)

		fmt.Println("============")

		err := state.Set(Keypath("data/flox"), nil, S{"a", "b", "c", "d"})
		require.NoError(t, err)

		err = state.Save()
		require.NoError(t, err)

		state = db.StateAtVersion(nil, false)
		defer state.Close()
		state.DebugPrint(debugPrint, true, 0)

		val, exists, err := state.Value(Keypath("data/flox"), nil)
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, S{"a", "b", "c", "d"}, val)
	})

	t.Run("struct", func(t *testing.T) {
		type SomeStruct struct {
			Foo string `tree:"foo"`
			Bar uint64 `tree:"bar"`
		}
		type TestStruct struct {
			Asdf       []interface{}          `tree:"asdf"`
			Flo        float64                `tree:"flo"`
			Flox       []interface{}          `tree:"flox"`
			Floxx      string                 `tree:"floxx"`
			Hello      map[string]interface{} `tree:"hello"`
			SomeStruct SomeStruct             `tree:"someStruct"`
		}

		val := TestStruct{
			Asdf: S{"1234", float64(987.2), uint64(333)},
			Flo:  321,
			Flox: S{
				uint64(65),
				M{"yup": "yes", "hey": uint64(321)},
				"jkjkjkj",
			},
			Floxx: "asdf123",
			Hello: M{
				"xyzzy": uint64(33),
			},
			SomeStruct: SomeStruct{
				Foo: "fooooo",
				Bar: 54321,
			},
		}

		expected := M{
			"asdf": S{"1234", float64(987.2), uint64(333)},
			"flo":  float64(321),
			"flox": S{
				uint64(65),
				M{"yup": "yes", "hey": uint64(321)},
				"jkjkjkj",
			},
			"floxx": "asdf123",
			"hello": M{
				"xyzzy": uint64(33),
			},
			"someStruct": M{
				"foo": "fooooo",
				"bar": uint64(54321),
			},
		}

		db := setupDBTree(t)
		defer db.DeleteDB()

		state := db.State(true)

		err := state.Set(Keypath("data"), nil, val)
		require.NoError(t, err)

		err = state.Save()
		require.NoError(t, err)

		state = db.State(false)
		got, exists, err := state.Value(Keypath("data"), nil)
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, expected, got)
	})

	// t.Run("memory node", func(t *testing.T) {
	// 	db := setupVersionedDBTreeWithValue(t, Keypath("data"), fixture1.input)
	// 	defer db.DeleteDB()

	// 	state := db.StateAtVersion(nil, true)

	// 	memNode := NewMemoryNode()

	// 	memNode.Set(nil, nil, M{
	// 		"foo": M{"one": uint64(1), "two": uint64(2)},
	// 		"bar": S{"hi", float64(123)},
	// 	})

	// 	err := state.Set(Keypath("data/flox"), nil, memNode)
	// 	require.NoError(t, err)

	// 	err = state.Save()
	// 	require.NoError(t, err)

	// 	state = db.StateAtVersion(nil, false)
	// 	state.DebugPrint(debugPrint, true, 0)
	// })

	// t.Run("db node inside memory node", func(t *testing.T) {
	// 	db := setupVersionedDBTreeWithValue(t, Keypath("data"), fixture1.input)
	// 	defer db.DeleteDB()

	// 	state := db.StateAtVersion(nil, true)

	// 	memNode := NewMemoryNode()
	// 	innerDBNode := state.NodeAt(Keypath("data/flox"), nil)

	// 	memNode.Set(nil, nil, M{
	// 		"foo": innerDBNode,
	// 	})

	// 	memNode.DebugPrint(debugPrint, true, 0)

	// 	err := state.Set(Keypath("data/hello/xyzzy"), nil, memNode)
	// 	require.NoError(t, err)

	// 	err = state.Save()
	// 	require.NoError(t, err)

	// 	state = db.StateAtVersion(nil, false)
	// 	state.DebugPrint(debugPrint, true, 0)
	// })
}

func TestDBNode_Scan(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		type SomeStruct struct {
			Foo string `tree:"foo"`
			Bar uint64 `tree:"bar"`
		}
		type TestStruct struct {
			Asdf       []interface{}          `tree:"asdf"`
			Flo        float64                `tree:"flo"`
			Flox       []interface{}          `tree:"flox"`
			Floxx      string                 `tree:"floxx"`
			Hello      map[string]interface{} `tree:"hello"`
			SomeStruct SomeStruct             `tree:"someStruct"`
		}

		expected := TestStruct{
			Asdf: S{"1234", float64(987.2), uint64(333)},
			Flo:  321,
			Flox: S{
				uint64(65),
				M{"yup": "yes", "hey": uint64(321)},
				"jkjkjkj",
			},
			Floxx: "asdf123",
			Hello: M{
				"xyzzy": uint64(33),
			},
			SomeStruct: SomeStruct{
				Foo: "fooooo",
				Bar: 54321,
			},
		}

		fixture := M{
			"asdf": S{"1234", float64(987.2), uint64(333)},
			"flo":  float64(321),
			"flox": S{
				uint64(65),
				M{"yup": "yes", "hey": uint64(321)},
				"jkjkjkj",
			},
			"floxx": "asdf123",
			"hello": M{
				"xyzzy": uint64(33),
			},
			"someStruct": M{
				"foo": "fooooo",
				"bar": uint64(54321),
			},
		}

		db := setupDBTreeWithValue(t, Keypath("data"), fixture)
		defer db.DeleteDB()

		state := db.State(false)
		var got TestStruct
		err := state.NodeAt(Keypath("data"), nil).Scan(&got)
		require.NoError(t, err)
		require.Equal(t, expected, got)
	})
}

func TestVersionedDBTree_Set_Range_String(t *testing.T) {
	i := rand.Int()
	tree, err := NewVersionedDBTree(fmt.Sprintf("/tmp/tree-badger-test-%v", i))
	require.NoError(t, err)
	defer tree.DeleteDB()
	v := types.RandomID()

	err = tree.Update(&v, func(tx *DBNode) error {
		err := tx.Set(Keypath("foo/string"), nil, "abcdefgh")
		require.NoError(t, err)
		return nil
	})
	require.NoError(t, err)

	state := tree.StateAtVersion(&v, false)
	defer state.Close()

	str, exists, err := state.Value(Keypath("foo/string"), nil)
	require.True(t, exists)
	require.NoError(t, err)
	require.Equal(t, "abcdefgh", str)
	state.Close()

	err = tree.Update(&v, func(tx *DBNode) error {
		err := tx.Set(Keypath("foo/string"), &Range{3, 6}, "xx")
		require.NoError(t, err)
		return nil
	})
	require.NoError(t, err)

	state = tree.StateAtVersion(&v, false)
	defer state.Close()

	str, exists, err = state.Value(Keypath("foo/string"), nil)
	require.True(t, exists)
	require.NoError(t, err)
	require.Equal(t, "abcxxgh", str)
}

func TestVersionedDBTree_Set_Range_Slice(t *testing.T) {

	tests := []struct {
		name          string
		setKeypath    Keypath
		setRange      *Range
		setVals       []interface{}
		expectedSlice []interface{}
	}{
		{"start grow", Keypath("foo/slice"), &Range{0, 2}, S{testVal5, testVal6, testVal7, testVal8},
			S{testVal5, testVal6, testVal7, testVal8, testVal3, testVal4}},
		{"start same", Keypath("foo/slice"), &Range{0, 2}, S{testVal5, testVal6},
			S{testVal5, testVal6, testVal3, testVal4}},
		{"start shrink", Keypath("foo/slice"), &Range{0, 2}, S{testVal5},
			S{testVal5, testVal3, testVal4}},
		{"middle grow", Keypath("foo/slice"), &Range{1, 3}, S{testVal5, testVal6, testVal7, testVal8},
			S{testVal1, testVal5, testVal6, testVal7, testVal8, testVal4}},
		{"middle same", Keypath("foo/slice"), &Range{1, 3}, S{testVal5, testVal6},
			S{testVal1, testVal5, testVal6, testVal4}},
		{"middle shrink", Keypath("foo/slice"), &Range{1, 3}, S{testVal5},
			S{testVal1, testVal5, testVal4}},
		{"end grow", Keypath("foo/slice"), &Range{2, 4}, S{testVal5, testVal6, testVal7, testVal8},
			S{testVal1, testVal2, testVal5, testVal6, testVal7, testVal8}},
		{"end same", Keypath("foo/slice"), &Range{2, 4}, S{testVal5, testVal6},
			S{testVal1, testVal2, testVal5, testVal6}},
		{"end shrink", Keypath("foo/slice"), &Range{1, 4}, S{testVal5},
			S{testVal1, testVal5}},
		{"end append", Keypath("foo/slice"), &Range{4, 4}, S{testVal5, testVal6, testVal7, testVal8},
			S{testVal1, testVal2, testVal3, testVal4, testVal5, testVal6, testVal7, testVal8}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			db := setupVersionedDBTreeWithValue(t, nil, M{
				"foo": M{
					"bar":   M{"baz": uint64(123)},
					"slice": S{testVal1, testVal2, testVal3, testVal4},
				},
			})
			defer db.DeleteDB()

			state := db.StateAtVersion(nil, true)
			defer state.Close()

			err := state.Set(test.setKeypath, test.setRange, test.setVals)
			require.NoError(t, err)
			err = state.Save()
			require.NoError(t, err)

			state = db.StateAtVersion(nil, false)
			defer state.Close()

			val, exists, err := state.Value(nil, nil)
			require.True(t, exists)
			require.NoError(t, err)
			require.Equal(t, M{
				"foo": M{
					"bar":   M{"baz": uint64(123)},
					"slice": test.expectedSlice,
				},
			}, val)
		})
	}
}

func TestDBNode_Delete_NoRange(t *testing.T) {
	t.Run("slice", func(t *testing.T) {
		db := setupVersionedDBTreeWithValue(t, Keypath("data"), fixture1.input)
		defer db.DeleteDB()

		state := db.StateAtVersion(nil, true)

		err := state.Delete(Keypath("data/flox"), nil)
		require.NoError(t, err)

		err = state.Save()
		require.NoError(t, err)

		state = db.StateAtVersion(nil, false)
		state.DebugPrint(debugPrint, true, 0)

		expected := append(
			makeSetKeypathFixtureOutputs(Keypath("data")),
			prefixFixtureOutputs(Keypath("data"), fixture1.output)...,
		)
		expected = removeFixtureOutputsWithPrefix(Keypath("data/flox"), expected...)
		for _, x := range expected {
			fmt.Println("EXPECT ~>", x.keypath)
		}

		iter := state.Iterator(nil, false, 0)
		defer iter.Close()

		i := 0
		for iter.Rewind(); iter.Valid(); iter.Next() {
			require.Equal(t, expected[i].keypath, iter.Node().Keypath())
			i++
		}
	})
}

func TestVersionedDBTree_CopyToMemory(t *testing.T) {
	tests := []struct {
		name    string
		keypath Keypath
	}{
		{"root value", Keypath(nil)},
		{"value", Keypath("flo")},
		{"slice", Keypath("flox")},
		{"map", Keypath("flox").PushIndex(1)},
	}

	t.Run("after .NodeAt", func(t *testing.T) {
		for _, test := range tests {
			test := test
			t.Run(test.name, func(t *testing.T) {
				db := setupVersionedDBTreeWithValue(t, nil, fixture1.input)
				defer db.DeleteDB()

				state := db.StateAtVersion(nil, false)
				defer state.Close()

				copied, err := state.NodeAt(test.keypath, nil).CopyToMemory(nil, nil)
				require.NoError(t, err)

				expected := filterFixtureOutputsWithPrefix(test.keypath, fixture1.output...)
				expected = removeFixtureOutputPrefixes(test.keypath, expected...)

				memnode := copied.(*MemoryNode)
				require.Equal(t, len(expected), len(memnode.keypaths))
				for i := range memnode.keypaths {
					require.Equal(t, expected[i].keypath, memnode.keypaths[i])
				}
			})
		}
	})

	t.Run("without .NodeAt", func(t *testing.T) {
		for _, test := range tests {
			test := test
			t.Run(test.name, func(t *testing.T) {
				db := setupVersionedDBTreeWithValue(t, nil, fixture1.input)
				defer db.DeleteDB()

				state := db.StateAtVersion(nil, false)
				defer state.Close()

				copied, err := state.CopyToMemory(test.keypath, nil)
				require.NoError(t, err)

				expected := filterFixtureOutputsWithPrefix(test.keypath, fixture1.output...)
				expected = removeFixtureOutputPrefixes(test.keypath, expected...)

				memnode := copied.(*MemoryNode)
				require.Equal(t, len(expected), len(memnode.keypaths))
				for i := range memnode.keypaths {
					require.Equal(t, expected[i].keypath, memnode.keypaths[i])
				}
			})
		}
	})

}

func TestDBNode_Iterator(t *testing.T) {
	tests := []struct {
		name        string
		setKeypath  Keypath
		iterKeypath Keypath
		fixture     fixture
	}{
		{"root set, root iter, map value", Keypath(nil), Keypath(nil), fixture1},
		{"root set, root iter, map value 2", Keypath(nil), Keypath(nil), fixture2},
		{"root set, root iter, float value", Keypath(nil), Keypath(nil), fixture5},
		{"root set, root iter, string value", Keypath(nil), Keypath(nil), fixture6},
		{"root set, root iter, bool value", Keypath(nil), Keypath(nil), fixture7},

		{"non-root set, root iter, map value", Keypath("foo/bar"), Keypath(nil), fixture1},
		{"non-root set, root iter, map value 2", Keypath("foo/bar"), Keypath(nil), fixture2},
		{"non-root set, root iter, float value", Keypath("foo/bar"), Keypath(nil), fixture5},
		{"non-root set, root iter, string value", Keypath("foo/bar"), Keypath(nil), fixture6},
		{"non-root set, root iter, bool value", Keypath("foo/bar"), Keypath(nil), fixture7},

		{"root set, non-root iter, map value", Keypath(nil), Keypath("flox"), fixture1},
		{"root set, non-root iter, map value 2", Keypath(nil), Keypath("flox"), fixture2},
		{"root set, non-root iter, float value", Keypath(nil), Keypath("flox"), fixture5},
		{"root set, non-root iter, string value", Keypath(nil), Keypath("flox"), fixture6},
		{"root set, non-root iter, bool value", Keypath(nil), Keypath("flox"), fixture7},

		{"non-root set, non-root iter, map value", Keypath("foo/bar"), Keypath("flox"), fixture1},
		{"non-root set, non-root iter, map value 2", Keypath("foo/bar"), Keypath("flox"), fixture2},
		{"non-root set, non-root iter, float value", Keypath("foo/bar"), Keypath("flox"), fixture5},
		{"non-root set, non-root iter, string value", Keypath("foo/bar"), Keypath("flox"), fixture6},
		{"non-root set, non-root iter, bool value", Keypath("foo/bar"), Keypath("flox"), fixture7},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			db := setupVersionedDBTreeWithValue(t, test.setKeypath, test.fixture.input)
			defer db.DeleteDB()

			state := db.StateAtVersion(nil, false)

			setKeypathOutputs := makeSetKeypathFixtureOutputs(test.setKeypath)
			valueOutputs := prefixFixtureOutputs(test.setKeypath, test.fixture.output)
			expected := append(setKeypathOutputs, valueOutputs...)
			expected = filterFixtureOutputsWithPrefix(test.iterKeypath, expected...)

			iter := state.Iterator(test.iterKeypath, false, 0)
			defer iter.Close()
			var i int
			for iter.Rewind(); iter.Valid(); iter.Next() {
				node := iter.Node()
				require.Equal(t, expected[i].keypath, node.Keypath())
				i++
			}
			require.Equal(t, len(expected), i)

		})
	}
}

func TestDBNode_ReusableIterator(t *testing.T) {
	val := M{
		"aaa": uint64(123),
		"bbb": uint64(123),
		"ccc": M{
			"111": M{
				"a": uint64(1),
				"b": uint64(1),
				"c": uint64(1),
			},
		},
		"ddd": uint64(123),
		"eee": uint64(123),
	}

	db := setupVersionedDBTreeWithValue(t, Keypath("foo"), val)
	defer db.DeleteDB()

	state := db.StateAtVersion(nil, true)
	iter := state.Iterator(Keypath("foo"), false, 0)
	defer iter.Close()

	iter.Rewind()
	require.True(t, iter.Valid())
	require.Equal(t, Keypath("foo"), iter.Node().Keypath())

	iter.Next()
	require.True(t, iter.Valid())
	require.Equal(t, Keypath("foo/aaa"), iter.Node().Keypath())

	iter.Next()
	require.True(t, iter.Valid())
	require.Equal(t, Keypath("foo/bbb"), iter.Node().Keypath())

	iter.Next()
	require.True(t, iter.Valid())
	require.Equal(t, Keypath("foo/ccc"), iter.Node().Keypath())

	{
		reusableIter := iter.Node().Iterator(Keypath("111"), true, 10)
		require.IsType(t, &reusableIterator{}, reusableIter)

		reusableIter.Rewind()
		require.True(t, reusableIter.Valid())
		require.Equal(t, Keypath("foo/ccc/111"), reusableIter.Node().Keypath())

		reusableIter.Next()
		require.True(t, reusableIter.Valid())
		require.Equal(t, Keypath("foo/ccc/111/a"), reusableIter.Node().Keypath())

		reusableIter.Next()
		require.True(t, reusableIter.Valid())
		require.Equal(t, Keypath("foo/ccc/111/b"), reusableIter.Node().Keypath())

		reusableIter.Next()
		require.True(t, reusableIter.Valid())
		require.Equal(t, Keypath("foo/ccc/111/c"), reusableIter.Node().Keypath())

		reusableIter.Next()
		require.False(t, reusableIter.Valid())

		require.True(t, iter.Valid())
		require.Equal(t, Keypath("foo/ccc"), iter.Node().Keypath())

		reusableIter.Close()

		require.Equal(t, []byte("foo/ccc"), iter.(*dbIterator).iter.Item().Key()[33:])

		iter.Next()
		require.True(t, iter.Valid())
		require.Equal(t, Keypath("foo/ccc/111"), iter.Node().Keypath())
	}
}

func TestDBNode_ChildIterator(t *testing.T) {
	tests := []struct {
		name        string
		setKeypath  Keypath
		iterKeypath Keypath
		fixture     fixture
	}{
		{"root set, root iter, map value", Keypath(nil), Keypath(nil), fixture1},
		{"root set, root iter, map value 2", Keypath(nil), Keypath(nil), fixture2},
		{"root set, root iter, float value", Keypath(nil), Keypath(nil), fixture5},
		{"root set, root iter, string value", Keypath(nil), Keypath(nil), fixture6},
		{"root set, root iter, bool value", Keypath(nil), Keypath(nil), fixture7},

		{"non-root set, root iter, map value", Keypath("foo/bar"), Keypath(nil), fixture1},
		{"non-root set, root iter, map value 2", Keypath("foo/bar"), Keypath(nil), fixture2},
		{"non-root set, root iter, float value", Keypath("foo/bar"), Keypath(nil), fixture5},
		{"non-root set, root iter, string value", Keypath("foo/bar"), Keypath(nil), fixture6},
		{"non-root set, root iter, bool value", Keypath("foo/bar"), Keypath(nil), fixture7},

		{"root set, non-root iter, map value", Keypath(nil), Keypath("flox"), fixture1},
		{"root set, non-root iter, map value 2", Keypath(nil), Keypath("flox"), fixture2},
		{"root set, non-root iter, float value", Keypath(nil), Keypath("flox"), fixture5},
		{"root set, non-root iter, string value", Keypath(nil), Keypath("flox"), fixture6},
		{"root set, non-root iter, bool value", Keypath(nil), Keypath("flox"), fixture7},

		{"non-root set, non-root iter, map value", Keypath("foo/bar"), Keypath("flox"), fixture1},
		{"non-root set, non-root iter, map value 2", Keypath("foo/bar"), Keypath("flox"), fixture2},
		{"non-root set, non-root iter, float value", Keypath("foo/bar"), Keypath("flox"), fixture5},
		{"non-root set, non-root iter, string value", Keypath("foo/bar"), Keypath("flox"), fixture6},
		{"non-root set, non-root iter, bool value", Keypath("foo/bar"), Keypath("flox"), fixture7},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			db := setupVersionedDBTreeWithValue(t, test.setKeypath, test.fixture.input)
			defer db.DeleteDB()

			state := db.StateAtVersion(nil, false)

			prefixOutputs := makeSetKeypathFixtureOutputs(test.setKeypath)
			valueOutputs := combineFixtureOutputs(test.setKeypath, test.fixture)
			expected := append(prefixOutputs, valueOutputs...)
			expected = filterFixtureOutputsToDirectDescendantsOf(test.iterKeypath, expected...)

			iter := state.ChildIterator(test.iterKeypath, false, 0)
			defer iter.Close()
			var i int
			for iter.Rewind(); iter.Valid(); iter.Next() {
				node := iter.Node()
				require.Equal(t, expected[i].keypath, node.Keypath())
				i++
			}
			require.Equal(t, len(expected), i)

		})
	}
}

func TestDBNode_ReusableChildIterator(t *testing.T) {
	val := M{
		"aaa": uint64(123),
		"bbb": uint64(123),
		"ccc": M{
			"111": M{
				"a": uint64(1),
				"b": uint64(1),
				"c": uint64(1),
			},
		},
		"ddd": uint64(123),
		"eee": uint64(123),
	}

	db := setupVersionedDBTreeWithValue(t, Keypath("foo"), val)
	defer db.DeleteDB()

	state := db.StateAtVersion(nil, true)
	iter := state.ChildIterator(Keypath("foo"), false, 0)
	defer iter.Close()

	iter.Rewind()
	require.True(t, iter.Valid())
	require.Equal(t, Keypath("foo/aaa"), iter.Node().Keypath())

	iter.Next()
	require.True(t, iter.Valid())
	require.Equal(t, Keypath("foo/bbb"), iter.Node().Keypath())

	iter.Next()
	require.True(t, iter.Valid())
	require.Equal(t, Keypath("foo/ccc"), iter.Node().Keypath())

	{
		reusableIter := iter.Node().Iterator(Keypath("111"), true, 10)
		require.IsType(t, &reusableIterator{}, reusableIter)

		reusableIter.Rewind()
		require.True(t, reusableIter.Valid())
		require.Equal(t, Keypath("foo/ccc/111"), reusableIter.Node().Keypath())

		reusableIter.Next()
		require.True(t, reusableIter.Valid())
		require.Equal(t, Keypath("foo/ccc/111/a"), reusableIter.Node().Keypath())

		reusableIter.Next()
		require.True(t, reusableIter.Valid())
		require.Equal(t, Keypath("foo/ccc/111/b"), reusableIter.Node().Keypath())

		reusableIter.Next()
		require.True(t, reusableIter.Valid())
		require.Equal(t, Keypath("foo/ccc/111/c"), reusableIter.Node().Keypath())

		reusableIter.Next()
		require.False(t, reusableIter.Valid())

		require.True(t, iter.Valid())
		require.Equal(t, Keypath("foo/ccc"), iter.Node().Keypath())

		reusableIter.Close()

		// require.Equal(t, []byte("foo/ccc"), iter.(*dbChildIterator).iter.Item().Key()[33:])

		iter.Next()
		require.True(t, iter.Valid())
		require.Equal(t, Keypath("foo/ddd"), iter.Node().Keypath())
	}
}

func TestDBNode_DepthFirstIterator(t *testing.T) {
	tests := []struct {
		name        string
		setKeypath  Keypath
		iterKeypath Keypath
		fixture     fixture
	}{
		{"root set, root iter, map value", Keypath(nil), Keypath(nil), fixture1},
		{"root set, root iter, map value 2", Keypath(nil), Keypath(nil), fixture2},
		{"root set, root iter, float value", Keypath(nil), Keypath(nil), fixture5},
		{"root set, root iter, string value", Keypath(nil), Keypath(nil), fixture6},
		{"root set, root iter, bool value", Keypath(nil), Keypath(nil), fixture7},

		{"non-root set, root iter, map value", Keypath("foo/bar"), Keypath(nil), fixture1},
		{"non-root set, root iter, map value 2", Keypath("foo/bar"), Keypath(nil), fixture2},
		{"non-root set, root iter, float value", Keypath("foo/bar"), Keypath(nil), fixture5},
		{"non-root set, root iter, string value", Keypath("foo/bar"), Keypath(nil), fixture6},
		{"non-root set, root iter, bool value", Keypath("foo/bar"), Keypath(nil), fixture7},

		{"root set, non-root iter, map value", Keypath(nil), Keypath("flox"), fixture1},
		{"root set, non-root iter, map value 2", Keypath(nil), Keypath("eee"), fixture2},
		{"root set, non-root iter, float value", Keypath(nil), Keypath("flox"), fixture5},
		{"root set, non-root iter, string value", Keypath(nil), Keypath("flox"), fixture6},
		{"root set, non-root iter, bool value", Keypath(nil), Keypath("flox"), fixture7},

		{"non-root set, non-root iter, map value", Keypath("foo/bar"), Keypath("foo/bar/flox"), fixture1},
		{"non-root set, non-root iter, map value 2", Keypath("foo/bar"), Keypath("foo/bar/eee"), fixture2},
		{"non-root set, non-root iter, float value", Keypath("foo/bar"), Keypath("foo/bar"), fixture5},
		{"non-root set, non-root iter, string value", Keypath("foo/bar"), Keypath("foo/bar"), fixture6},
		{"non-root set, non-root iter, bool value", Keypath("foo/bar"), Keypath("foo/bar"), fixture7},
		{"non-root set, non-root iter, nonexistent value", Keypath("foo/bar"), Keypath("foo/bar/asdf"), fixture7},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			db := setupVersionedDBTreeWithValue(t, test.setKeypath, test.fixture.input)
			defer db.DeleteDB()

			state := db.StateAtVersion(nil, false)

			prefixOutputs := makeSetKeypathFixtureOutputs(test.setKeypath)
			valueOutputs := combineFixtureOutputs(test.setKeypath, test.fixture)
			expected := append(prefixOutputs, valueOutputs...)
			expected = filterFixtureOutputsWithPrefix(test.iterKeypath, expected...)
			expected = reverseFixtureOutputs(expected...)

			iter := state.DepthFirstIterator(test.iterKeypath, false, 0)
			defer iter.Close()
			var i int
			for iter.Rewind(); iter.Valid(); iter.Next() {
				node := iter.Node()
				require.Equal(t, expected[i].keypath, node.Keypath())
				i++
			}
			require.Equal(t, len(expected), i)

		})
	}
}

func TestVersionedDBTree_CopyVersion(t *testing.T) {
	i := rand.Int()
	tree, err := NewVersionedDBTree(fmt.Sprintf("/tmp/tree-badger-test-%v", i))
	require.NoError(t, err)
	defer tree.DeleteDB()

	srcVersion := types.RandomID()
	dstVersion := types.RandomID()

	err = tree.Update(&srcVersion, func(tx *DBNode) error {
		err := tx.Set(nil, nil, fixture1.input)
		require.NoError(t, err)
		return nil
	})
	require.NoError(t, err)

	err = tree.CopyVersion(dstVersion, srcVersion)
	require.NoError(t, err)

	srcVal, exists, err := tree.StateAtVersion(&srcVersion, false).Value(nil, nil)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, srcVal, fixture1.input)

	dstVal, exists, err := tree.StateAtVersion(&dstVersion, false).Value(nil, nil)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, dstVal, fixture1.input)

	var count int
	err = tree.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		iter := txn.NewIterator(opts)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			count++
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, len(fixture1.output)*2, count)
}

// func TestVersionedDBTree_CopyToMemory(t *testing.T) {
//  t.Parallel()

//  i := rand.Int()
//  tree, err := NewVersionedDBTree(fmt.Sprintf("/tmp/tree-badger-test-%v", i))
//  require.NoError(t, err)
//  defer tree.DeleteDB()

//  err = tree.Update(func(tx *DBNode) error {
//      _, err := tx.Set(nil, nil, testVal1)
//      require.NoError(t, err)
//      return nil
//  })
//  require.NoError(t, err)

//  expected := []struct {
//      keypath  Keypath
//      nodeType NodeType
//      val      interface{}
//  }{
//      {Keypath(""), NodeTypeMap, testVal1},
//      {Keypath("hello"), NodeTypeMap, testVal1["hello"]},
//      {Keypath("hello/xyzzy"), NodeTypeValue, testVal1["hello"].(M)["xyzzy"]},
//      {Keypath("flox"), NodeTypeSlice, testVal1["flox"]},
//      {Keypath("flox").PushIndex(0), NodeTypeValue, testVal1["flox"].(S)[0]},
//      {Keypath("flox").PushIndex(1), NodeTypeMap, testVal1["flox"].(S)[1]},
//      {Keypath("flox").PushIndex(1).Push(Keypath("yup")), NodeTypeValue, testVal1["flox"].(S)[1].(M)["yup"]},
//      {Keypath("flox").PushIndex(1).Push(Keypath("hey")), NodeTypeValue, testVal1["flox"].(S)[1].(M)["hey"]},
//      {Keypath("flox").PushIndex(2), NodeTypeValue, testVal1["flox"].(S)[2]},
//  }

//  expectedValues := map[string]interface{}{
//      "":                                   testVal1,
//      "hello":                              testVal1["hello"],
//      "hello/xyzzy":                        testVal1["hello"].(M)["xyzzy"],
//      "flox":                               testVal1["flox"],
//      string(Keypath("flox").PushIndex(0)): testVal1["flox"].(S)[0],
//      string(Keypath("flox").PushIndex(1)): testVal1["flox"].(S)[1],
//      string(Keypath("flox").PushIndex(1).Push(Keypath("yup"))): testVal1["flox"].(S)[1].(M)["yup"],
//      string(Keypath("flox").PushIndex(1).Push(Keypath("hey"))): testVal1["flox"].(S)[1].(M)["hey"],
//      string(Keypath("flox").PushIndex(2)):                      testVal1["flox"].(S)[2],
//  }

//  sort.Slice(expectedKeypaths, func(i, j int) bool { return bytes.Compare(expectedKeypaths[i], expectedKeypaths[j]) < 0 })

//  copied, err := node.CopyToMemory(nil)
//  require.NoError(t, err)

//  memnode := copied.(*MemoryNode)
//  for i := range memnode.keypaths {
//      require.Equal(t, expectedKeypaths[i], memnode.keypaths[i])
//  }
// }

//func TestVersionedDBTree_encodeGoValue(t *testing.T) {
//    t.Parallel()
//
//    cases := []struct {
//        input    interface{}
//        expected []byte
//    }{
//        {"asdf", []byte("vsasdf")},
//        {float64(321.23), []byte("vf")},
//    }
//
//    encodeGoValue()
//}

//func debugPrint(t *testing.T, tree *DBNode) {
//    keypaths, values, err := tree.Contents(nil, nil)
//    require.NoError(t, err)
//
//    fmt.Println("KEYPATHS:")
//    for i, kp := range keypaths {
//        fmt.Println("  -", kp, ":", values[i])
//    }
//
//    v, _, err := tree.Value(nil, nil)
//    require.NoError(t, err)
//
//    fmt.Println(prettyJSON(v))
//}

func (t *VersionedDBTree) View(v *types.ID, fn func(*DBNode) error) error {
	state := t.StateAtVersion(v, false)
	defer state.Close()
	return fn(state)
}

func (t *VersionedDBTree) Update(v *types.ID, fn func(*DBNode) error) error {
	state := t.StateAtVersion(v, true)
	defer state.Close()

	err := fn(state)
	if err != nil {
		return err
	}

	err = state.Save()
	if err != nil {
		return err
	}
	return nil
}

func setupDBTree(t *testing.T) *DBTree {
	t.Helper()

	i := rand.Int()
	db, err := NewDBTree(fmt.Sprintf("/tmp/tree-badger-test-%v", i))
	require.NoError(t, err)
	return db
}

func setupDBTreeWithValue(t *testing.T, keypath Keypath, val interface{}) *DBTree {
	t.Helper()

	i := rand.Int()

	db, err := NewDBTree(fmt.Sprintf("/tmp/tree-badger-test-%v", i))
	require.NoError(t, err)

	state := db.State(true)
	defer state.Save()

	err = state.Set(keypath, nil, val)
	require.NoError(t, err)

	return db
}

func setupVersionedDBTreeWithValue(t *testing.T, keypath Keypath, val interface{}) *VersionedDBTree {
	t.Helper()

	i := rand.Int()

	db, err := NewVersionedDBTree(fmt.Sprintf("/tmp/tree-badger-test-%v", i))
	require.NoError(t, err)

	state := db.StateAtVersion(nil, true)
	defer state.Save()

	err = state.Set(keypath, nil, val)
	require.NoError(t, err)

	return db
}
