package diffmatchpatch

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIndexConversion(t *testing.T) {
	n := runeMax - (runeSkipEnd - runeSkipStart)
	indexes := make([]index, n)
	for i := 0; i < n; i++ {
		indexes[i] = index(i)
	}
	indexes2 := stringToIndex(indexesToString(indexes))
	assert.EqualValues(t, indexes, indexes2)
}
