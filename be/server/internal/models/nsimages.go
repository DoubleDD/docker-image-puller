package models

type NamespaceImages struct {
	Namespace   string       `json:"ns"`
	Deployments []Deployment `json:"deployments"`
}
type Deployment struct {
	ImageName string `json:"old"`
	Name      string `json:"name"`
	NewImage  string `json:"new"`
}
