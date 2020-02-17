package service

import (
	"bytes"
	"log"
	"runtime"
	"strings"

	. "bitbucket.org/welovetravel/xops/service/lint"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

// copied from https://github.com/instrumenta/kubeval/blob/9c9c0a5b3cc619dbd94129af77c8512bfd0f1763/kubeval/utils.go#L24
func DetectLineBreak(haystack []byte) string {
	windowsLineEnding := bytes.Contains(haystack, []byte("\r\n"))
	if windowsLineEnding && runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}

/**
* This will take a buffer of bytes that encode a YAML object definition (potentially multi-doc)
* and will return one or many (in the case of multiple objects) metav1Objects
* This is an interface defined in meta/v1 and it's probably the most convenient to use. I love it.
* You can access common properties of resources like Name, Namespace, Kind.
**/
func ConvertToMetaV1Objects(data *bytes.Buffer) []metav1.Object {
	var objects []metav1.Object
	newline := DetectLineBreak(data.Bytes())
	bits := bytes.Split(data.Bytes(), []byte(newline+"---"+newline))
	// 1. Iterate over each byte representation of an object
	for _, resource := range bits {
		if len(resource) == 0 || len(strings.Trim(string(resource), newline)) == 0 {
			continue
		}
		// 2. Decode the object into its corresponding k8s type (eg *appsv1.Deployment)
		concrete, _, err := scheme.Codecs.UniversalDeserializer().Decode(resource, nil, nil)
		if err != nil {
			continue
		}
		// 3. Try to get the object to conform to the metav1.Object interface
		metav1Object, err := meta.Accessor(concrete)
		if err != nil {
			continue
		}
		objects = append(objects, metav1Object)
	}
	return objects
}

func AttachMetaData(data *bytes.Buffer, yamlFilePath string) []*YamlDerivedKubernetesResource {
	// set up required for each call of this function, one call is for one yaml file.
	currentObjectNum := 0
	lineNumber := FindLineNumbers(data)

	var yamlObjects []*YamlDerivedKubernetesResource
	decode := scheme.Codecs.UniversalDeserializer().Decode
	// idea borrowed from kubeval
	bits := bytes.Split(data.Bytes(), []byte(DetectLineBreak(data.Bytes())+"---"+DetectLineBreak(data.Bytes())))

	for _, resource := range bits {
		if len(resource) == 0 || len(strings.Trim(string(resource), "\n")) == 0 {
			continue
		}
		obj, _, err := decode(resource, nil, nil)
		if err != nil {
			log.Fatal(err)
		}
		currentResource := &YamlDerivedKubernetesResource{
			Resource: obj,
			Metadata: Metadata{
				LineNumber: lineNumber[currentObjectNum],
				FilePath:   yamlFilePath,
			},
		}
		yamlObjects = append(yamlObjects, currentResource)
		currentObjectNum += 1
	}
	return yamlObjects
}

/**
* For each object (in the order that they occur in the yaml file), tell me what line number the object starts on.
* This is brittle, will break as soon as kubernetes objects aren't given the apiVersion as the first key sorry about this.
 */
func FindLineNumbers(data *bytes.Buffer) []int {
	objectSignifier := []byte("apiVersion:")
	numObjects := bytes.Count(data.Bytes(), objectSignifier)
	lineNum := make([]int, numObjects)
	currentObject := 0

	for i, line := range bytes.Split(data.Bytes(), []byte("\n")) {
		if bytes.Contains(line, objectSignifier) {
			lineNum[currentObject] = i + 1
			currentObject += 1
		}
	}
	return lineNum
}
