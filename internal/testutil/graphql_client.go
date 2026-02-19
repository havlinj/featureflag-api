package testutil

import (
	"crypto/tls"
	"net/http"
	"github.com/jan-havlin-dev/featureflag-api/transport/graphql"
)

type GraphQLClient struct {
    URL       string
    AuthToken string
    Client    *http.Client
}

func NewClient(url, authToken string) *GraphQLClient {
    return &GraphQLClient{
        URL:       url,
        AuthToken: authToken,
        Client: &http.Client{
            Transport: &http.Transport{
                TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
            },
        },
    }
}

func (c *GraphQLClient) DoRequest(query string, variables map[string]interface{}) (*graphql.GraphQLResponse, error) {
    panic("unimplemented")
/*     reqBody := GraphQLRequest{
        Query:     query,
        Variables: variables,
    }

    bodyBytes, err := json.Marshal(reqBody)
    if err != nil {
        return nil, err
    }

    req, err := http.NewRequest("POST", c.URL, bytes.NewBuffer(bodyBytes))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/json")
    if c.AuthToken != "" {
        req.Header.Set("Authorization", "Bearer "+c.AuthToken)
    }

    resp, err := c.Client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    respBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var gqlResp GraphQLResponse
    if err := json.Unmarshal(respBytes, &gqlResp); err != nil {
        return nil, err
    }

    return &gqlResp, nil */
}
