package models

type CreateRequest struct {
	Collection string                 `json:"collection"`
	Document   map[string]interface{} `json:"document"`
}

type ReadRequest struct {
	Collection string                 `json:"collection"`
	Filter     map[string]interface{} `json:"filter"`
}

type UpdateRequest struct {
	Collection string                 `json:"collection"`
	Filter     map[string]interface{} `json:"filter"`
	Document   map[string]interface{} `json:"document"`
}

type DeleteRequest struct {
	Collection string `json:"collection"`
	Id         string `json:"id"`
}
