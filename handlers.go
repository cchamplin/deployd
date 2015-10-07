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
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

// TODO Decide what to display here
func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "OK\n")
}

// Return listing of packages
func PackageIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(repo.Packages()); err != nil {
		log.Error.Printf("Package index request failed, encoding error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Return package details for specific package ID
func PackageShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	packageId := vars["packageId"]
	pkg, err := repo.FindPackage(packageId)

	if err == nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(pkg); err != nil {
			log.Error.Printf("Failed to encode package %s details: %v", packageId, err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	// If we didn't find it, 404
	w.WriteHeader(http.StatusNotFound)
	if err := json.NewEncoder(w).Encode(jsonErr{Code: http.StatusNotFound, Text: "Not Found"}); err != nil {
		log.Error.Printf("Failed to return 404, encoding error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

}

// List deployed packages
func DeploymentIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(repo.Deployments()); err != nil {
		log.Error.Printf("Deployment index request failed, encoding error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Return deployment details for deploymentId
func DeploymentShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	deploymentId := vars["deploymentId"]
	deployment, err := repo.FindDeployment(deploymentId)
	if err == nil {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(deployment); err != nil {
			log.Error.Printf("Failed to encode deployment %s details: %v", deploymentId, err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	// If we didn't find it, 404
	w.WriteHeader(http.StatusNotFound)
	if err := json.NewEncoder(w).Encode(jsonErr{Code: http.StatusNotFound, Text: "Not Found"}); err != nil {
		log.Error.Printf("Failed to return 404, encoding error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

}

func PackageDeploy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	packageId := vars["packageId"]
	pkg, err := repo.FindPackage(packageId)
	if err == nil {

		w.WriteHeader(http.StatusOK)

		if err := r.ParseForm(); err != nil {
			log.Warning.Printf("Failed to parse deployment request details: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(jsonErr{Code: http.StatusBadRequest, Text: "Bad Request"}); err != nil {
				log.Error.Printf("Failed to return 400, encoding error: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}

		watch := true
		if val,ok := r.Form["watch"]; ok {
			watch,_ = strconv.ParseBool(val[0])
		}

		// Parse out post vairables to be used as deployment template replacements
		items := make(map[string]string)
		for key, values := range r.PostForm {
			if len(values) > 0 {
				items[key] = values[0]
			}
		}

		deployment := pkg.DeployPackage(repo, items, watch)
		if err := json.NewEncoder(w).Encode(deployment); err != nil {
			log.Error.Printf("Failed to encode deployment %s response details: %v", deployment.Id, err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	// If we didn't find it, 404
	w.WriteHeader(http.StatusNotFound)
	if err := json.NewEncoder(w).Encode(jsonErr{Code: http.StatusNotFound, Text: "Not Found"}); err != nil {
		log.Error.Printf("Failed to return 404, encoding error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

}

func PackageDeployTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	packageId := vars["packageId"]
	templateName := vars["templateName"]
	pkg, err := repo.FindPackage(packageId)
	if err == nil {

		w.WriteHeader(http.StatusOK)

		if err := r.ParseForm(); err != nil {
			log.Warning.Printf("Failed to parse deployment request details: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(jsonErr{Code: http.StatusBadRequest, Text: "Bad Request"}); err != nil {
				log.Error.Printf("Failed to return 400, encoding error: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}

		watch := false
		if val,ok := r.Form["watch"]; ok {
			watch,_ = strconv.ParseBool(val[0])
		}

		// Parse out post vairables to be used as deployment template replacements
		items := make(map[string]string)
		for key, values := range r.PostForm {
			if len(values) > 0 {
				items[key] = values[0]
			}
		}

		deployment := pkg.DeployPackageTemplate(repo,templateName, items,watch)
		if err := json.NewEncoder(w).Encode(deployment); err != nil {
			log.Error.Printf("Failed to encode deployment %s response details: %v", deployment.Id, err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	// If we didn't find it, 404
	w.WriteHeader(http.StatusNotFound)
	if err := json.NewEncoder(w).Encode(jsonErr{Code: http.StatusNotFound, Text: "Not Found"}); err != nil {
		log.Error.Printf("Failed to return 404, encoding error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

}
