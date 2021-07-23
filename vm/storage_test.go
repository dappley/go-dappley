// Copyright (C) 2018 go-nebulas authors
//
// This file is part of the go-nebulas library.
//
// the go-nebulas library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-nebulas library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-nebulas library.  If not, see <http://www.gnu.org/licenses/>.
//

package vm

import (
	"testing"

	errorValues "github.com/dappley/go-dappley/errors"
	"github.com/stretchr/testify/assert"
)

func TestStorageKeyPattern(t *testing.T) {
	tests := []struct {
		key       string
		want      error
		domainKey string
		itemKey   string
	}{
		{"", errorValues.ErrInvalidStorageKey, "", ""},
		{"string", errorValues.ErrInvalidStorageKey, "", ""},
		{"_map[key]", errorValues.ErrInvalidStorageKey, "", ""},
		{"@map[key", errorValues.ErrInvalidStorageKey, "", ""},
		{"@mapkey]", errorValues.ErrInvalidStorageKey, "", ""},
		{"@123[key]", errorValues.ErrInvalidStorageKey, "", ""},
		{"@|abc[key]", errorValues.ErrInvalidStorageKey, "", ""},
		{"@abc123-[key]", errorValues.ErrInvalidStorageKey, "", ""},
		{"@abc$[key]", errorValues.ErrInvalidStorageKey, "", ""},
		{"@a[key]", errorValues.ErrInvalidStorageKey, "", ""},
		{"@$[key]", errorValues.ErrInvalidStorageKey, "", ""},
		{"@_[key]", errorValues.ErrInvalidStorageKey, "", ""},
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
				err = errorValues.ErrInvalidStorageKey
			} else {
				domainKey, itemKey = matches[0][1], matches[0][2]
			}

			assert.Equal(t, tt.want, err)
			assert.Equal(t, tt.domainKey, domainKey)
			assert.Equal(t, tt.itemKey, itemKey)
		})
	}
}
