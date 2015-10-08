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
	"../log"
	"github.com/satori/go.uuid"
	"path/filepath"
	GoTemplate "text/template"
)

type GoTemplateList map[string]*GoTemplate.Template

type Package struct {
	Id                 string     `json:"id"`
	Tag                string     `json:"tag"`
	Name               string     `json:"name"`
	Version            string     `json:"version"`
	Strict             bool       `json:"strict"`
	Templates          []Template `json:"templates"`
	TemplatesBefore    []string   `json:"template_before"`
	TemplatesAfter     []string   `json:"template_after"`
	ProcessedTemplates GoTemplateList
}

type Packages []Package

// Callback from REST handler
func (p *Package) DeployPackage(r *Repository, replacements map[string]string, watch bool) *Deployment {

	// Every deployment gets a new UUID
	u1 := uuid.NewV4().String()

	log.Info.Printf("Deploying %s - %s", p.Name, u1)

	deployment := Deployment{Id: u1, PackageId: p.Id, Status: "NOT STARTED", StatusMessage: "Not Started", Variables: replacements, Watch: watch}

	// This should possibly be moved to somewhere else
	r.AddDeployment(&deployment)

	log.Trace.Printf("Starting deployment %s of %s", u1, p.Name)

	// Start go routine for this deployment
	go deployment.Deploy(p, r.DeploymentNotifier)
	return &deployment
}

func (p *Package) DeployPackageTemplate(r *Repository, templateName string, replacements map[string]string, watch bool) *Deployment {

	// Every deployment gets a new UUID
	u1 := uuid.NewV4().String()

	log.Info.Printf("Deploying %s - %s:%s", u1, p.Name, templateName)

	deployment := Deployment{Id: u1, PackageId: p.Id, Status: "NOT STARTED", StatusMessage: "Not Started", Variables: replacements, Watch: watch, Template: templateName}

	log.Trace.Printf("Starting deployment %s of %s:%s", u1, p.Name, templateName)

	// Start go routine for this deployment
	go deployment.DeployTemplate(p, r.DeploymentNotifier, templateName)
	return &deployment
}

func (pkg *Package) processTemplate(name string, value string, funcMap GoTemplate.FuncMap) {
	tmpl := GoTemplate.Must(GoTemplate.New(name).Funcs(funcMap).Parse(value))
	// We want templates to fail if we a suitable variable
	// was not provided in the REST request
	tmpl.Option("missingkey=error")
	pkg.ProcessedTemplates[name] = tmpl
}

func (pkg *Package) processTemplateFile(configDirectory string, name string, value string, funcMap GoTemplate.FuncMap) {
	var tmpl *GoTemplate.Template

	// Template files can be absolute or live locally under the
	// configuration directory in /tpl
	if filepath.IsAbs(value) {
		tmpl = GoTemplate.Must(GoTemplate.New(name).Funcs(funcMap).ParseFiles(value))
	} else {
		value = configDirectory + "/tpl/" + value
		tmpl = GoTemplate.Must(GoTemplate.New(name).Funcs(funcMap).ParseFiles(value))
	}

	// See above
	tmpl.Option("missingkey=error")
	pkg.ProcessedTemplates[name] = tmpl
}
