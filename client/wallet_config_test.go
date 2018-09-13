// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package client

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestLoadCliConfigFromFile(t *testing.T) {

	tests := []struct {
		name     string
		content  string
		expected *WalletConfig
	}{
		{
			name:    "CorrectFileContent",
			content: fakeCliFileContent(),
			expected: &WalletConfig{
				filePath: "wallet.dat",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.name + "_walletConfig.conf"
			ioutil.WriteFile(filename, []byte(tt.content), 0644)
			defer os.Remove(filename)
			configContent := LoadWalletConfigFromFile(filename)
			assert.Equal(t, tt.expected, configContent)
		})
	}

}

func fakeCliFileContent() string {
	return `
	filePath: "wallet.dat"
	`
}
