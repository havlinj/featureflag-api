package graphql

type GraphQLRequest struct {
    Query     string                 `json:"query"`
    Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
    Data   map[string]interface{} `json:"data"`
    Errors []interface{}          `json:"errors,omitempty"`
}