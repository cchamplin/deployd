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
	"fmt"
	"time"

	"../../log"
	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

type EtcdConf struct {
	etcdConfig client.Config
	etcdClient client.Client
	kapi       client.KeysAPI
	path       string
}

func (e *EtcdConf) Init(endpoint string, path string) {
	e.path = path
	var endpoints []string
	endpoints = append(endpoints, fmt.Sprintf("http://%s", endpoint))

	e.etcdConfig = client.Config{
		Endpoints:               endpoints,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second * 5,
	}

	// Initialize etcd client
	c, err := client.New(e.etcdConfig)
	e.etcdClient = c
	if err != nil {
		log.Error.Printf("Failed to initialize etcd client: %v", err)
		return
	}

	// Create a keys api
	// Are we okay to use this instances for the lifetime
	// of the application? What happens if the etcd
	// instance we are connecting dies?
	e.kapi = client.NewKeysAPI(e.etcdClient)

}

func (e *EtcdConf) GetPath() string {
	return e.path
}

func (e *EtcdConf) GetValue(key string) map[string]interface{} {
	result, err := e.kapi.Get(context.Background(), key, nil)
	if err != nil {
		log.Warning.Printf("Could not load key value: %s", key)
		return nil
	}
	// Return the first value if it's a list
	if result.Node.Nodes != nil && len(result.Node.Nodes) > 1 {
		nodes := result.Node.Nodes
		for _, node := range nodes {
			var output map[string]interface{}
			if err = json.Unmarshal([]byte(node.Value), &output); err != nil {
				log.Error.Printf("Failed to parse file %s: %v", node.Value, err)
				return nil
			}
			return output
		}
	} else {
		var output map[string]interface{}
		if err = json.Unmarshal([]byte(result.Node.Value), &output); err != nil {
			log.Error.Printf("Failed to parse file %s: %v", result.Node.Value, err)
			return nil
		}
		return output
	}
	return nil
}

func (e *EtcdConf) GetString(key string) string {
	result, err := e.kapi.Get(context.Background(), key, nil)
	if err != nil {
		log.Warning.Printf("Could not load key value: %s", key)
		return ""
	}
	// Return the first value if it's a list
	if result.Node.Nodes != nil && len(result.Node.Nodes) > 1 {
		nodes := result.Node.Nodes
		for _, node := range nodes {
			return node.Value
		}
	} else {
		return result.Node.Value
	}
	return ""
}

func (e *EtcdConf) GetValues(key string) map[string]interface{} {
	result, err := e.kapi.Get(context.Background(), key, nil)
	if err != nil {
		log.Warning.Printf("Could not load key value: %s", key)
		return nil
	}
	nodes := result.Node.Nodes
	var results = make(map[string]interface{})
	for _, node := range nodes {
		var output map[string]interface{}
		if err = json.Unmarshal([]byte(node.Value), &output); err != nil {
			log.Error.Printf("Failed to parse file %s: %v", node.Value, err)
			continue
		}
		results[node.Key] = output
	}
	return results
}
