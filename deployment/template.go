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

package deployment

import (
	"os"

	"github.com/cchamplin/deployd/metrics"
)

type TemplateDef struct {
	Src         string      `json:"src"`
	Dest        string      `json:"dest"`
	Description string      `json:"description"`
	Before      interface{} `json:"before"`
	After       interface{} `json:"after"`
	Contents    string      `json:"contents"`
	Watch       interface{} `json:"watch"`
	Owner       string      `json:"owner"`
	Group       string      `json:"group"`
	Mode        string      `json:"mode"`
}

type Template struct {
	Src         string             `json:"src"`
	Dest        string             `json:"dest"`
	Description string             `json:"description"`
	Before      ExecutionFragments `json:"before"`
	After       ExecutionFragments `json:"after"`
	Contents    string             `json:"contents"`
	Watch       []string           `json:"watch"`
	Owner       string             `json:"owner"`
	Group       string             `json:"group"`
	Mode        string             `json:"mode"`
	fileMode    os.FileMode
	uid         int
	gid         int
	metrics     *metrics.Metrics
}
