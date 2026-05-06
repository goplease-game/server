package api_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/service"
	"github.com/ognev-dev/goplease/server/handler"
	"github.com/ognev-dev/goplease/server/request"
	"github.com/ognev-dev/goplease/test"
	"github.com/ognev-dev/goplease/test/factory/random"
	"github.com/stretchr/testify/assert"
)

func TestCreatePage(t *testing.T) {
	// only admins can create pages for now
	admin := loginAsAdmin(t)

	req := request.CreatePage{
		PublicID: random.String(),
		Title:    random.Title(),
		Content:  random.String(),
	}

	var resp ds.Page
	CREATE(t, "/pages/", req, &resp)

	contentHTML, err := app.MarkdownToHTML(req.Content)
	test.CheckErr(t, err)
	assert.Equal(t, resp.Content, contentHTML)

	// check entity created
	test.AssertInDB(t, tt.DB, "entities", test.Data{
		"id":         resp.ID,
		"public_id":  req.PublicID,
		"title":      req.Title,
		"owner_id":   admin.ID,
		"type":       ds.EntityTypePage,
		"status":     ds.EntityStatusApproved,
		"visibility": ds.EntityVisibilityPublic,
	})

	// check page created
	test.AssertInDB(t, tt.DB, "pages", test.Data{
		"id":          resp.ID,
		"content_raw": req.Content,
		"content":     contentHTML,
	})

	// check log created
	test.AssertInDB(t, tt.DB, "event_logs", test.Data{
		"entity_id": resp.ID,
		"user_id":   admin.ID,
		"type":      ds.EventLogEntityAdded,
	})

	t.Run("public_id already taken", func(t *testing.T) {
		var errResp handler.Error
		Request(t, RequestArgs{
			method:       http.MethodPost,
			path:         "/pages/",
			body:         req,
			bindResponse: &errResp,
			assertStatus: http.StatusUnprocessableEntity,
		})

		errMsg, ok := errResp.InputErrors["public_id"]
		assert.True(t, ok)
		assert.NotEmpty(t, errMsg)
	})
}

func TestUpdatePage_WithReview(t *testing.T) {
	user := login(t)

	page := create[ds.Page](t)
	var resp service.EntityChange
	GET(t, pf("/pages/%s/edit/", page.PublicID), &resp)

	// first response's data should be same as page and with revision=0
	assert.Equal(t, page.ID, resp.ID)
	assert.Equal(t, 0, resp.Revision)
	assert.Len(t, page.Data(), len(resp.Data))

	for k, v := range page.Data() {
		assert.Equal(t, fmt.Sprintf("%v", v), fmt.Sprintf("%v", resp.Data[k]))
	}

	newContent := random.Edit(page.ContentRaw)

	// do update (change only content)
	updateReq := request.UpdatePage{
		CreatePage: request.CreatePage{
			PublicID: page.PublicID,
			Title:    page.Title,
			Content:  newContent,
		},
	}
	var updateResp ds.EntityChangeRequest
	UPDATE(t, pf("/pages/%s/", page.PublicID), updateReq, &updateResp)

	assert.Equal(t, 1, updateResp.Revision)
	assert.Equal(t, ds.EntityChangePending, updateResp.Status)

	contentPacth := app.MakePatch(page.ContentRaw, newContent)

	// new change request should be created
	test.AssertInDB(t, tt.DB, "entity_change_requests", test.Data{
		"user_id":   user.ID,
		"entity_id": page.ID,
		"status":    ds.EntityChangePending,
		"revision":  1,
		"diff":      map[string]any{"content": contentPacth},
	})

	// page itself should not be changed
	test.AssertInDB(t, tt.DB, "entities", test.Data{
		"id":    page.ID,
		"title": resp.Data["title"],
	})

	// next edit should return "in-progress" data
	GET(t, pf("/pages/%s/edit/", page.PublicID), &resp)
	assert.Equal(t, page.ID, resp.ID)
	assert.Equal(t, 1, resp.Revision)
	assert.Len(t, page.Data(), len(resp.Data))
	assert.Equal(t, any(updateReq.Content), resp.Data["content"])

	// updating page that already have change request for review
	// should only update that request
	newTitle := random.Edit(page.Title)
	titlePatch := app.MakePatch(page.Title, newTitle)
	updateReq.Title = newTitle
	UPDATE(t, pf("/pages/%s/", page.PublicID), updateReq, &updateResp)
	test.AssertInDB(t, tt.DB, "entity_change_requests", test.Data{
		"user_id":    user.ID,
		"entity_id":  page.ID,
		"status":     ds.EntityChangePending,
		"revision":   2,            // revision should be incremented
		"updated_at": test.NotNull, // updated_at should be set
		"diff": map[string]any{
			"title":   titlePatch,
			"content": contentPacth,
		},
	})
}

func TestUpdatePage_WithoutReview(t *testing.T) {
	admin := loginAsAdmin(t)

	page := create[ds.Page](t)
	newContent := random.Edit(page.ContentRaw)

	// do update (change only content)
	req := request.UpdatePage{
		CreatePage: request.CreatePage{
			PublicID: page.PublicID,
			Title:    page.Title,
			Content:  newContent,
		},
	}
	var resp ds.EntityChangeRequest
	UPDATE(t, pf("/pages/%s/", page.PublicID), req, &resp)

	assert.Equal(t, 1, resp.Revision)
	assert.Equal(t, ds.EntityChangeCommitted, resp.Status)

	newContentHTML, err := app.MarkdownToHTML(newContent)
	test.CheckErr(t, err)

	// page should be updated
	test.AssertInDB(t, tt.DB, "pages", test.Data{
		"id":          page.ID,
		"content_raw": newContent,
		"content":     newContentHTML,
	})

	contentPatch := app.MakePatch(page.ContentRaw, newContent)
	test.AssertInDB(t, tt.DB, "entity_change_requests", test.Data{
		"user_id":   admin.ID,
		"entity_id": page.ID,
		"status":    ds.EntityChangeCommitted,
		"revision":  1,
		"diff":      map[string]any{"content": contentPatch},
	})
}
