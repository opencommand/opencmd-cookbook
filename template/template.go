package template

import (
	"html/template"
)

type NamedTemplate struct {
	Name     string
	Files    []string
	Template *template.Template
}
