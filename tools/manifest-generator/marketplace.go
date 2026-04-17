package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type MarketplaceClient struct {
	baseURL    string
	httpClient *http.Client
	csrfToken  string
}

type Release struct {
	Major       int
	Minor       int
	Patch       int
	Build       int
	VersionType string
	VersionFull string
	IsStable    bool
}

func NewMarketplaceClient() (*MarketplaceClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	client := &MarketplaceClient{
		baseURL: "https://marketplace.mendix.com/xas/",
		httpClient: &http.Client{
			Jar: jar,
		},
	}

	if err := client.initSession(); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *MarketplaceClient) initSession() error {
	reqID := uuid.New().String()
	payload := map[string]interface{}{
		"action": "get_session_data",
		"params": map[string]interface{}{},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mx-ReqToken", reqID)
	req.Header.Set("Cookie", "DeviceType=Desktop; Profile=Responsive")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}

	if csrf, ok := result["csrftoken"].(string); ok {
		c.csrfToken = csrf
	}

	return nil
}

func (c *MarketplaceClient) FetchReleases(versionTypes []string, minMajor int) ([]Release, error) {
	var allReleases []Release
	offset := 0
	pageSize := 50

	for {
		releases, hasMore, err := c.fetchPage(offset, pageSize, versionTypes, minMajor)
		if err != nil {
			return nil, err
		}

		allReleases = append(allReleases, releases...)

		if !hasMore {
			break
		}
		offset += pageSize
	}

	return allReleases, nil
}

func (c *MarketplaceClient) fetchPage(offset, limit int, versionTypes []string, minMajor int) ([]Release, bool, error) {
	reqID := uuid.New().String()
	payload := map[string]interface{}{
		"action": "retrieve_by_xpath",
		"params": map[string]interface{}{
			"xpath": "//AppStore.Framework",
			"schema": map[string]interface{}{
				"amount": limit,
				"offset": offset,
				"sort": [][]interface{}{
					{"Major", "desc"},
					{"Minor", "desc"},
					{"Patch", "desc"},
					{"Build", "desc"},
				},
			},
			"count":      true,
			"aggregates": false,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, false, err
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mx-ReqToken", reqID)
	req.Header.Set("X-Csrf-Token", c.csrfToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, err
	}

	var result struct {
		Objects      []map[string]interface{} `json:"objects"`
		HasMoreItems bool                     `json:"hasMoreItems"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, false, err
	}

	var releases []Release
	for _, obj := range result.Objects {
		release, err := parseRelease(obj, versionTypes, minMajor)
		if err != nil {
			continue
		}
		if release != nil {
			releases = append(releases, *release)
		}
	}

	return releases, result.HasMoreItems, nil
}

func parseRelease(obj map[string]interface{}, versionTypes []string, minMajor int) (*Release, error) {
	attrs, ok := obj["attributes"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing attributes")
	}

	versionType := getAttrString(attrs, "VersionType")
	status := getAttrString(attrs, "Status")

	if status == "Deprecated" {
		return nil, nil
	}

	if !contains(versionTypes, versionType) {
		return nil, nil
	}

	// Use the Version field which contains the full version string (e.g., "10.18.4.61760")
	// VersionText may be shortened for display (e.g., "10.18.4")
	versionFull := getAttrString(attrs, "Version")
	if versionFull == "" {
		// Fallback to VersionText if Version field not available
		versionFull = getAttrString(attrs, "VersionText")
	}

	versionFull = strings.Split(versionFull, " (build")[0]
	parts := strings.Split(versionFull, ".")

	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid version: %s", versionFull)
	}

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])
	build := 0
	if len(parts) == 4 {
		build, _ = strconv.Atoi(parts[3])
	}

	if major < minMajor {
		return nil, nil
	}

	// Build full version string
	fullVersion := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	if build > 0 {
		fullVersion = fmt.Sprintf("%d.%d.%d.%d", major, minor, patch, build)
	}

	return &Release{
		Major:       major,
		Minor:       minor,
		Patch:       patch,
		Build:       build,
		VersionType: versionType,
		VersionFull: fullVersion,
		IsStable:    versionType == "LTS" || versionType == "MTS" || versionType == "Stable",
	}, nil
}

func getAttrString(attrs map[string]interface{}, key string) string {
	attr, ok := attrs[key].(map[string]interface{})
	if !ok {
		return ""
	}
	if v, ok := attr["value"].(string); ok {
		return v
	}
	return ""
}

func getAttrInt(attrs map[string]interface{}, key string) int {
	attr, ok := attrs[key].(map[string]interface{})
	if !ok {
		return 0
	}
	// Try float64 first (JSON numbers)
	if v, ok := attr["value"].(float64); ok {
		return int(v)
	}
	// Try string parsing
	if v, ok := attr["value"].(string); ok {
		result, _ := strconv.Atoi(v)
		return result
	}
	return 0
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
