package config

import (
	"testing"
	"io/ioutil"
	"os"
	"github.com/stretchr/testify/assert"
	"github.com/sirupsen/logrus"
)

func TestLoadConfigFromFile(t *testing.T){
	logrus.SetLevel(logrus.ErrorLevel)
	tests := []struct{
		name 		string
		content 	string
		expected	*Config
	}{
		{
			name: 		"CorrectFileContent",
			content: 	fakeFileContent(),
			expected:	&Config{
				DynastyConfig{
					producers: 	[]string{
						"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
						"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
					},
				},
				ConsensusConfig{
					minerAddr: "1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				},
				NodeConfig{
					port: 5,
					seed: "/ip4/127.0.0.1/tcp/34836/ipfs/QmPtahvwSvnSHymR5HZiSTpkm9xHymx9QLNkUjJ7mfygGs",
				},
			},
		},
		{
			name: 		"EmptySeed",
			content: 	noSeedContent(),
			expected:	&Config{
				DynastyConfig{
					producers: 	[]string{
						"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
						"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
					},
				},
				ConsensusConfig{
					minerAddr: "1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
				},
				NodeConfig{
					port: 5,
					seed: "",
				},
			},
		},
		{
			name: 		"WrongFileContent",
			content: 	"WrongFileContent",
			expected:	nil,
		},
		{
			name: 		"EmptyFile",
			content: 	"",
			expected:	&Config{
				DynastyConfig{
					producers: 	[]string(nil),
				},
				ConsensusConfig{
					minerAddr: "",
				},
				NodeConfig{
					port: 0,
				},
			},
		},
	}

	for _,tt := range tests {
		t.Run(tt.name, func(t *testing.T){
			filename := tt.name + "_config.conf"
			ioutil.WriteFile(filename, []byte(tt.content), 0644)
			defer os.Remove(filename)
			configContent := LoadConfigFromFile(filename)
			assert.Equal(t, tt.expected, configContent)
		})
	}

}

func fakeFileContent() string{
	return `
	dynastyConfig{
		producers: [
			"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
			"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"
		]
	}
	consensusConfig{
		minerAddr: "1MeSBgufmzwpiJNLemUe1emxAussBnz7a7"
	}
	nodeConfig{
		port: 5
		seed: "/ip4/127.0.0.1/tcp/34836/ipfs/QmPtahvwSvnSHymR5HZiSTpkm9xHymx9QLNkUjJ7mfygGs"
	}`
}

func noSeedContent() string{
	return `
	dynastyConfig{
		producers: [
			"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
			"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"
		]
	}
	consensusConfig{
		minerAddr: "1MeSBgufmzwpiJNLemUe1emxAussBnz7a7"
	}
	nodeConfig{
		port: 5
	}`
}
