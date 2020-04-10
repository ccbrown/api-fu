package future

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOk(t *testing.T) {
	f := Ok(1)
	require.True(t, f.IsReady())
	require.True(t, f.Result().IsOk())
	require.False(t, f.Result().IsErr())
	assert.Equal(t, 1, f.Result().Value)
}

func TestErr(t *testing.T) {
	f := Err(fmt.Errorf("foo"))
	require.True(t, f.IsReady())
	require.False(t, f.Result().IsOk())
	require.True(t, f.Result().IsErr())
	assert.Error(t, f.Result().Error)
}

func TestMap(t *testing.T) {
	f := Ok(1).Map(func(r Result) Result {
		return Result{
			Value: float64(r.Value.(int)),
		}
	})
	require.True(t, f.IsReady())
	assert.Equal(t, 1.0, f.Result().Value)
}

func TestMapOk(t *testing.T) {
	f := Ok(1).MapOk(func(v interface{}) interface{} {
		return float64(v.(int))
	})
	require.True(t, f.IsReady())
	assert.Equal(t, 1.0, f.Result().Value)
}

func TestThen(t *testing.T) {
	f := Ok(1).Then(func(r Result) Future {
		return Ok(float64(r.Value.(int)))
	})
	require.True(t, f.IsReady())
	assert.Equal(t, 1.0, f.Result().Value)
}

func TestPoll(t *testing.T) {
	v := 0

	f := New(func() (Result, bool) {
		return Result{Value: v}, v != 0
	})
	f = f.Map(func(r Result) Result {
		return Result{Value: r.Value.(int) + 1}
	})
	f = f.Then(func(r Result) Future {
		return Ok(r.Value.(int) + 1)
	})

	f.Poll()
	if f.IsReady() {
		t.Fatalf("expected future to not be ready")
	}

	v = 1

	f.Poll()
	require.True(t, f.IsReady())
	assert.Equal(t, 3, f.Result().Value)
}

func TestJoin(t *testing.T) {
	t.Run("Ready", func(t *testing.T) {
		f := Join(Ok(1), Ok(2))

		require.True(t, f.IsReady())
		assert.Equal(t, []interface{}{1, 2}, f.Result().Value)
	})

	t.Run("NotReady", func(t *testing.T) {
		ready := false

		f := Join(New(func() (Result, bool) {
			return Result{Value: 1}, ready
		}), Ok(2))

		require.False(t, f.IsReady())

		ready = true
		f.Poll()

		require.True(t, f.IsReady())
		assert.Equal(t, []interface{}{1, 2}, f.Result().Value)
	})

	t.Run("Error", func(t *testing.T) {
		f := Join(Err(fmt.Errorf("foo")), Ok(2))

		require.True(t, f.IsReady())
		assert.True(t, f.Result().IsErr())
	})
}
