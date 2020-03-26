package executor

import (
	"encoding/json"
	"strconv"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func TestOrderedMapEncoding(t *testing.T) {
	m := NewOrderedMap()
	m.Set("foo", "bar")
	m.Set("foo2", "bar2")
	buf, err := json.Marshal(m)
	assert.NoError(t, err)
	assert.Equal(t, `{"foo":"bar","foo2":"bar2"}`, string(buf))
}

func BenchmarkOrderedMapEncoding(b *testing.B) {
	m := NewOrderedMap()
	for i := 0; i < 2000; i++ {
		m.Set("foo"+strconv.Itoa(i), "bar")
		m2 := NewOrderedMap()
		for j := 0; j < 10; j++ {
			m2.Set("foo"+strconv.Itoa(j), "bar")
			m3 := NewOrderedMap()
			for k := 0; k < 10; k++ {
				m3.Set("foo"+strconv.Itoa(k), "bar")
			}
			m2.Set("m"+strconv.Itoa(j), m3)
		}
		m.Set("m"+strconv.Itoa(i), m2)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sink, _ = jsoniter.ConfigFastest.Marshal(m)
	}
}
