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

package config

import (
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestLoadCliConfigFromFile(t *testing.T) {
	logger.SetLevel(logger.ErrorLevel)
	tests := []struct {
		name     string
		content  string
		expected *CliConfig
	}{
		{
			name:    "CorrectFileContent",
			content: fakeCliFileContent(),
			expected: &CliConfig{
				port:     5,
				password: "password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.name + "_cliconfig.conf"
			ioutil.WriteFile(filename, []byte(tt.content), 0644)
			defer os.Remove(filename)
			configContent := LoadCliConfigFromFile(filename)
			assert.Equal(t, tt.expected, configContent)
		})
	}

}

func fakeCliFileContent() string {
	return `
	port: 5,
	password: "password"
	`
}
