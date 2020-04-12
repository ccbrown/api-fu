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
	m.Append("foo", "bar")
	m.Append("foo2", "bar2")
	assert.Len(t, m.Items(), 2)

	buf, err := json.Marshal(m)
	assert.NoError(t, err)
	assert.Equal(t, `{"foo":"bar","foo2":"bar2"}`, string(buf))
}

func BenchmarkOrderedMapEncoding(b *testing.B) {
	m := NewOrderedMap()
	for i := 0; i < 2000; i++ {
		m.Append("foo"+strconv.Itoa(i), "bar")
		m2 := NewOrderedMap()
		for j := 0; j < 10; j++ {
			m2.Append("foo"+strconv.Itoa(j), "bar")
			m3 := NewOrderedMap()
			for k := 0; k < 10; k++ {
				m3.Append("foo"+strconv.Itoa(k), "bar")
			}
			m2.Append("m"+strconv.Itoa(j), m3)
		}
		m.Append("m"+strconv.Itoa(i), m2)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sink, _ = jsoniter.ConfigFastest.Marshal(m)
	}
}
