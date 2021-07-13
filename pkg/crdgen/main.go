package main

import (
	"fmt"
	"strings"

	gitjobv1 "github.com/rancher/gitjob/pkg/apis/gitjob.cattle.io/v1"
	"github.com/rancher/wrangler/pkg/crd"
	_ "github.com/rancher/wrangler/pkg/generated/controllers/apiextensions.k8s.io"
	"github.com/rancher/wrangler/pkg/schemas/openapi"
	"github.com/rancher/wrangler/pkg/yaml"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

var NamespacedType = "GitJob.gitjob.cattle.io/v1"

func main() {
	fmt.Println("{{- if .Capabilities.APIVersions.Has \"apiextensions.k8s.io/v1\" -}}")
	fmt.Println(generateGitJobCrd())
	fmt.Println("{{- else -}}")
	fmt.Println(generateGitJobCrdV1Beta1())
	fmt.Println("{{- end -}}")
}

func generateGitJobCrd() string {
	crdObject, err := crd.NamespacedType(NamespacedType).
		WithStatus().
		WithSchema(mustSchema(gitjobv1.GitJob{})).
		WithColumnsFromStruct(gitjobv1.GitJob{}).
		WithCustomColumn(apiextv1.CustomResourceColumnDefinition{
			Name:     "Age",
			Type:     "date",
			JSONPath: ".metadata.creationTimestamp",
		}).ToCustomResourceDefinition()
	if err != nil {
		panic(err)
	}
	return generateYamlString(crdObject)
}

func generateGitJobCrdV1Beta1() string {
	crdObject, err := crd.NamespacedType(NamespacedType).
		WithStatus().
		WithSchemaV1Beta1(mustSchemaV1Beta1(gitjobv1.GitJob{})).
		WithColumnsFromStructV1Beta1(gitjobv1.GitJob{}).
		WithCustomColumnV1Beta1(apiextv1beta1.CustomResourceColumnDefinition{
			Name:     "Age",
			Type:     "date",
			JSONPath: ".metadata.creationTimestamp",
		}).ToCustomResourceDefinitionV1Beta1()
	if err != nil {
		panic(err)
	}
	return generateYamlString(crdObject)
}

func mustSchema(obj interface{}) *apiextv1.JSONSchemaProps {
	result, err := openapi.ToOpenAPIFromStruct(obj)
	if err != nil {
		panic(err)
	}
	return result
}

func mustSchemaV1Beta1(obj interface{}) *apiextv1beta1.JSONSchemaProps {
	result, err := openapi.ToOpenAPIFromStructV1Beta1(obj)
	if err != nil {
		panic(err)
	}
	return result
}

func generateYamlString(crdObject runtime.Object) string {
	crdYaml, err := yaml.Export(crdObject)
	if err != nil {
		panic(err)
	}
	return strings.Trim(string(crdYaml), "\n")
}
