// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/optional"
	"code.gitea.io/gitea/modules/timeutil"
	webhook_module "code.gitea.io/gitea/modules/webhook"

	"github.com/stretchr/testify/assert"
)

func TestHookContentType_Name(t *testing.T) {
	assert.Equal(t, "json", ContentTypeJSON.Name())
	assert.Equal(t, "form", ContentTypeForm.Name())
}

func TestIsValidHookContentType(t *testing.T) {
	assert.True(t, IsValidHookContentType("json"))
	assert.True(t, IsValidHookContentType("form"))
	assert.False(t, IsValidHookContentType("invalid"))
}

func TestWebhook_History(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	hook1 := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/history1",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
		IsActive:    true,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook1))
	task1, err := CreateHookTask(t.Context(), &HookTask{HookID: hook1.ID, IsDelivered: true, PayloadVersion: 2})
	assert.NoError(t, err)
	task2, err := CreateHookTask(t.Context(), &HookTask{HookID: hook1.ID, IsDelivered: true, PayloadVersion: 2})
	assert.NoError(t, err)
	task3, err := CreateHookTask(t.Context(), &HookTask{HookID: hook1.ID, IsDelivered: true, PayloadVersion: 2})
	assert.NoError(t, err)

	tasks, err := hook1.History(t.Context(), 0)
	assert.NoError(t, err)
	if assert.Len(t, tasks, 3) {
		assert.Equal(t, task3.ID, tasks[0].ID)
		assert.Equal(t, task2.ID, tasks[1].ID)
		assert.Equal(t, task1.ID, tasks[2].ID)
	}

	hook2 := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/history2",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
		IsActive:    false,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook2))
	tasks, err = hook2.History(t.Context(), 0)
	assert.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestWebhook_UpdateEvent(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	webhook := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/update_event",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
	}
	assert.NoError(t, CreateWebhook(t.Context(), webhook))
	hookEvent := &webhook_module.HookEvent{
		PushOnly:       true,
		SendEverything: false,
		ChooseEvents:   false,
		HookEvents: webhook_module.HookEvents{
			webhook_module.HookEventCreate:      false,
			webhook_module.HookEventPush:        true,
			webhook_module.HookEventPullRequest: false,
		},
	}
	webhook.HookEvent = hookEvent
	assert.NoError(t, webhook.UpdateEvent())
	assert.NotEmpty(t, webhook.Events)
	actualHookEvent := &webhook_module.HookEvent{}
	assert.NoError(t, json.Unmarshal([]byte(webhook.Events), actualHookEvent))
	assert.Equal(t, *hookEvent, *actualHookEvent)
}

func TestWebhook_EventsArray(t *testing.T) {
	assert.Equal(t, []string{
		"create", "delete", "fork", "push",
		"issues", "issue_assign", "issue_label", "issue_milestone", "issue_comment",
		"pull_request", "pull_request_assign", "pull_request_label", "pull_request_milestone",
		"pull_request_comment", "pull_request_review_approved", "pull_request_review_rejected",
		"pull_request_review_comment", "pull_request_sync", "pull_request_review_request", "wiki", "repository", "release",
		"package", "status", "workflow_run", "workflow_job",
	},
		(&Webhook{
			HookEvent: &webhook_module.HookEvent{SendEverything: true},
		}).EventsArray(),
	)

	assert.Equal(t, []string{"push"},
		(&Webhook{
			HookEvent: &webhook_module.HookEvent{PushOnly: true},
		}).EventsArray(),
	)
}

func TestCreateWebhook(t *testing.T) {
	hook := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/unit_test",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":false,"send_everything":false,"choose_events":false,"events":{"create":false,"push":true,"pull_request":true}}`,
	}
	unittest.AssertNotExistsBean(t, hook)
	assert.NoError(t, CreateWebhook(t.Context(), hook))
	unittest.AssertExistsAndLoadBean(t, hook)
}

func TestGetWebhookByRepoID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		RepoID:      1,
		URL:         "https://www.example.com/by_repo_id",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))

	loaded, err := GetWebhookByRepoID(t.Context(), 1, hook.ID)
	assert.NoError(t, err)
	assert.Equal(t, hook.ID, loaded.ID)

	_, err = GetWebhookByRepoID(t.Context(), unittest.NonexistentID, unittest.NonexistentID)
	assert.Error(t, err)
	assert.True(t, IsErrWebhookNotExist(err))
}

func TestGetWebhookByOwnerID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		OwnerID:     3,
		URL:         "https://www.example.com/by_owner_id",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
		IsActive:    true,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))

	loaded, err := GetWebhookByOwnerID(t.Context(), 3, hook.ID)
	assert.NoError(t, err)
	assert.Equal(t, hook.ID, loaded.ID)

	_, err = GetWebhookByOwnerID(t.Context(), unittest.NonexistentID, unittest.NonexistentID)
	assert.Error(t, err)
	assert.True(t, IsErrWebhookNotExist(err))
}

func TestGetActiveWebhooksByRepoID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook1 := &Webhook{
		RepoID:      1,
		URL:         "https://www.example.com/active1",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
		IsActive:    true,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook1))
	hook2 := &Webhook{
		RepoID:      1,
		URL:         "https://www.example.com/inactive1",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
		IsActive:    false,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook2))

	hooks, err := db.Find[Webhook](t.Context(), ListWebhookOptions{RepoID: 1, IsActive: optional.Some(true)})
	assert.NoError(t, err)
	if assert.Len(t, hooks, 1) {
		assert.Equal(t, hook1.ID, hooks[0].ID)
		assert.True(t, hooks[0].IsActive)
	}
}

func TestGetWebhooksByRepoID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook1 := &Webhook{
		RepoID:      1,
		URL:         "https://www.example.com/repo1_hook1",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
		IsActive:    true,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook1))
	hook2 := &Webhook{
		RepoID:      1,
		URL:         "https://www.example.com/repo1_hook2",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
		IsActive:    false,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook2))

	hooks, err := db.Find[Webhook](t.Context(), ListWebhookOptions{RepoID: 1})
	assert.NoError(t, err)
	if assert.Len(t, hooks, 2) {
		assert.Equal(t, hook1.ID, hooks[0].ID)
		assert.Equal(t, hook2.ID, hooks[1].ID)
	}
}

func TestGetActiveWebhooksByOwnerID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		OwnerID:     3,
		URL:         "https://www.example.com/owner_active",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
		IsActive:    true,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))

	hooks, err := db.Find[Webhook](t.Context(), ListWebhookOptions{OwnerID: 3, IsActive: optional.Some(true)})
	assert.NoError(t, err)
	if assert.Len(t, hooks, 1) {
		assert.Equal(t, hook.ID, hooks[0].ID)
		assert.True(t, hooks[0].IsActive)
	}
}

func TestGetWebhooksByOwnerID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		OwnerID:     3,
		URL:         "https://www.example.com/owner_hook",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
		IsActive:    true,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))

	hooks, err := db.Find[Webhook](t.Context(), ListWebhookOptions{OwnerID: 3})
	assert.NoError(t, err)
	if assert.Len(t, hooks, 1) {
		assert.Equal(t, hook.ID, hooks[0].ID)
		assert.True(t, hooks[0].IsActive)
	}
}

func TestUpdateWebhook(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		RepoID:      1,
		URL:         "https://www.example.com/update",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
		IsActive:    false,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))
	hook.IsActive = true
	hook.ContentType = ContentTypeForm
	unittest.AssertNotExistsBean(t, hook)
	assert.NoError(t, UpdateWebhook(t.Context(), hook))
	unittest.AssertExistsAndLoadBean(t, hook)
}

func TestDeleteWebhookByRepoID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		RepoID:      1,
		URL:         "https://www.example.com/delete_by_repo",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))
	unittest.AssertExistsAndLoadBean(t, &Webhook{ID: hook.ID, RepoID: 1})
	assert.NoError(t, DeleteWebhookByRepoID(t.Context(), 1, hook.ID))
	unittest.AssertNotExistsBean(t, &Webhook{ID: hook.ID, RepoID: 1})

	err := DeleteWebhookByRepoID(t.Context(), unittest.NonexistentID, unittest.NonexistentID)
	assert.Error(t, err)
	assert.True(t, IsErrWebhookNotExist(err))
}

func TestDeleteWebhookByOwnerID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		OwnerID:     3,
		URL:         "https://www.example.com/delete_by_owner",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))
	unittest.AssertExistsAndLoadBean(t, &Webhook{ID: hook.ID, OwnerID: 3})
	assert.NoError(t, DeleteWebhookByOwnerID(t.Context(), 3, hook.ID))
	unittest.AssertNotExistsBean(t, &Webhook{ID: hook.ID, OwnerID: 3})

	err := DeleteWebhookByOwnerID(t.Context(), unittest.NonexistentID, unittest.NonexistentID)
	assert.Error(t, err)
	assert.True(t, IsErrWebhookNotExist(err))
}

func TestHookTasks(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/hook_tasks",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))
	task1, err := CreateHookTask(t.Context(), &HookTask{HookID: hook.ID, IsDelivered: true, PayloadVersion: 2})
	assert.NoError(t, err)
	task2, err := CreateHookTask(t.Context(), &HookTask{HookID: hook.ID, IsDelivered: true, PayloadVersion: 2})
	assert.NoError(t, err)
	task3, err := CreateHookTask(t.Context(), &HookTask{HookID: hook.ID, IsDelivered: true, PayloadVersion: 2})
	assert.NoError(t, err)

	hookTasks, err := HookTasks(t.Context(), hook.ID, 1)
	assert.NoError(t, err)
	if assert.Len(t, hookTasks, 3) {
		assert.Equal(t, task3.ID, hookTasks[0].ID)
		assert.Equal(t, task2.ID, hookTasks[1].ID)
		assert.Equal(t, task1.ID, hookTasks[2].ID)
	}

	hookTasks, err = HookTasks(t.Context(), unittest.NonexistentID, 1)
	assert.NoError(t, err)
	assert.Empty(t, hookTasks)
}

func TestCreateHookTask(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/create_task",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))
	hookTask := &HookTask{
		HookID:         hook.ID,
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(t.Context(), hookTask)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)
}

func TestUpdateHookTask(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/update_task",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))
	hookTask := &HookTask{HookID: hook.ID, PayloadVersion: 2}
	_, err := CreateHookTask(t.Context(), hookTask)
	assert.NoError(t, err)

	hookTask.PayloadContent = "new payload content"
	hookTask.IsDelivered = true
	unittest.AssertNotExistsBean(t, hookTask)
	assert.NoError(t, UpdateHookTask(t.Context(), hookTask))
	unittest.AssertExistsAndLoadBean(t, hookTask)
}

func TestCleanupHookTaskTable_PerWebhook_DeletesDelivered(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/cleanup1",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))
	hookTask := &HookTask{
		HookID:         hook.ID,
		IsDelivered:    true,
		Delivered:      timeutil.TimeStampNanoNow(),
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(t.Context(), hookTask)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)

	assert.NoError(t, CleanupHookTaskTable(t.Context(), PerWebhook, 168*time.Hour, 0))
	unittest.AssertNotExistsBean(t, hookTask)
}

func TestCleanupHookTaskTable_PerWebhook_LeavesUndelivered(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/cleanup2",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))
	hookTask := &HookTask{
		HookID:         hook.ID,
		IsDelivered:    false,
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(t.Context(), hookTask)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)

	assert.NoError(t, CleanupHookTaskTable(t.Context(), PerWebhook, 168*time.Hour, 0))
	unittest.AssertExistsAndLoadBean(t, hookTask)
}

func TestCleanupHookTaskTable_PerWebhook_LeavesMostRecentTask(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/cleanup3",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))
	hookTask := &HookTask{
		HookID:         hook.ID,
		IsDelivered:    true,
		Delivered:      timeutil.TimeStampNanoNow(),
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(t.Context(), hookTask)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)

	assert.NoError(t, CleanupHookTaskTable(t.Context(), PerWebhook, 168*time.Hour, 1))
	unittest.AssertExistsAndLoadBean(t, hookTask)
}

func TestCleanupHookTaskTable_OlderThan_DeletesDelivered(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/cleanup4",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))
	hookTask := &HookTask{
		HookID:         hook.ID,
		IsDelivered:    true,
		Delivered:      timeutil.TimeStampNano(time.Now().AddDate(0, 0, -8).UnixNano()),
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(t.Context(), hookTask)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)

	assert.NoError(t, CleanupHookTaskTable(t.Context(), OlderThan, 168*time.Hour, 0))
	unittest.AssertNotExistsBean(t, hookTask)
}

func TestCleanupHookTaskTable_OlderThan_LeavesUndelivered(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/cleanup5",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))
	hookTask := &HookTask{
		HookID:         hook.ID,
		IsDelivered:    false,
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(t.Context(), hookTask)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)

	assert.NoError(t, CleanupHookTaskTable(t.Context(), OlderThan, 168*time.Hour, 0))
	unittest.AssertExistsAndLoadBean(t, hookTask)
}

func TestCleanupHookTaskTable_OlderThan_LeavesTaskEarlierThanAgeToDelete(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	hook := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/cleanup6",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":true}`,
	}
	assert.NoError(t, CreateWebhook(t.Context(), hook))
	hookTask := &HookTask{
		HookID:         hook.ID,
		IsDelivered:    true,
		Delivered:      timeutil.TimeStampNano(time.Now().AddDate(0, 0, -6).UnixNano()),
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(t.Context(), hookTask)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)

	assert.NoError(t, CleanupHookTaskTable(t.Context(), OlderThan, 168*time.Hour, 0))
	unittest.AssertExistsAndLoadBean(t, hookTask)
}
