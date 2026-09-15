package controller_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ajuda-dev/backend/src/config/job"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationInbox_Unauthorized(t *testing.T) {
	app := setupApp()
	req, _ := http.NewRequest(http.MethodGet, "/v1/notifications", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestNotificationInbox_ListFiltersAndIsolation(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanUsersTable()
	})
	userA, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "inbox a", Email: "inbox_a_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	userB, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "inbox b", Email: "inbox_b_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)

	unreadA := createInboxOutbox(t, userA.Id, domain.OutboxTypeCommunityEventPendingApproval, "unread-a")
	readA := createInboxOutbox(t, userA.Id, domain.OutboxTypeMentoringInvitePending, "read-a")
	createInboxOutbox(t, userB.Id, domain.OutboxTypeCommunityEventPendingApproval, "b-only")
	createInboxOutbox(t, userA.Id, "PASSWORD_RESET", "email-hidden")
	_, markErr := outboxEventRepository.MarkRead(readA.Id, userA.Id)
	require.Nil(t, markErr)

	app := setupApp()
	page := listNotifications(t, app, validTokenFor(t, userA.Id), "")
	require.Len(t, page.Data, 1)
	assert.False(t, page.HasNext)
	assert.Equal(t, unreadA.Id, page.Data[0].Id)
	assert.Equal(t, domain.OutboxTypeCommunityEventPendingApproval, page.Data[0].Type)
	assert.Nil(t, page.Data[0].ReadAt)
	assert.JSONEq(t, `{"title":"unread-a"}`, string(page.Data[0].Payload))

	readPage := listNotifications(t, app, validTokenFor(t, userA.Id), "status=read")
	require.Len(t, readPage.Data, 1)
	assert.Equal(t, readA.Id, readPage.Data[0].Id)
	require.NotNil(t, readPage.Data[0].ReadAt)

	allPage := listNotifications(t, app, validTokenFor(t, userA.Id), "status=all")
	require.Len(t, allPage.Data, 2)

	bPage := listNotifications(t, app, validTokenFor(t, userB.Id), "status=unread")
	require.Len(t, bPage.Data, 1)
	assert.NotEqual(t, unreadA.Id, bPage.Data[0].Id)
}

func TestNotificationInbox_InvalidStatus(t *testing.T) {
	t.Cleanup(cleanUsersTable)
	user, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "inbox status", Email: "inbox_st_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	app := setupApp()
	req, _ := http.NewRequest(http.MethodGet, "/v1/notifications?status=foo", nil)
	req.Header.Set("Authorization", "Bearer "+validTokenFor(t, user.Id))
	resp, testErr := app.Test(req)
	require.NoError(t, testErr)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestNotificationInbox_Pagination(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanUsersTable()
	})
	user, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "inbox page", Email: "inbox_p_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	first := createInboxOutbox(t, user.Id, domain.OutboxTypeCommunityEventPendingApproval, "older")
	time.Sleep(5 * time.Millisecond)
	second := createInboxOutbox(t, user.Id, domain.OutboxTypeMentoringInvitePending, "newer")

	app := setupApp()
	token := validTokenFor(t, user.Id)
	pageOne := listNotifications(t, app, token, "limit=1&page=1")
	require.Len(t, pageOne.Data, 1)
	assert.True(t, pageOne.HasNext)
	assert.Equal(t, second.Id, pageOne.Data[0].Id)

	pageTwo := listNotifications(t, app, token, "limit=1&page=2")
	require.Len(t, pageTwo.Data, 1)
	assert.False(t, pageTwo.HasNext)
	assert.Equal(t, first.Id, pageTwo.Data[0].Id)
}

func TestNotificationInbox_OfflineStillListedAfterSent(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanUsersTable()
	})
	user, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "inbox offline", Email: "inbox_off_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	event := createInboxOutbox(t, user.Id, domain.OutboxTypeCommunityEventPendingApproval, "offline")
	job.RunOutboxOnce(outboxEventRepository, job.NewSSEOutboxHandler(nil), 50)

	var stored entity.OutboxEventEntity
	require.NoError(t, db.First(&stored, event.Id).Error)
	assert.Equal(t, domain.OutboxStatusSent, stored.Status)
	assert.Nil(t, stored.ReadAt)

	app := setupApp()
	page := listNotifications(t, app, validTokenFor(t, user.Id), "status=unread")
	require.Len(t, page.Data, 1)
	assert.Equal(t, event.Id, page.Data[0].Id)
}

func TestNotificationInbox_MarkRead(t *testing.T) {
	t.Cleanup(func() {
		db.Exec("DELETE FROM outbox_events")
		cleanUsersTable()
	})
	userA, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "inbox read a", Email: "inbox_ra_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	userB, err := userRepository.CreateUser(&domain.UserDomain{
		Name: "inbox read b", Email: "inbox_rb_" + uuidv7.New().String() + "@ajuda.dev", Password: "123456",
	})
	require.Nil(t, err)
	item := createInboxOutbox(t, userA.Id, domain.OutboxTypeCommunityEventPendingApproval, "to-read")
	other := createInboxOutbox(t, userB.Id, domain.OutboxTypeCommunityEventPendingApproval, "other")

	app := setupApp()
	tokenA := validTokenFor(t, userA.Id)

	first := markNotificationRead(t, app, tokenA, fmt.Sprintf("%d", item.Id))
	assert.Equal(t, fiber.StatusOK, first.status)
	require.NotNil(t, first.body.ReadAt)
	readAt := *first.body.ReadAt
	assert.Equal(t, domain.OutboxStatusPending, workerStatus(t, item.Id))

	unread := listNotifications(t, app, tokenA, "status=unread")
	assert.Empty(t, unread.Data)

	second := markNotificationRead(t, app, tokenA, fmt.Sprintf("%d", item.Id))
	assert.Equal(t, fiber.StatusOK, second.status)
	require.NotNil(t, second.body.ReadAt)
	assert.Equal(t, readAt, *second.body.ReadAt)

	foreign := markNotificationRead(t, app, tokenA, fmt.Sprintf("%d", other.Id))
	assert.Equal(t, fiber.StatusNotFound, foreign.status)

	badId := markNotificationRead(t, app, tokenA, "not-a-number")
	assert.Equal(t, fiber.StatusBadRequest, badId.status)

	req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/v1/notifications/%d/read", item.Id), nil)
	resp, testErr := app.Test(req)
	require.NoError(t, testErr)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func createInboxOutbox(t *testing.T, userId, outboxType, title string) *domain.OutboxEventDomain {
	t.Helper()
	payload, _ := json.Marshal(map[string]string{"title": title})
	event := &domain.OutboxEventDomain{
		Type:    outboxType,
		UserId:  userId,
		Payload: payload,
		Status:  domain.OutboxStatusPending,
	}
	require.Nil(t, outboxEventRepository.Create(nil, event))
	return event
}

func listNotifications(t *testing.T, app *fiber.App, token, query string) dto.PageableNotificationDto {
	t.Helper()
	url := "/v1/notifications"
	if query != "" {
		url += "?" + query
	}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	var page dto.PageableNotificationDto
	require.NoError(t, json.Unmarshal(body, &page))
	if page.Data == nil {
		page.Data = []dto.NotificationDtoOut{}
	}
	return page
}

type markReadResult struct {
	status int
	body   dto.NotificationDtoOut
}

func markNotificationRead(t *testing.T, app *fiber.App, token, id string) markReadResult {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPut, "/v1/notifications/"+id+"/read", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	result := markReadResult{status: resp.StatusCode}
	if resp.StatusCode == fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		require.NoError(t, json.Unmarshal(body, &result.body))
	}
	return result
}

func workerStatus(t *testing.T, id int64) string {
	t.Helper()
	var stored entity.OutboxEventEntity
	require.NoError(t, db.First(&stored, id).Error)
	return stored.Status
}
