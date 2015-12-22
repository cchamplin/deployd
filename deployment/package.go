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
	"path/filepath"
	GoTemplate "text/template"

	"../log"
	"../metrics"
	"github.com/satori/go.uuid"
)

type GoTemplateList map[string]*GoTemplate.Template

type PackageDef struct {
	Id              string        `json:"id"`
	Tag             string        `json:"tag"`
	Name            string        `json:"name"`
	Version         string        `json:"version"`
	Strict          bool          `json:"strict"`
	Templates       []TemplateDef `json:"templates"`
	TemplatesBefore []interface{} `json:"template_before"`
	TemplatesAfter  []interface{} `json:"template_after"`
}

type Package struct {
	Id                 string             `json:"id"`
	Tag                string             `json:"tag"`
	Name               string             `json:"name"`
	Version            string             `json:"version"`
	Strict             bool               `json:"strict"`
	Templates          []*Template        `json:"templates"`
	TemplatesBefore    ExecutionFragments `json:"template_before"`
	TemplatesAfter     ExecutionFragments `json:"template_after"`
	ProcessedTemplates GoTemplateList
	metrics            *metrics.Metrics
}

type Packages []Package
type PackageDefs []PackageDef

// Callback from REST handler
func (p *Package) DeployPackage(r *Repository, replacements map[string]string, watch bool) *Deployment {

	// Every deployment gets a new UUID
	u1 := uuid.NewV4().String()

	log.Info.Printf("Deploying %s - %s", p.Name, u1)
	replacements["__package"] = p.Name
	replacements["__packageId"] = p.Id
	replacements["__deploymentId"] = u1

	deployment := Deployment{Id: u1, PackageId: p.Id, Status: "NOT STARTED", StatusMessage: "Not Started", Variables: replacements, Watch: watch}

	// This should possibly be moved to somewhere else
	r.AddDeployment(&deployment)
	r.JournalDeployment(&deployment)

	log.Trace.Printf("Starting deployment %s of %s", u1, p.Name)

	// Start go routine for this deployment
	go deployment.Deploy(p, r)
	return &deployment
}

func (p *Package) ReDeployPackage(r *Repository, d *Deployment) *Deployment {

	// Every deployment gets a new UUID
	log.Info.Printf("ReDeploying %s - %s", p.Name, d.Id)

	d.Status = "NOT STARTED"
	d.StatusMessage = "Not Started"

	// This should possibly be moved to somewhere else
	// TODO What should our backend do for duplicates?
	r.AddDeployment(d)
	// TODO Don't journal redeployments?
	//r.JournalDeployment(&deployment)

	log.Trace.Printf("Starting re-deployment %s of %s", d.Id, p.Name)

	// Start go routine for this deployment
	go d.Deploy(p, r)
	return d
}

// Deploy a single template file from a Package
// TODO Should this require an existing deployment id?
// Probably yes.
// Is the usecase for being able to ad-hoc deploy template files
// without deploying a whole package worthwhile and not too
// dangerous? We may be breaking assumptions that Package
// creators have about the state of a deployment
func (p *Package) DeployPackageTemplate(r *Repository, templateName string, replacements map[string]string, watch bool) *Deployment {

	// Every deployment gets a new UUID
	u1 := uuid.NewV4().String()

	log.Info.Printf("Deploying %s - %s:%s", u1, p.Name, templateName)
	replacements["__package"] = p.Name
	replacements["__packageId"] = p.Id
	replacements["__deploymentId"] = u1

	deployment := Deployment{Id: u1, PackageId: p.Id, Status: "NOT STARTED", StatusMessage: "Not Started", Variables: replacements, Watch: watch, Template: templateName}

	// This should possibly be moved to somewhere else
	// TODO should individual template deployements
	// be counted in the backend?
	// The case for not would be if a template deployment
	// is simple there to update a file
	//r.AddDeployment(&deployment)
	r.JournalDeployment(&deployment)

	log.Trace.Printf("Starting deployment %s of %s:%s", u1, p.Name, templateName)

	// Start go routine for this deployment
	go deployment.DeployTemplate(p, r, templateName)
	return &deployment
}

func (p *Package) ReDeployPackageTemplate(r *Repository, d *Deployment) *Deployment {

	// Every deployment gets a new UUID
	log.Info.Printf("ReDeploying %s - %s:%s", p.Name, d.Id, d.Template)

	d.Status = "NOT STARTED"
	d.StatusMessage = "Not Started"

	// This should possibly be moved to somewhere else
	// TODO What should our backend do for duplicates?
	//r.AddDeployment(&deployment)
	// TODO Don't journal redeployments?
	//r.JournalDeployment(&deployment)

	log.Trace.Printf("Starting re-deployment %s of %s:%s", d.Id, p.Name, d.Template)

	// Start go routine for this deployment
	go d.DeployTemplate(p, r, d.Template)
	return d
}

func (pkg *Package) processTemplate(name string, value string, funcMap GoTemplate.FuncMap) {
	tmpl := GoTemplate.Must(GoTemplate.New(name).Funcs(funcMap).Parse(value))
	// We want templates to fail if we a suitable variable
	// was not provided in the REST request
	tmpl.Option("missingkey=error")
	pkg.ProcessedTemplates[name] = tmpl
}

func (pkg *Package) processTemplateFile(configDirectory string, name string, value string, funcMap GoTemplate.FuncMap) error {
	var tmpl *GoTemplate.Template

	// Template files can be absolute or live locally under the
	// configuration directory in /tpl
	if filepath.IsAbs(value) {
		if _, err := os.Stat(value); err != nil {
			return err
		}
		tmpl = GoTemplate.Must(GoTemplate.New(name).Funcs(funcMap).ParseFiles(value))
	} else {
		value = configDirectory + "/tpl/" + value
		if _, err := os.Stat(value); err != nil {
			return err
		}
		tmpl = GoTemplate.Must(GoTemplate.New(name).Funcs(funcMap).ParseFiles(value))
	}

	// See above
	tmpl.Option("missingkey=error")
	pkg.ProcessedTemplates[name] = tmpl
	return nil
}
