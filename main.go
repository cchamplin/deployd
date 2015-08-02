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
	"./backends"
	"./cluster"
	"./deployment"
	"./log"
	"flag"
	"io/ioutil"
	golog "log"
	"net/http"
	"os"
	"strconv"
	GoTemplate "text/template"
)

var repo *deployment.Repository
var clstr cluster.Cluster

func main() {
	// Todo refactor this, probably split it out into a separate file
	var configFlag = flag.String("config", "/etc/deployd", "Directory for deployd to search for packages.json in")
	var dFlag = flag.Bool("d", false, "--d Display infor and warning messages during runtime")
	var debugFlag = flag.Bool("debug", false, "-Display info and warning messages during runtime")
	var verboseFlag = flag.Bool("verbose", false, "Display all available output during runtime")
	var clusterFlag = flag.Bool("nocluster", false, "Set true to disable clustering")
	flag.Parse()

	if *verboseFlag {
		log.InitLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	} else if *dFlag || *debugFlag {
		log.InitLogger(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	} else {
		log.InitLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, os.Stderr)
	}

	config := LoadConfiguration(*configFlag)
	if config == nil {
		golog.Fatal("deployd cannot be started without proper configuration")
	}
	log.Info.Printf("Starting... %s", config.Addr+":"+strconv.Itoa(config.Port))
	// This whole setup has code smell...
	// TODO refactor this
	if !*clusterFlag {
		log.Info.Printf("Starting with clustering")
		var backend = new(backends.EtcdBackend)

		clstr.Init(backend, *configFlag)
		backend.Init(&clstr, cluster.LocalMachine(config.Addr+":"+strconv.Itoa(config.Port), config.AllowedTags))
	}
	// Initialize repo
	repo = new(deployment.Repository)
	var funcMap = GoTemplate.FuncMap{"getv": clstr.Backend.GetValue, "getvs": clstr.Backend.GetValues, "gets": clstr.Backend.GetString}
	repo.Init(*configFlag, config.AllowUntagged, config.AllowedTags, funcMap, clstr.Backend)

	// Intialize the router
	router := NewRouter()

	// Start the server
	golog.Fatal(http.ListenAndServe(config.Addr+":"+strconv.Itoa(config.Port), router))
}
