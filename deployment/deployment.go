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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	//"strings"
	GoTemplate "text/template"
)

type Deployment struct {
	Id            string            `json:"id"`
	PackageId     string            `json:"packageId"`
	StatusMessage string            `json:"statusMessage"`
	Status        string            `json:"status"`
	Variables     map[string]string `json:"replacements"`
	Watch         bool              `json:"watch"`
	Template      string            `json:"template"`
}

const (
	STATUS_WORKING     = "WORKING"
	STATUS_WAITING     = "WAITING"
	STATUS_REPLICATING = "REPLICATING"
	STATUS_COMPLETE    = "COMPLETE"
	STATUS_FAILED      = "FAILED"
)

type DeploymentNotifier interface {
	DeploymentComplete(d *Deployment)
	Watch(key string, callback func(string))
}

type Deployments map[string]*Deployment

func (d *Deployment) Deploy(p *Package, notifier DeploymentNotifier) {
	log.Info.Printf("Deploying %s", p.Name)

	d.StatusMessage = "Running initialization commands"
	d.Status = STATUS_WORKING
	if ok := d.handleCommandTemplates(p.TemplatesBefore, p); !ok {
		return
	}

	d.handleTemplates(p, notifier)

	d.StatusMessage = "Running finalization commands"
	if ok := d.handleCommandTemplates(p.TemplatesAfter, p); !ok {
		return
	}

	d.StatusMessage = "Package Deployed"
	d.Status = STATUS_COMPLETE
	if notifier != nil {
		notifier.DeploymentComplete(d)
	}
}

func (d *Deployment) DeployTemplate(p *Package, notifier DeploymentNotifier, templateName string) {
	log.Info.Printf("Deploying %s:%s", p.Name, templateName)

	for i := 0; i < len(p.Templates); i++ {
		if p.Templates[i].Src == templateName {
			d.handleTemplate(&p.Templates[i], p, notifier)
		}
	}

	d.StatusMessage = "Package Template Deployed"
	d.Status = STATUS_COMPLETE
	if notifier != nil {
		notifier.DeploymentComplete(d)
	}
}

func (d *Deployment) handleCommandTemplates(templates []string, p *Package) bool {
	for i := 0; i < len(templates); i++ {
		cmd := templates[i]
		d.StatusMessage = fmt.Sprintf("Running command: %d of %d", i, len(templates))
		if ok := d.handleCommandTemplate(cmd, p); !ok {
			return false
		}
	}
	return true
}

func (d *Deployment) handleCommandTemplate(tmplIdx string, p *Package) bool {
	s, err := p.ProcessedTemplates.handle(tmplIdx, &d.Variables)
	if err == nil {
		if ok := exec_cmd(s); !ok && p.Strict {
			d.failStrict()
			return false
		}
	} else if p.Strict {
		log.Info.Printf("Deployment for package %s failed to complete: %v", d.PackageId, err)
		d.StatusMessage = fmt.Sprintf("Deployment %s of package %s failed: %v", d.Id, d.PackageId, err)
		d.Status = STATUS_FAILED
		return false
	}
	return true
}

func (d *Deployment) handleTemplateFile(tmplIdx string, p *Package, notifier DeploymentNotifier, dest string) (string, bool) {
	val, err := p.ProcessedTemplates.handle(tmplIdx, &d.Variables)
	if err != nil {
		log.Info.Printf("Deployment for package %s failed to complete: %v", d.PackageId, err)
		d.StatusMessage = fmt.Sprintf("Deployment %s of package %s failed: %v", d.Id, d.PackageId, err)
		d.Status = STATUS_FAILED
		return "", false
	}
	if d.Watch && notifier != nil {
		for i := 0; i < len(p.Templates); i++ {
			if p.Templates[i].Src+".tpl" == tmplIdx {
				if p.Templates[i].Watch != "" {
					watch, _ := p.ProcessedTemplates.handle(p.Templates[i].Watch, &d.Variables)
					log.Info.Printf("Starting watch for template %s on key %s", tmplIdx, watch)
					// TODO we should maintain an internal list of watches, and the associated
					// meta data so we don't end up with multiple watches for the same thing
					notifier.Watch(watch, func(value string) {
						out, _ := p.ProcessedTemplates.handle(tmplIdx, &d.Variables)
						log.Trace.Printf("Writing to file %s", dest)
						d1 := []byte(out)
						// TODO handle permissions correctly
						ioutil.WriteFile(dest, d1, p.Templates[i].fileMode)
						os.Chown(dest, p.Templates[i].uid, p.Templates[i].gid)
					})

				}
				break
			}
		}
	}
	return val, true
}

func (d *Deployment) handleTemplates(p *Package, notifier DeploymentNotifier) bool {
	for i := 0; i < len(p.Templates); i++ {
		if ok := d.handleTemplate(&p.Templates[i], p, notifier); !ok {
			return false
		}
	}
	return true
}

func (d *Deployment) handleTemplate(tmp *Template, p *Package, notifier DeploymentNotifier) bool {
	id := tmp.Src
	d.StatusMessage = tmp.Description
	log.Trace.Printf("Running %s for deployment %s of package %s", d.Status, d.Id, d.PackageId)

	var output string
	var dest string
	dest, ok := d.handleTemplateFile(id+"_dest", p, nil, "")
	if !ok {
		return false
	}
	if tmp.Before != "" {
		if ok := d.handleCommandTemplate(id+"_before", p); !ok {
			return false
		}
	}

	if output, ok = d.handleTemplateFile(id+".tpl", p, notifier, dest); !ok {
		return false
	}
	log.Trace.Printf("Writing to file %s", dest)
	d1 := []byte(output)
	err := ioutil.WriteFile(dest, d1, os.FileMode(tmp.fileMode))
	os.Chown(dest, tmp.uid, tmp.gid)
	if err != nil {
		log.Info.Printf("Deployment for package %s failed to complete. Could not write file: %s - %v", p.Id, dest, err)
		d.StatusMessage = fmt.Sprintf("Deployment %s of package %s failed: %v", d.Id, d.PackageId, err)
		d.Status = STATUS_FAILED
		return false
	}

	if tmp.After != "" {
		if ok := d.handleCommandTemplate(id+"_after", p); !ok {
			return false
		}
	}
	return true
}

// helper method for strict failures
func (d *Deployment) failStrict() {
	log.Trace.Printf("Exiting deployment %s of %s because of strict failure", d.Id, d.PackageId)
	d.StatusMessage = "Deployment failed"
	d.Status = STATUS_FAILED
}

func (ts GoTemplateList) handle(idx string, variables *map[string]string) (string, error) {
	tmpl, pass := ts[idx]
	if pass {
		s, err := exec_template(tmpl, variables)
		if err == nil {
			return s, nil
		} else {
			return "", err
		}
	} else {
		log.Error.Printf("Could not retrieve template file, no such index %s", idx)
	}
	return "", errors.New("Could not retrieve template file")
}

// Perform variable replacement on template
func exec_template(template *GoTemplate.Template, variables *map[string]string) (string, error) {
	var doc bytes.Buffer
	err := template.Execute(&doc, variables)
	if err != nil {
		log.Error.Printf("Could not execute template file: %v", err)
		return "", err
	}
	s := doc.String()
	return s, nil
}

// Execute shell command
func exec_cmd(cmd string) bool {
	//parts := strings.Fields(cmd)
	//	head := parts[0]
	//	parts = parts[1:len(parts)]

	//out, err := exec.Command(head, parts...).CombinedOutput()
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		log.Info.Printf("Failed to execute command %s: %v", cmd, err)
		log.Trace.Printf("Command Text: %s", out)
		return false
	}
	log.Trace.Printf("Executed command %s: %s", cmd, out)
	return true
}
