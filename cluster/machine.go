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

package cluster

import (
	"encoding/json"
	"github.com/cchamplin/deployd/deployment"
	"github.com/cchamplin/deployd/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Machine struct {
	Id       string   `json:"id"`
	Endpoint string   `json:"endpoint"`
	Tags     []string `json:"tags"`
}

type Machines []*Machine

func (m *Machine) Serialize() string {
	data, _ := json.Marshal(m)
	return string(data)
}

func DeserializeMachine(s string) *Machine {
	var result Machine
	err := json.Unmarshal([]byte(s), &result)
	if err != nil {
		log.Error.Printf("Failed to parse json machine json: %v \"%s\"", err, s)
		return nil
	}
	return &result
}

func (m *Machine) TryDeploy(d deployment.Deployment) bool {

	return true
}

func LocalMachine(endpoint string, tags []string) *Machine {
	m := new(Machine)
	m.Id = MachineID("/")
	m.Endpoint = endpoint
	m.Tags = tags
	return m
}

func MachineID(root string) string {
	fullPath := filepath.Join(root, "/etc/machine-id")
	id, err := ioutil.ReadFile(fullPath)
	if err != nil {
		host, err := os.Hostname()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(host)
	}
	return strings.TrimSpace(string(id))
}
