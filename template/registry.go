package template

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"sync"
)

type Registry struct {
	mu         sync.RWMutex
	templates  map[string]*NamedTemplate
	fileToKeys map[string][]string
}

func (r *Registry) Render(name string, w io.Writer, data any) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.templates[name].Template.ExecuteTemplate(w, "base", data)
}

func (r *Registry) Load(name string, files []string) error {
	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.templates[name] = &NamedTemplate{
		Name:     name,
		Files:    files,
		Template: tmpl,
	}

	for _, file := range files {
		file = filepath.Clean(file)
		r.fileToKeys[file] = append(r.fileToKeys[file], name)
	}
	return nil
}

func (r *Registry) Reload(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	nt, ok := r.templates[name]
	if !ok {
		return fmt.Errorf("template not found: %s", name)
	}

	tmpl, err := template.ParseFiles(nt.Files...)
	if err != nil {
		return err
	}

	nt.Template = tmpl
	return nil
}

func (r *Registry) GetTemplatesAffectedBy(file string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if templates, ok := r.fileToKeys[file]; ok {
		return templates, nil
	}
	return nil, fmt.Errorf("no templates depend on file: %s", file)
}

var Reg = &Registry{
	templates:  make(map[string]*NamedTemplate),
	fileToKeys: make(map[string][]string),
}
