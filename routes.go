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

import "net/http"

type Route struct {
	Name        string
	Method      string
	Pattern     string
	Permissions string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

var routes = Routes{
	Route{
		"Index",
		"GET",
		"/",
		"Index",
		Index,
	},
	Route{
		"PackageIndex",
		"GET",
		"/packages",
		"List Packages",
		PackageIndex,
	},
	Route{
		"PackageShow",
		"GET",
		"/packages/{packageId}",
		"Package Info",
		PackageShow,
	},
	Route{
		"PackageDeploy",
		"POST",
		"/packages/{packageId}/deploy",
		"Deploy Package",
		PackageDeploy,
	},
	Route{
		"PackageDeployTemplate",
		"POST",
		"/packages/{packageId}/deploy/{templateName}",
		"Deploy Template",
		PackageDeployTemplate,
	},
	Route{
		"DeploymentIndex",
		"GET",
		"/deployments",
		"List Deployments",
		DeploymentIndex,
	},
	Route{
		"DeploymentShow",
		"GET",
		"/deployments/{deploymentId}",
		"Deployment Info",
		DeploymentShow,
	},
	Route{
		"CurrentUser",
		"GET",
		"/auth",
		"Current User",
		CurrentUser,
	},
	Route{
		"Login",
		"POST",
		"/auth",
		"*",
		Authenticate,
	},
	Route{
		"UsersIndex",
		"GET",
		"/users",
		"List Users",
		UserIndex,
	},
	Route{
		"UsersShow",
		"GET",
		"/users/{userId}",
		"User Info",
		UserShow,
	},
	Route{
		"UserUpdate",
		"PUT",
		"/users/{userId}",
		"User Info",
		UpdateUser,
	},
	Route{
		"UserCreate",
		"POST",
		"/users",
		"User Info",
		CreateUser,
	},
}
