package utils

import (
	"bytes"
	v1 "github.com/mk100120/app-controller/api/v1"
	"html/template"
	v12 "k8s.io/api/apps/v1"
	v14 "k8s.io/api/core/v1"
	v13 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func NewDeployment(application *v1.Application) *v12.Deployment {
	d := &v12.Deployment{}
	err := yaml.Unmarshal(parseTemplate("deployment", application), d)
	if err != nil {
		panic(err)
	}
	return d
}

func NewIngress(application *v1.Application) *v13.Ingress {
	ingress := &v13.Ingress{}
	err := yaml.Unmarshal(parseTemplate("ingress", application), ingress)
	if err != nil {
		panic(err)
	}
	return ingress
}

func NewService(application *v1.Application) *v14.Service {
	service := &v14.Service{}
	err := yaml.Unmarshal(parseTemplate("service", application), service)
	if err != nil {
		panic(err)
	}
	return service
}

func parseTemplate(templateName string, application *v1.Application) []byte {
	templ, err := template.ParseFiles("internal/controller/template/" + templateName + ".yaml")
	if err != nil {
		panic(err)
	}
	b := new(bytes.Buffer)
	err = templ.Execute(b, application)
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}
