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
	f := Err[bool](fmt.Errorf("foo"))
	require.True(t, f.IsReady())
	require.False(t, f.Result().IsOk())
	require.True(t, f.Result().IsErr())
	assert.Error(t, f.Result().Error)
}

func TestMap(t *testing.T) {
	f := Map(Ok(1), func(r Result[int]) Result[float64] {
		return Result[float64]{
			Value: float64(r.Value),
		}
	})
	require.True(t, f.IsReady())
	assert.Equal(t, 1.0, f.Result().Value)
}

func TestMapOk(t *testing.T) {
	f := MapOk(Ok(1), func(v int) float64 {
		return float64(v)
	})
	require.True(t, f.IsReady())
	assert.Equal(t, 1.0, f.Result().Value)
}

func TestThen(t *testing.T) {
	f := Then(Ok(1), func(r Result[int]) Future[float64] {
		return Ok(float64(r.Value))
	})
	require.True(t, f.IsReady())
	assert.Equal(t, 1.0, f.Result().Value)
}

func TestPoll(t *testing.T) {
	v := 0

	f := New(func() (Result[int], bool) {
		return Result[int]{Value: v}, v != 0
	})
	f = Map(f, func(r Result[int]) Result[int] {
		return Result[int]{Value: r.Value + 1}
	})
	f = MapOk(f, func(v int) int {
		return v + 1
	})
	f = Then(f, func(r Result[int]) Future[int] {
		return Ok(r.Value + 1)
	})

	f.Poll()
	require.False(t, f.IsReady())

	v = 1

	f.Poll()
	require.True(t, f.IsReady())
	assert.Equal(t, 4, f.Result().Value)
}

func TestJoin(t *testing.T) {
	t.Run("Ready", func(t *testing.T) {
		f := Join(Ok(1), Ok(2))

		require.True(t, f.IsReady())
		assert.Equal(t, []int{1, 2}, f.Result().Value)
	})

	t.Run("NotReady", func(t *testing.T) {
		ready := false

		f := Join(New(func() (Result[int], bool) {
			return Result[int]{Value: 1}, ready
		}), Ok(2))

		require.False(t, f.IsReady())

		f.Poll()
		require.False(t, f.IsReady())

		ready = true
		f.Poll()

		require.True(t, f.IsReady())
		assert.Equal(t, []int{1, 2}, f.Result().Value)
	})

	t.Run("NotReadyError", func(t *testing.T) {
		ready := false

		f := Join(New(func() (Result[int], bool) {
			return Result[int]{Error: fmt.Errorf("foo")}, ready
		}), Ok(2))

		require.False(t, f.IsReady())

		f.Poll()
		require.False(t, f.IsReady())

		ready = true
		f.Poll()

		require.True(t, f.IsReady())
		assert.True(t, f.Result().IsErr())
	})

	t.Run("Error", func(t *testing.T) {
		f := Join(Err[int](fmt.Errorf("foo")), Ok(2))

		require.True(t, f.IsReady())
		assert.True(t, f.Result().IsErr())
	})
}

func TestAfter(t *testing.T) {
	t.Run("Ready", func(t *testing.T) {
		f := After(Ok(1), Ok(2))

		require.True(t, f.IsReady())
		assert.True(t, f.Result().IsOk())
	})

	t.Run("NotReady", func(t *testing.T) {
		ready := false

		f := After(New(func() (Result[int], bool) {
			return Result[int]{Value: 1}, ready
		}), Ok(2))

		require.False(t, f.IsReady())

		f.Poll()
		require.False(t, f.IsReady())

		ready = true
		f.Poll()

		require.True(t, f.IsReady())
		assert.True(t, f.Result().IsOk())
	})

	t.Run("NotReadyError", func(t *testing.T) {
		ready := false

		f := After(New(func() (Result[int], bool) {
			return Result[int]{Error: fmt.Errorf("foo")}, ready
		}), Ok(2))

		require.False(t, f.IsReady())

		f.Poll()
		require.False(t, f.IsReady())

		ready = true
		f.Poll()

		require.True(t, f.IsReady())
		assert.True(t, f.Result().IsErr())
	})

	t.Run("Error", func(t *testing.T) {
		f := After(Err[int](fmt.Errorf("foo")), Ok(2))

		require.True(t, f.IsReady())
		assert.True(t, f.Result().IsErr())
	})
}
