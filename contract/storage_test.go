package sc

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestStorageKeyPattern(t *testing.T) {
	tests := []struct {
		key       string
		want      error
		domainKey string
		itemKey   string
	}{
		{"", ErrInvalidStorageKey, "", ""},
		{"string", ErrInvalidStorageKey, "", ""},
		{"_map[key]", ErrInvalidStorageKey, "", ""},
		{"@map[key", ErrInvalidStorageKey, "", ""},
		{"@mapkey]", ErrInvalidStorageKey, "", ""},
		{"@123[key]", ErrInvalidStorageKey, "", ""},
		{"@|abc[key]", ErrInvalidStorageKey, "", ""},
		{"@abc123-[key]", ErrInvalidStorageKey, "", ""},
		{"@abc$[key]", ErrInvalidStorageKey, "", ""},
		{"@a[key]", ErrInvalidStorageKey, "", ""},
		{"@$[key]", ErrInvalidStorageKey, "", ""},
		{"@_[key]", ErrInvalidStorageKey, "", ""},
		{"@$abc[]", nil, "$abc", ""},
		{"@$abc[key]", nil, "$abc", "key"},
		{"@abc[key]", nil, "abc", "key"},
		{"@abc12_12[key]", nil, "abc12_12", "key"},
		{"@abc[[key]]", nil, "abc", "[key]"},
		{"@abc[zzz[key]yyy?]", nil, "abc", "zzz[key]yyy?"},
		{"@abc[[key]yyy]", nil, "abc", "[key]yyy"},
		{"@abc[zzz[key]]", nil, "abc", "zzz[key]"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			var (
				domainKey string
				itemKey   string
				err       error
			)

			matches := StorageKeyPattern.FindAllStringSubmatch(tt.key, -1)
			if matches == nil {
				err = ErrInvalidStorageKey
			} else {
				domainKey, itemKey = matches[0][1], matches[0][2]
			}

			assert.Equal(t, tt.want, err)
			assert.Equal(t, tt.domainKey, domainKey)
			assert.Equal(t, tt.itemKey, itemKey)
		})
	}
}
