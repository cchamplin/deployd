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
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"

	GoTemplate "text/template"
)

type Repository struct {
	deploymentNotifier DeploymentNotifier
	packages           Packages
	deployments        Deployments
	mutex              *sync.Mutex
	configDirectory    string
	journalBackend     log.Journal
}

// Give us some seed data
// The notifier is used for the storage backend, I'm not happy with this
// design, it'll need to be refactored
func (r *Repository) Init(configDir string, allowUntagged bool, tags []string, journalBackend log.Journal, funcMap GoTemplate.FuncMap, notifier DeploymentNotifier) {
	log.Trace.Printf("Initializing")
	r.deploymentNotifier = notifier
	r.mutex = &sync.Mutex{}
	r.configDirectory = configDir
	r.journalBackend = journalBackend
	// Load the package definitions from the config directory
	r.LoadPackages(funcMap)

	r.deployments = make(map[string]*Deployment)

	if r.journalBackend != nil {
		r.LoadJournaledDeployments()
	}

}

func (r Repository) DeploymentComplete(d *Deployment) {
	r.JournalDeployment(d)
	r.deploymentNotifier.DeploymentComplete(d)
}
func (r Repository) DeploymentFailed(d *Deployment) {
	r.JournalDeployment(d)
}
func (r Repository) Watch(key string, callback func(string)) {
	r.deploymentNotifier.Watch(key, callback)
}

func (r *Repository) Packages() Packages {
	return r.packages
}

func (r *Repository) Deployments() Deployments {
	return r.deployments
}

func (r *Repository) FindPackage(id string) (Package, error) {
	for _, p := range r.packages {
		if p.Id == id {
			return p, nil
		}
	}

	return Package{}, errors.New("No such package exist")
}

func (r *Repository) FindDeployment(id string) (*Deployment, error) {
	// This mutex is here to protect us from possible corruption as a result
	// of multiple deployments coming in at the same time.
	r.mutex.Lock()
	defer r.mutex.Unlock()
	item, found := r.deployments[id]
	if found {
		return item, nil
	}
	return nil, errors.New("No such deployment exist")
}

func (r *Repository) AddDeployment(d *Deployment) {
	// This mutex is here to protect us from possible corruption as a result
	// of multiple deployments coming in at the same time.
	r.mutex.Lock()
	r.deployments[d.Id] = d
	r.mutex.Unlock()
}

func (r *Repository) JournalDeployment(d *Deployment) {
	if r.journalBackend != nil {

		go func() {
			// TODO decide how to act when a journal write fails
			ok := r.journalBackend.WriteEntry(d)
			if !ok {
				log.Error.Printf("Failed to write entry to journal")
			}
		}()
	} else {
		log.Trace.Printf("No journal backend loaded")
	}
}

func (r *Repository) LoadJournaledDeployments() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	entries := r.journalBackend.ReadEntries(func() interface{} {
		return &Deployment{}
	})
	for _, entry := range entries {
		d := entry.(*Deployment)
		r.deployments[d.Id] = d
		if d.Watch {
			if d.Template != "" {
				pkg, _ := r.FindPackage(d.PackageId)
				for _, tmpl := range pkg.Templates {
					if tmpl.Src == d.Template && tmpl.Watch != "" {
						var dest string
						dest, ok := d.handleTemplateFile(tmpl.Src+"_dest", &pkg, nil, "")
						if !ok {
							log.Warning.Printf("Could not resume watch for deployment %s", d.Id)
							continue
						}
						d.handleWatch(&pkg, &tmpl, r, dest)
						break
					}
				}
			} else {
				// TODO The error from FindPackage() should be handleNewNode
				// decide what should happen if the package is no longer loaded
				pkg, _ := r.FindPackage(d.PackageId)
				for _, tmpl := range pkg.Templates {
					if tmpl.Watch != "" {
						var dest string
						dest, ok := d.handleTemplateFile(tmpl.Src+"_dest", &pkg, nil, "")
						if !ok {
							log.Warning.Printf("Could not resume watch for deployment %s", d.Id)
							continue
						}
						d.handleWatch(&pkg, &tmpl, r, dest)
					}
				}
			}
		}
	}
	log.Info.Printf("Read %d journaled deployments", len(r.deployments))
	redeploys := 0
	for _, d := range r.deployments {
		if d.Status != "COMPLETE" {
			redeploys += 1
			if d.Template == "" {
				pkg, _ := r.FindPackage(d.PackageId)
				pkg.ReDeployPackage(r, d)
			} else {
				pkg, _ := r.FindPackage(d.PackageId)
				pkg.ReDeployPackageTemplate(r, d)
			}
		}
	}
	if redeploys > 0 {
		log.Info.Printf("Redeployed %d journaled deployments", redeploys)
	}
}

func (r *Repository) LoadPackages(funcMap GoTemplate.FuncMap) {
	log.Trace.Printf("Loading packages from %s ", filepath.Clean(r.configDirectory+"/packages.json"))
	ok := r.loadPackagesFromFile(filepath.Clean(r.configDirectory+"/packages.json"), funcMap)
	if !ok {
		log.Warning.Printf("Could not load packages from packages.json")
	}

	_, err := os.Stat(filepath.Clean(r.configDirectory + "/conf.d/"))
	if err == nil {
		log.Trace.Printf("Loading packages from %s ", filepath.Clean(r.configDirectory+"/conf.d"))
		files, _ := filepath.Glob(filepath.Clean(r.configDirectory + "/conf.d/*.json"))
		for _, f := range files {
			ok := r.loadPackagesFromFile(f, funcMap)
			if !ok {
				log.Info.Printf("Could not load packages from file: %s", f)
			}
		}
	}

	if r.packages == nil || len(r.packages) <= 0 {
		log.Warning.Printf("No package definitions were found")
	} else {
		log.Info.Printf("%d packages have been loaded", len(r.packages))
	}
}

func (r *Repository) loadPackagesFromFile(file string, funcMap GoTemplate.FuncMap) bool {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Warning.Printf("Failed to read file %s: %v", file, err)
		return false
	}

	// Deserialize the data
	var tPkgs Packages
	err = json.Unmarshal([]byte(data), &tPkgs)
	if err != nil {
		log.Warning.Printf("Failed to parse json file %s: %v", file, err)
		return false
	}

	log.Trace.Printf("Parsed %d packages from file %s", len(tPkgs), file)
	for idx, _ := range tPkgs {

		if tPkgs[idx].ProcessedTemplates == nil {
			tPkgs[idx].ProcessedTemplates = make(map[string]*GoTemplate.Template)
		}

		// Shell commands to be executed before templating takes place
		for _, cmd := range tPkgs[idx].TemplatesBefore {
			tPkgs[idx].processTemplate(cmd, cmd, funcMap)
		}

		// Loop through all of the templates and process them in turn
		for _, tmp := range tPkgs[idx].Templates {
			//tmpl := GoTemplate.Must(GoTemplate.New(tmp.Src + "_src").Parse(tmp.Src))
			//packages[idx].ProcessedTemplates[tmp.Src+"_src"] = tmpl
			if tmp.Mode == "" {
				tmp.fileMode = 0644
			}
			if tmp.Owner == "" {
				tmp.uid = os.Geteuid()
			} else {
				if u, err := user.Lookup(tmp.Owner); err == nil {
					if tmp.uid, err = strconv.Atoi(u.Uid); err != nil {
						tmp.uid = os.Geteuid()
					}
				} else {
					tmp.uid = os.Geteuid()
				}
			}
			if tmp.Group == "" {
				tmp.gid = os.Getgid()
			} else {
				// Right now we don't have a way to get a gid
				// from a group name
				// See: https://github.com/golang/go/issues/2617
				tmp.gid = os.Getgid()
			}
			log.Trace.Printf("Processing Template: %s", tmp.Src)
			// Most parts of the template definition (destination,template it self,
			// commands)
			tPkgs[idx].processTemplate(tmp.Src+"_dest", tmp.Dest, funcMap)
			if tmp.Watch != "" {
				tPkgs[idx].processTemplate(tmp.Watch, tmp.Watch, funcMap)
			}
			err := tPkgs[idx].processTemplateFile(r.configDirectory, tmp.Src+".tpl", tmp.Src+".tpl", funcMap)
			if err != nil {
				// TODO this isn't enough, we need to remove the package from the list
				log.Warning.Printf("Template file could not be processed: %s in package %s", tmp.Src, tPkgs[idx].Id)
				goto nextPackage
			}

			// TODO evaluate if these need to be command lists and if there
			// is such a use case
			tPkgs[idx].processTemplate(tmp.Src+"_before", tmp.Before, funcMap)
			tPkgs[idx].processTemplate(tmp.Src+"_after", tmp.After, funcMap)
		}

		// Shell commands to be executed after templating takes place
		for _, cmd := range tPkgs[idx].TemplatesAfter {
			tPkgs[idx].processTemplate(cmd, cmd, funcMap)
		}
	nextPackage:
	}
	// Compose the package list
	// TODO figure out how to handle duplicate package ids
	r.packages = append(r.packages, tPkgs...)
	return true
}
