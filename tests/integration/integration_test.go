package integration

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
    suite.Suite
    baseURL    string
    httpClient *http.Client
    testTeam   string
}

func TestIntegrationSuite(t *testing.T) {
    suite.Run(t, new(IntegrationTestSuite))
}

func (suite *IntegrationTestSuite) SetupSuite() {
    port := getEnv("PORT", "8080")
    suite.baseURL = fmt.Sprintf("http://localhost:%s", port)
    suite.httpClient = &http.Client{Timeout: 10 * time.Second}
    suite.testTeam = "integration_team"
    
    suite.waitForService()

    suite.setupTestData()
}

func getEnv(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    return value
}

func (suite *IntegrationTestSuite) waitForService() {
    for i := 0; i < 30; i++ {
        resp, err := http.Get(suite.baseURL + "/stats/system")
        if err == nil && resp.StatusCode == 200 {
            fmt.Println("✅ Service is ready for testing")
            return
        }
        time.Sleep(2 * time.Second)
    }
    suite.T().Fatal("Service didn't become ready in time")
}

func (suite *IntegrationTestSuite) setupTestData() {
    teamData := map[string]interface{}{
        "team_name": suite.testTeam,
        "members": []map[string]interface{}{
            {"user_id": "int_u1", "username": "Integration User 1", "is_active": true},
            {"user_id": "int_u2", "username": "Integration User 2", "is_active": true},
            {"user_id": "int_u3", "username": "Integration User 3", "is_active": true},
        },
    }

    jsonData, _ := json.Marshal(teamData)
    resp, err := suite.httpClient.Post(suite.baseURL+"/team/add", "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Printf("Warning: Could not setup test data: %v\n", err)
        return
    }
    resp.Body.Close()
    
    fmt.Println("✅ Test data setup complete")
}

func (suite *IntegrationTestSuite) TestTeamWorkflow() {
    t := suite.T()

    resp, err := suite.httpClient.Get(suite.baseURL + "/team/get?team_name=" + suite.testTeam)
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    var teamResponse map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&teamResponse)
    resp.Body.Close()

    assert.Equal(t, suite.testTeam, teamResponse["team_name"])
    members := teamResponse["members"].([]interface{})
    assert.Len(t, members, 3)

    teamData := map[string]interface{}{
        "team_name": suite.testTeam,
        "members": []map[string]interface{}{
            {"user_id": "new_user", "username": "New User", "is_active": true},
        },
    }

    jsonData, _ := json.Marshal(teamData)
    resp, err = suite.httpClient.Post(suite.baseURL+"/team/add", "application/json", bytes.NewBuffer(jsonData))
    assert.NoError(t, err)
    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

    var errorResponse map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&errorResponse)
    resp.Body.Close()
    assert.Equal(t, "TEAM_EXISTS", errorResponse["error"].(map[string]interface{})["code"])
}

func (suite *IntegrationTestSuite) TestUserWorkflow() {
    t := suite.T()

    deactivateData := map[string]interface{}{
        "user_id":   "int_u2",
        "is_active": false,
    }

    jsonData, _ := json.Marshal(deactivateData)
    resp, err := suite.httpClient.Post(suite.baseURL+"/users/setIsActive", "application/json", bytes.NewBuffer(jsonData))
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    var userResponse map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&userResponse)
    resp.Body.Close()

    user := userResponse["user"].(map[string]interface{})
    assert.Equal(t, false, user["is_active"])


    resp, err = suite.httpClient.Get(suite.baseURL + "/team/get?team_name=" + suite.testTeam)
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    var teamResponse map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&teamResponse)
    resp.Body.Close()

    members := teamResponse["members"].([]interface{})
    for _, member := range members {
        m := member.(map[string]interface{})
        if m["user_id"] == "int_u2" {
            assert.Equal(t, false, m["is_active"])
        }
    }
}

func (suite *IntegrationTestSuite) TestPRWorkflow() {
    t := suite.T()

    prData := map[string]interface{}{
        "pull_request_id":   "int_pr_1",
        "pull_request_name": "Integration Test PR",
        "author_id":         "int_u1",
    }

    jsonData, _ := json.Marshal(prData)
    resp, err := suite.httpClient.Post(suite.baseURL+"/pullRequest/create", "application/json", bytes.NewBuffer(jsonData))
    assert.NoError(t, err)
    assert.Equal(t, http.StatusCreated, resp.StatusCode)

    var prResponse map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&prResponse)
    resp.Body.Close()

    pr := prResponse["pr"].(map[string]interface{})
    assert.Equal(t, "int_pr_1", pr["pull_request_id"])
    assert.Equal(t, "OPEN", pr["status"])
    
    reviewers := pr["assigned_reviewers"].([]interface{})
    assert.True(t, len(reviewers) >= 1, "Should have at least one reviewer")

    if len(reviewers) > 0 {
        resp, err = suite.httpClient.Get(suite.baseURL + "/users/getReview?user_id=" + reviewers[0].(string))
        assert.NoError(t, err)
        assert.Equal(t, http.StatusOK, resp.StatusCode)

        var reviewsResponse map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&reviewsResponse)
        resp.Body.Close()

        assert.Equal(t, reviewers[0].(string), reviewsResponse["user_id"])
        pullRequests := reviewsResponse["pull_requests"].([]interface{})
        assert.GreaterOrEqual(t, len(pullRequests), 1)
        
        found := false
        for _, pr := range pullRequests {
            if pr.(map[string]interface{})["pull_request_id"] == "int_pr_1" {
                found = true
                break
            }
        }
        assert.True(t, found, "PR should be in reviewer's list")
    }

    mergeData := map[string]interface{}{
        "pull_request_id": "int_pr_1",
    }

    jsonData, _ = json.Marshal(mergeData)
    resp, err = suite.httpClient.Post(suite.baseURL+"/pullRequest/merge", "application/json", bytes.NewBuffer(jsonData))
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    json.NewDecoder(resp.Body).Decode(&prResponse)
    resp.Body.Close()

    pr = prResponse["pr"].(map[string]interface{})
    assert.Equal(t, "MERGED", pr["status"])

    resp, err = suite.httpClient.Post(suite.baseURL+"/pullRequest/merge", "application/json", bytes.NewBuffer(jsonData))
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func (suite *IntegrationTestSuite) TestReassignmentWorkflow() {
    t := suite.T()

    prData := map[string]interface{}{
        "pull_request_id":   "int_pr_reassign",
        "pull_request_name": "Reassignment Test PR",
        "author_id":         "int_u1",
    }

    jsonData, _ := json.Marshal(prData)
    resp, err := suite.httpClient.Post(suite.baseURL+"/pullRequest/create", "application/json", bytes.NewBuffer(jsonData))
    assert.NoError(t, err)
    assert.Equal(t, http.StatusCreated, resp.StatusCode)

    var prResponse map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&prResponse)
    resp.Body.Close()

    pr := prResponse["pr"].(map[string]interface{})
    reviewers := pr["assigned_reviewers"].([]interface{})
    assert.True(t, len(reviewers) >= 1, "PR should have at least one reviewer")

    oldReviewer := reviewers[0].(string)

    reassignData := map[string]interface{}{
        "pull_request_id": "int_pr_reassign",
        "old_user_id":     oldReviewer,
    }

    jsonData, _ = json.Marshal(reassignData)
    resp, err = suite.httpClient.Post(suite.baseURL+"/pullRequest/reassign", "application/json", bytes.NewBuffer(jsonData))
    
    if resp.StatusCode == http.StatusConflict {
        var conflictResponse map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&conflictResponse)
        resp.Body.Close()
        
        errorCode := conflictResponse["error"].(map[string]interface{})["code"]
        assert.Contains(t, []string{"NO_CANDIDATE", "NOT_ASSIGNED"}, errorCode)
        return
    }

    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    var reassignResponse map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&reassignResponse)
    resp.Body.Close()

    newPR := reassignResponse["pr"].(map[string]interface{})
    newReviewers := newPR["assigned_reviewers"].([]interface{})
    
    assert.NotContains(t, newReviewers, oldReviewer)
    assert.Contains(t, reassignResponse, "replaced_by")
}

func (suite *IntegrationTestSuite) TestErrorScenarios() {
    t := suite.T()

    testCases := []struct {
        name       string
        url        string
        method     string
        data       map[string]interface{}
        wantStatus int
        wantError  string
    }{
        {
            name:       "Get non-existent team",
            url:        "/team/get?team_name=nonexistent",
            method:     "GET",
            wantStatus: http.StatusNotFound,
            wantError:  "NOT_FOUND",
        },
        {
            name:   "Create PR with non-existent author",
            url:    "/pullRequest/create",
            method: "POST",
            data: map[string]interface{}{
                "pull_request_id":   "error_pr_1",
                "pull_request_name": "Error Test PR",
                "author_id":         "nonexistent_user",
            },
            wantStatus: http.StatusNotFound,
            wantError:  "NOT_FOUND",
        },
        {
            name:   "Merge non-existent PR",
            url:    "/pullRequest/merge",
            method: "POST",
            data: map[string]interface{}{
                "pull_request_id": "nonexistent_pr",
            },
            wantStatus: http.StatusNotFound,
            wantError:  "NOT_FOUND",
        },
        {
            name:   "Reassign non-assigned reviewer", 
            url:    "/pullRequest/reassign",
            method: "POST",
            data: map[string]interface{}{
                "pull_request_id": "int_pr_1", 
                "old_user_id":     "nonexistent_user",
            },
            wantStatus: http.StatusNotFound,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            var req *http.Request
            var err error

            if tc.method == "POST" {
                jsonData, _ := json.Marshal(tc.data)
                req, err = http.NewRequest(tc.method, suite.baseURL+tc.url, bytes.NewBuffer(jsonData))
                if err == nil {
                    req.Header.Set("Content-Type", "application/json")
                }
            } else {
                req, err = http.NewRequest(tc.method, suite.baseURL+tc.url, nil)
            }

            assert.NoError(t, err)

            resp, err := suite.httpClient.Do(req)
            assert.NoError(t, err)
            defer resp.Body.Close()

            assert.Equal(t, tc.wantStatus, resp.StatusCode)

            if tc.wantError != "" {
                var errorResponse map[string]interface{}
                json.NewDecoder(resp.Body).Decode(&errorResponse)
                assert.Equal(t, tc.wantError, errorResponse["error"].(map[string]interface{})["code"])
            }
        })
    }
}

func (suite *IntegrationTestSuite) TestStatistics() {
    t := suite.T()

    endpoints := []string{
        "/stats/system",
        "/stats/users", 
        "/stats/prs",
        "/stats/top-reviewers",
    }

    for _, endpoint := range endpoints {
        resp, err := suite.httpClient.Get(suite.baseURL + endpoint)
        assert.NoError(t, err)
        assert.Equal(t, http.StatusOK, resp.StatusCode)
        
        var response map[string]interface{}
        err = json.NewDecoder(resp.Body).Decode(&response)
        assert.NoError(t, err)
        resp.Body.Close()
    }
}