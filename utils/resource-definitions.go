package utils

type ResourceInfo struct {
	Namespace string
	Kind      string
	Name      string
	Origin    map[interface{}]interface{}
	FileName  string
	Labels    map[string]string
}
