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
	"fmt"
	"io/ioutil"
	"path/filepath"

	"../conf"
	"../log"
)

type Cluster struct {
	Machines      Machines `json:"machines"`
	Backend       Backend
	ClusterConfig map[string]interface{}
}

func (c *Cluster) ParseConfig(configDirectory string) {
	file := filepath.Clean(configDirectory + "/cluster.json")
	log.Info.Printf("Loading cluster configuration from %s ", file)
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Warning.Printf("Failed to read file %s: %v", file, err)
		return
	}
	if err = json.Unmarshal(data, &c.ClusterConfig); err != nil {
		log.Error.Printf("Failed to parse file %s: %v", file, err)
	}
}

func (c *Cluster) ParseConfigFromBackend(configBackend conf.ConfigurationBackend) {
	log.Info.Printf("Loading cluster configuration from backend")

	data := configBackend.GetString(fmt.Sprintf("%s/cluster", configBackend.GetPath()))

	if err := json.Unmarshal([]byte(data), &c.ClusterConfig); err != nil {
		log.Error.Printf("Failed to parse configuration: %v", err)
	}
}

func (c *Cluster) Init(backend Backend, configDirectory string) {
	c.Backend = backend
	c.ParseConfig(configDirectory)
}

func (c *Cluster) InitFromConfig(backend Backend, config conf.ServerConfiguration) {
	c.Backend = backend
	c.ParseConfigFromBackend(config.Backend)
}

func (c *Cluster) AddMachine(machine *Machine) {
	c.Machines = append(c.Machines, machine)
}

func (c *Cluster) RemoveMachine(machine *Machine) {

}

func (c *Cluster) GetMachine(id string) *Machine {
	for _, m := range c.Machines {
		if m.Id == id {
			return m
		}
	}
	return nil
}

func (c *Cluster) List() *Machines {
	return &c.Machines
}
