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

package conf

import (
	"encoding/json"
	"github.com/cchamplin/deployd/log"
	"io/ioutil"
	"path/filepath"
)

type DefaultConf struct {
	path string
}

func (e *DefaultConf) Init(path string) {
	e.path = path
}

func (e *DefaultConf) GetPath() string {
	return e.path
}

func (e *DefaultConf) GetValue(key string) map[string]interface{} {
	var file = key
	if key == e.path+"/config" {
		file = filepath.Clean(e.path + "/deployd.json")
	}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Error.Printf("Failed to read file %s: %v", file, err)
		return nil
	}
	var output map[string]interface{}
	if err = json.Unmarshal([]byte(data), &output); err != nil {
		log.Error.Printf("Failed to parse file %s: %v", data, err)
		return nil
	}
	return output

}

func (e *DefaultConf) GetString(key string) string {
	var file = key
	if key == e.path+"/config" {
		file = filepath.Clean(e.path + "/deployd.json")
	}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Error.Printf("Failed to read file %s: %v", file, err)
		return ""
	}
	return string(data)
}

func (e *DefaultConf) GetValues(key string) map[string]interface{} {
	var file = key
	if key == e.path+"/config" {
		file = filepath.Clean(e.path + "/deployd.json")
	}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Error.Printf("Failed to read file %s: %v", file, err)
		return nil
	}
	var output map[string]interface{}
	if err = json.Unmarshal([]byte(data), &output); err != nil {
		log.Error.Printf("Failed to parse file %s: %v", data, err)
		return nil
	}
	return output
}
