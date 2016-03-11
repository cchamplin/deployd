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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/cchamplin/deployd/log"
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
	EstComplete   int64             `json:"estComplete"`
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
	DeploymentFailed(d *Deployment)
	Watch(key string, callback func(string))
}

type Deployments map[string]*Deployment

func (d *Deployment) Deploy(p *Package, notifier DeploymentNotifier) {
	log.Info.Printf("Deploying %s", p.Name)
	metric := p.metrics.StartMeasure()
	defer p.metrics.StopMeasure(metric)
	d.StatusMessage = "Running initialization commands"
	d.Status = STATUS_WORKING
	d.EstComplete = 0
	if ok := d.handleExecutionFragments(p.TemplatesBefore, p); !ok {
		if notifier != nil {
			notifier.DeploymentFailed(d)
		}
		return
	}

	if ok := d.handleTemplates(p, notifier); !ok {
		if notifier != nil {
			notifier.DeploymentFailed(d)
		}
		return
	}

	d.StatusMessage = "Running finalization commands"
	if ok := d.handleExecutionFragments(p.TemplatesAfter, p); !ok {
		if notifier != nil {
			notifier.DeploymentFailed(d)
		}
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
			if ok := d.handleTemplate(p.Templates[i], p, notifier); !ok {
				if notifier != nil {
					notifier.DeploymentFailed(d)
				}
				return
			}
			break
		}
	}

	d.StatusMessage = "Package Template Deployed"
	d.Status = STATUS_COMPLETE
	if notifier != nil {
		notifier.DeploymentComplete(d)
	}
}

func (d *Deployment) handleExecutionFragments(fragments ExecutionFragments, p *Package) bool {
	for i := 0; i < len(fragments); i++ {
		fragment := fragments[i]
		metric := fragment.metrics.StartMeasure()

		// TODO should we fail if the status command fails?
		if fragment.StatusCmd != "" {
			if out, err := p.ProcessedTemplates.handle(fragment.StatusCmd, &d.Variables); err != nil {
				d.StatusMessage = out
			}
		} else {
			d.StatusMessage = fragment.Status
		}
		// TODO add functionality for check commands comparing a value against the output of the command
		if fragment.CheckCmd != "" {
			_, ok := d.handleCommandTemplate(fragment.CheckCmd, p, true)
			if ok {
				if _, ok := d.handleCommandTemplate(fragment.Cmd, p, false); !ok {
					log.Trace.Printf("Deployment for package %s command failed: %s", d.PackageId, fragment.Cmd)
					fragment.metrics.StopMeasure(metric)
					d.EstComplete += fragment.metrics.PercentOfTotal(p.metrics)
					return false
				}
			} else {
			}
		} else {
			if _, ok := d.handleCommandTemplate(fragment.Cmd, p, false); !ok {
				log.Trace.Printf("Deployment for package %s command failed: %s", d.PackageId, fragment.Cmd)
				fragment.metrics.StopMeasure(metric)
				d.EstComplete += fragment.metrics.PercentOfTotal(p.metrics)
				return false

			}
		}
		// TODO complete implementation for verification commands
		fragment.metrics.StopMeasure(metric)
		d.EstComplete += fragment.metrics.PercentOfTotal(p.metrics)
	}

	return true
}

func (d *Deployment) handleCommandTemplate(tmplIdx string, p *Package, strict bool) (string, bool) {
	s, err := p.ProcessedTemplates.handle(tmplIdx, &d.Variables)
	if err == nil {
		out, ok := exec_cmd(s)
		if !ok && p.Strict {
			d.failStrict()
			return out, false
		} else if !ok && strict {
			return out, false
		}
	} else if p.Strict {
		// TODO refactor this for execution fragments
		log.Info.Printf("Deployment for package %s failed to complete: %v", d.PackageId, err)
		d.StatusMessage = fmt.Sprintf("Deployment %s of package %s failed: %v", d.Id, d.PackageId, err)
		d.Status = STATUS_FAILED
		return "", false
	}
	return "", true
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
				if len(p.Templates[i].Watch) > 0 {
					d.handleWatches(p, p.Templates[i], notifier, dest)
				}
				break
			}
		}
	}
	return val, true
}

func (d *Deployment) handleWatches(p *Package, tmp *Template, notifier DeploymentNotifier, dest string) {
	for _, tWatch := range tmp.Watch {
		watch, _ := p.ProcessedTemplates.handle(tWatch, &d.Variables)
		log.Info.Printf("Starting watch for template %s on key %s", tmp.Src+".tpl", watch)
		// TODO we should maintain an internal list of watches, and the associated
		// meta data so we don't end up with multiple watches for the same thing
		notifier.Watch(watch, func(value string) {
			if len(tmp.Before) > 0 {
				if ok := d.handleExecutionFragments(tmp.Before, p); !ok {
					return
				}
			}
			out, _ := p.ProcessedTemplates.handle(tmp.Src+".tpl", &d.Variables)
			d.handleWrite(tmp, dest, out)
			if len(tmp.After) > 0 {
				if ok := d.handleExecutionFragments(tmp.After, p); !ok {
					return
				}
			}
		})
	}
}

func (d *Deployment) handleWrite(tmp *Template, dest string, output string) error {
	log.Trace.Printf("Writing to file %s", dest)
	d1 := []byte(output)
	err := ioutil.WriteFile(dest, d1, os.FileMode(tmp.fileMode))
	os.Chown(dest, tmp.uid, tmp.gid)
	return err
}

func (d *Deployment) handleTemplates(p *Package, notifier DeploymentNotifier) bool {
	for i := 0; i < len(p.Templates); i++ {
		if ok := d.handleTemplate(p.Templates[i], p, notifier); !ok {
			return false
		}
	}
	return true
}

func (d *Deployment) handleTemplate(tmp *Template, p *Package, notifier DeploymentNotifier) bool {
	id := tmp.Src
	metric := tmp.metrics.StartMeasure()
	defer func() {
		tmp.metrics.StopMeasure(metric)
		d.EstComplete = tmp.metrics.PercentOfTotal(p.metrics)
	}()
	d.StatusMessage = tmp.Description
	log.Trace.Printf("Running %s for deployment %s of package %s", d.Status, d.Id, d.PackageId)

	var output string
	var dest string
	dest, ok := d.handleTemplateFile(id+"_dest", p, nil, "")
	if !ok {
		return false
	}
	if len(tmp.Before) > 0 {
		if ok := d.handleExecutionFragments(tmp.Before, p); !ok {
			return false
		}
	}

	if output, ok = d.handleTemplateFile(id+".tpl", p, notifier, dest); !ok {
		return false
	}
	err := d.handleWrite(tmp, dest, output)
	if err != nil {
		log.Info.Printf("Deployment for package %s failed to complete. Could not write file: %s - %v", p.Id, dest, err)
		d.StatusMessage = fmt.Sprintf("Deployment %s of package %s failed: %v", d.Id, d.PackageId, err)
		d.Status = STATUS_FAILED
		return false
	}
	if len(tmp.After) > 0 {
		if ok := d.handleExecutionFragments(tmp.After, p); !ok {
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
func exec_cmd(cmd string) (string, bool) {
	//parts := strings.Fields(cmd)
	//	head := parts[0]
	//	parts = parts[1:len(parts)]

	//out, err := exec.Command(head, parts...).CombinedOutput()
	cmdExec := exec.Command("sh", "-c", cmd)

	cmdReader, err := cmdExec.StdoutPipe()
	if err != nil {
		log.Info.Printf("Failed to execute command %s: %v", cmd, err)
		return "", false
	}

	errReader, err := cmdExec.StderrPipe()
	if err != nil {
		log.Info.Printf("Failed to execute command %s: %v", cmd, err)
		return "", false
	}
	var buffer bytes.Buffer
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			//log.Trace.Printf("Output: %s\n", scanner.Text())
			buffer.WriteString(scanner.Text())
		}
	}()

	errScanner := bufio.NewScanner(errReader)
	go func() {
		for errScanner.Scan() {
			//log.Trace.Printf("Error: %s\n", errScanner.Text())
			buffer.WriteString(scanner.Text())
		}
	}()

	err = cmdExec.Start()
	if err != nil {
		log.Info.Printf("Failed to execute command %s: %v", cmd, err)
		//log.Trace.Printf("Command Text: %s", out)
		return "", false
	}

	err = cmdExec.Wait()
	if err != nil {
		log.Info.Printf("Failed to execute command %s: %v", cmd, err)
		//log.Trace.Printf("Command Text: %s", out)
		return buffer.String(), false
	}

	log.Trace.Printf("Executed command %s", cmd)
	return buffer.String(), true
}
