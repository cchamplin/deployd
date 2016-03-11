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
	"github.com/cchamplin/deployd/auth"
	"github.com/cchamplin/deployd/backends/conf"
	"github.com/cchamplin/deployd/log"
	"strings"
)

type ServerConfiguration struct {
	Addr          string                 `json:"bind-addr"`
	Port          int                    `json:"bind-port"`
	AllowedTags   []string               `json:"allowed-tags"`
	AllowUntagged bool                   `json:"allow-untagged"`
	Journal       map[string]interface{} `json:"journal"`
	Auth          map[string]interface{} `json:"journal"`
	Backend       ConfigurationBackend
}

type ConfigurationBackend interface {
	GetPath() string
	GetValue(key string) map[string]interface{}
	GetString(key string) string
	GetValues(key string) map[string]interface{}
}

func ConfigurationFromBackend(configLocation string) *ServerConfiguration {
	configParts := strings.Split(configLocation, ",")
	var backendType string
	var backendHost = "127.0.0.1:4100"
	var configPath = "/deployd/config"
	if len(configParts) == 3 {
		backendType = configParts[0]
		backendHost = configParts[1]
		configPath = configParts[2]
	} else if len(configParts) == 2 {
		backendType = configParts[0]
		backendHost = configParts[1]
	}

	var result ServerConfiguration
	var configBackend ConfigurationBackend

	switch backendType {
	case "etcd":
		var etcdBackend = new(conf.EtcdConf)
		etcdBackend.Init(backendHost, configPath)
		configBackend = etcdBackend
	case "default", "fs", "json":
		var defaultBackend = new(conf.DefaultConf)
		defaultBackend.Init(backendHost)
		configBackend = defaultBackend
	default:
		log.Error.Printf("%s is an unknown configuration provider", backendType)
		return nil
	}

	result.Backend = configBackend

	data := configBackend.GetString(fmt.Sprintf("%s/config", configBackend.GetPath()))

	err := json.Unmarshal([]byte(data), &result)
	if err != nil {
		log.Error.Printf("Failed to parse json: %v", err)
		return nil
	}
	return &result
}
