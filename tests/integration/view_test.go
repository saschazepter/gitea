// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestSiteManifest(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/assets/site-manifest.json")
	resp := MakeRequest(t, req, http.StatusOK)

	assert.Equal(t, "application/json;charset=utf-8", resp.Header().Get("Content-Type"))

	// When no Host header is present (test environment), absoluteAssetURL falls back to
	// setting.AppURL (which has a trailing slash), giving a double-slash prefix for icon paths.
	assetBase := setting.AppURL
	expectedJSON := fmt.Sprintf(`{
		"name": %q,
		"short_name": %q,
		"start_url": %q,
		"icons": [
			{"src": %q, "type": "image/png",     "sizes": "512x512"},
			{"src": %q, "type": "image/svg+xml",  "sizes": "512x512"}
		]
	}`,
		setting.AppName,
		setting.AppName,
		setting.AppURL,
		assetBase+"/assets/img/logo.png",
		assetBase+"/assets/img/logo.svg",
	)
	assert.JSONEq(t, expectedJSON, resp.Body.String())
}

func TestRenderFileSVGIsInImgTag(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")

	req := NewRequest(t, "GET", "/user2/repo2/src/branch/master/line.svg")
	resp := session.MakeRequest(t, req, http.StatusOK)

	doc := NewHTMLParser(t, resp.Body)
	src, exists := doc.doc.Find(".file-view img").Attr("src")
	assert.True(t, exists, "The SVG image should be in an <img> tag so that scripts in the SVG are not run")
	assert.Equal(t, "/user2/repo2/raw/branch/master/line.svg", src)
}

func TestCommitListActions(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")

	t.Run("WikiRevisionList", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/repo1/wiki/Home?action=_revision")
		resp := session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)
		AssertHTMLElement(t, htmlDoc, ".commit-list .copy-commit-id", true)
		AssertHTMLElement(t, htmlDoc, `.commit-list .view-single-diff`, false)
		AssertHTMLElement(t, htmlDoc, `.commit-list .view-commit-path`, false)
	})

	t.Run("RepoCommitList", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/repo1/commits/branch/master")
		resp := session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		AssertHTMLElement(t, htmlDoc, `.commit-list .copy-commit-id`, true)
		AssertHTMLElement(t, htmlDoc, `.commit-list .view-single-diff`, false)
		AssertHTMLElement(t, htmlDoc, `.commit-list .view-commit-path`, true)
	})

	t.Run("RepoFileHistory", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/repo1/commits/branch/master/README.md")
		resp := session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		AssertHTMLElement(t, htmlDoc, `.commit-list .copy-commit-id`, true)
		AssertHTMLElement(t, htmlDoc, `.commit-list .view-single-diff`, true)
		AssertHTMLElement(t, htmlDoc, `.commit-list .view-commit-path`, true)
	})
}
