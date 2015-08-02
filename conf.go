// The MIT License (MIT)
//
// Copyright (c) 2015 Caleb Champlin (caleb.champlin@gmail.com)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"./log"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
)

type ServerConfiguration struct {
	Addr          string   `json:"bind-addr"`
	Port          int      `json:"bind-port"`
	AllowedTags   []string `json:"allowed-tags"`
	AllowUntagged bool     `json:"allow-untagged"`
}

// Load the configuration file
func LoadConfiguration(configDir string) *ServerConfiguration {
	var result ServerConfiguration
	// Should we utilize a .conf prefix for configuration file?
	file := filepath.Clean(configDir + "/deployd.json")
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Error.Printf("Failed to read file %s: %v", file, err)
		return nil
	}

	err = json.Unmarshal([]byte(data), &result)
	if err != nil {
		log.Error.Printf("Failed to parse json file %s: %v", file, err)
		return nil
	}
	return &result
}
