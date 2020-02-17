package service

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-playground/validator/v10"
	"github.com/markbates/pkger"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	v1beta1Extensions "k8s.io/api/extensions/v1beta1"
	networkingV1 "k8s.io/api/networking/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	rbacV1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

type SecretKeyData struct {
	Name  string
	Key   string
	Value string
}

type SecretVolumeData struct {
	Name   string
	Path   string
	Secret string
}

type PVClaimData struct {
	Name              string
	Path              string
	Size              int64
	Project           string
	Role              string
	DeployEnvironment string
}

type ContainerData struct {
	Name          string
	Image         string
	Environ       map[string]string
	SecretEnviron []SecretKeyData
	SecretVolumes []SecretVolumeData
	PVClaims      []PVClaimData
}

type InitContainerData struct {
	Name    string
	Image   string
	Command string
}

type ServiceData struct {
	Name                string `validate:"required"`
	Type                string
	Role                string `validate:"is-recognized-role"`
	Project             string `validate:"required"`
	DeployEnvironment   string `validate:"is-recognized-deploy-environment"`
	Host                string
	Port                int `validate:"gte=1024,lte=10000"`
	CronSchedule        string
	WorkerReplicas      int `validate:"gte=0,lte=3"`
	CIServiceAccount    bool
	DbMigrationJob      bool
	DbMigrationTruncate bool
	Replicas            int `validate:"gte=0,lte=3"`
	InitContainers      []InitContainerData
	Container           ContainerData `validate="required,dive,required"`
	NodeGroup           string        `validate:"required"`
	PVClaims            []PVClaimData
}

func ValidateRole(fl validator.FieldLevel) bool {
	fieldValue := fl.Field().String()
	return fieldValue == "web" || fieldValue == "worker" || fieldValue == "infra"
}

func ValidateDeployEnvironment(fl validator.FieldLevel) bool {
	fieldValue := fl.Field().String()
	return fieldValue == "dev" || fieldValue == "qa" || fieldValue == "staging" || fieldValue == "production"
}

func ServiceDataStructLevelValidation(sl validator.StructLevel) {
	serviceData := sl.Current().Interface().(ServiceData)
	hostname := serviceData.Host

	hostnameRegexStringRFC952 := `^[a-zA-Z]+[a-zA-Z0-9\-\.]*[a-zA-Z0-9]$`
	hostnameRegexRFC952 := regexp.MustCompile(hostnameRegexStringRFC952)

	// For any roles, the project name must be a valid DNS since it forms
	// part of the namesapce name
	if !hostnameRegexRFC952.MatchString(serviceData.Project) {
		sl.ReportError(serviceData.Project, "ServiceProject", "Host", "Host", "")
	}

	// For any non-worker roles, the service name must be a valid hostname
	if serviceData.Role != "worker" {
		if !hostnameRegexRFC952.MatchString(serviceData.Name) {
			sl.ReportError(serviceData.Host, "ServiceName", "Host", "Host", "")
		}
	}

	// For a Web service, we want it to be:
	// 1. exposed on port 8443
	// 2. specify a valid hostname if type is external
	// 3. specify type to be either internal or external
	if serviceData.Role == "web" {
		if serviceData.Port != 8443 {
			sl.ReportError(serviceData.Port, "ServicePort", "Port", "Port", "")
		}
		if serviceData.Type == "external" {
			if len(serviceData.Host) == 0 {
				sl.ReportError(serviceData.Host, "ServiceHost", "Host", "Host", "")
			} else {
				if hostname[len(hostname)-1] == '.' {
					hostname = hostname[0 : len(hostname)-1]
				}
				if !(strings.ContainsAny(hostname, ".") && hostnameRegexRFC952.MatchString(hostname)) {
					sl.ReportError(serviceData.Host, "ServiceHost", "Host", "Host", "")
				}
			}

		}
		if !(len(serviceData.Type) != 0 && (serviceData.Type == "internal" || serviceData.Type == "external")) {
			sl.ReportError(serviceData.Type, "ServiceType", "Type", "Type", "")
		}
	}

	// For a infra service, we want it to be:
	// 1. Exposed on a valid port
	// 2. Exposed internally
	// 3. No host specified
	if serviceData.Role == "infra" {
		if len(serviceData.Host) != 0 {
			fmt.Printf("Host specified: %s\n", serviceData.Host)
			sl.ReportError(serviceData.Host, "ServiceHost", "Host", "Host", "")
		}
		if len(serviceData.Type) != 0 && serviceData.Type != "internal" {
			sl.ReportError(serviceData.Type, "ServiceType", "Type", "Type", "")
		}
	}

}

func MatchExpectations(data *ServiceData) bool {
	validate := validator.New()
	validate.RegisterValidation("is-recognized-role", ValidateRole)
	validate.RegisterValidation("is-recognized-deploy-environment", ValidateDeployEnvironment)
	validate.RegisterStructValidation(ServiceDataStructLevelValidation, ServiceData{})

	err := validate.Struct(data)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			fmt.Printf("Invalid Validation error: %v\n", err)
			return false
		}

		for _, err := range err.(validator.ValidationErrors) {

			fmt.Printf("Match expectations error: %#v\n", err)
		}

		fmt.Println()

		return false
	} else {
		return true
	}
}

func GenerateK8sService(data *ServiceData, directory string, forceWrite bool) {

	var generatedK8s bytes.Buffer
	// helping the pkger parser include the files I need
	_ = pkger.Dir("/service_templates")

	if MatchExpectations(data) {
		seedTemplates := []string{
			"/service_templates/namespace.yaml.tmpl",
			"/service_templates/sa.yaml.tmpl",
			"/service_templates/network_policy.yaml.tmpl",
			"/service_templates/ingress.yaml.tmpl",
			"/service_templates/db_migration.yaml.tmpl",
			"/service_templates/pvc_yaml.tmpl",
			"/service_templates/deployment.yaml.tmpl",
		}

		for _, sT := range seedTemplates {

			tmpl, err := ParseFiles(sT)
			if err != nil {
				log.Fatal(err)
			}
			tmpl.Execute(&generatedK8s, data)
			if err != nil {
				log.Fatal(err)
			}
		}

		if data.WorkerReplicas != 0 {
			data.Role = "worker"
			if MatchExpectations(data) {

				tmpl, err := ParseFiles("/service_templates/deployment.yaml.tmpl")
				if err != nil {
					log.Fatal(err)
				}
				tmpl.Execute(&generatedK8s, data)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				log.Fatal("Validations failed\n")
			}
		}

		if len(data.CronSchedule) != 0 {
			tmpl, err := ParseFiles("/service_templates/cron.yaml.tmpl")
			if err != nil {
				log.Fatal(err)
			}
			tmpl.Execute(&generatedK8s, data)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		log.Fatal("Validations failed")
	}

	// TODO: Summarize that resources that is described by the generated spec
	if directory == "" {

		fmt.Printf("%s\n", generatedK8s.String())

	} else {

		writeToDirectory(generatedK8s, directory, forceWrite)

	}

}
func writeToDirectory(generatedK8s bytes.Buffer, directoryPath string, forceWrite bool) {
	// deal with all those rubbish situations, like if the directory doesn't exist
	location := filepath.Dir(directoryPath)
	if _, err := os.Stat(location); os.IsNotExist(err) {
		log.Fatal(err)
		// The directory they want to create the new directory in should exist already.
	}
	// now the directory exists. create the new directory on top of it
	// unless it's the current directory
	err := os.Mkdir(directoryPath, os.ModeDir|0755)
	// if there's an error not related to the directory already existing, then leave
	// also leave if the err is that it already exists and they didn't specify force write.
	if (err != nil && !os.IsExist(err)) || (os.IsExist(err) && !forceWrite) {
		log.Fatal(err)
	}
	// now we have left if we really should have already. we either successfully created a fresh
	// directory or we DIDN'T BUT we had specified force write.
	// so now we can safely open the files in Create mode.
	decode := scheme.Codecs.UniversalDeserializer().Decode

	// idea borrowed from kubeval
	bits := bytes.Split(generatedK8s.Bytes(), []byte(DetectLineBreak(generatedK8s.Bytes())+"---"+DetectLineBreak(generatedK8s.Bytes())))
	// counter to give each resource a unique filename
	var counter = map[string]uint{
		"Pod":                   0,
		"Namespace":             0,
		"PersistentVolumeClaim": 0,
		"Deployment":            0,
		"Ingress":               0,
		"Role":                  0,
		"RoleBinding":           0,
		"ServiceAccount":        0,
		"Service":               0,
	}
	for _, resource := range bits {
		if len(resource) == 0 || len(strings.Trim(string(resource), "\n")) == 0 {
			continue
		}
		obj, groupVersionKind, err := decode(resource, nil, nil)
		if err != nil {
			log.Fatalf("Error while decoding YAML object: %v\n", err)
		}
		switch o := obj.(type) {
		case *v1.Pod:
			// write the file
			writeResourceToFile(directoryPath, resource, "Pod", counter)
		case *v1.Namespace:
			writeResourceToFile(directoryPath, resource, "Namespace", counter)
		case *v1.PersistentVolumeClaim:
			writeResourceToFile(directoryPath, resource, "PersistentVolumeClaim", counter)
		case *appsv1.Deployment:
			writeResourceToFile(directoryPath, resource, "Deployment", counter)
		case *v1beta1Extensions.Ingress:
			writeResourceToFile(directoryPath, resource, "Ingress", counter)
		case *networkingV1.NetworkPolicy:
			writeResourceToFile(directoryPath, resource, "NetworkPolicy", counter)
		case *rbacV1.Role:
			writeResourceToFile(directoryPath, resource, "Role", counter)
		case *rbacV1beta1.RoleBinding:
			writeResourceToFile(directoryPath, resource, "RoleBinding", counter)
		case *v1.ServiceAccount:
			writeResourceToFile(directoryPath, resource, "ServiceAccount", counter)
		case *v1.Service:
			writeResourceToFile(directoryPath, resource, "Service", counter)
		default:
			fmt.Printf("Unknown object: %#v: %v\n\n", o, groupVersionKind)
		}
	}

}

func writeResourceToFile(directory string, resource []byte, resourceType string, counter map[string]uint) {
	path := filepath.Join(directory, createFileName(resourceType, counter))
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(f, "%s\n", string(resource))
}

func createFileName(resourceType string, counter map[string]uint) string {
	numAsString := strconv.Itoa(int(counter[resourceType]))
	counter[resourceType]++
	if numAsString == "0" {
		numAsString = ""
	}
	return resourceType + numAsString + ".yaml"
}

// This is copy and pasted from golang's text/template/helper.go,
// replacing os file calls with pkger calls. oh god.
func ParseFiles(filenames ...string) (*template.Template, error) {
	if len(filenames) == 0 {
		return nil, fmt.Errorf("template: no files named in call to ParseFiles")
	}
	var t *template.Template
	for _, filename := range filenames {
		b, err := ReadFilePkgr(filename)
		if err != nil {
			return nil, err
		}
		s := string(b)
		name := filepath.Base(filename)

		var tmpl *template.Template
		t = template.New(name)
		if name == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(name)
		}
		_, err = tmpl.Parse(s)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

// I'm really sorry but I need to do this as well.
func ReadFilePkgr(filename string) ([]byte, error) {
	f, err := pkger.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var n int64 = bytes.MinRead

	if fi, err := f.Stat(); err == nil {
		if size := fi.Size() + bytes.MinRead; size > n {
			n = size
		}
	}
	return ioutil.ReadAll(f)
}
