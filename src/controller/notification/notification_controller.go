package notification

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/middleware"
	notificationdto "github.com/ajuda-dev/backend/src/controller/notification/dto"
	"github.com/ajuda-dev/backend/src/service/notification"
	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

type NotificationController interface {
	Stream() fiber.Handler
	List() fiber.Handler
	MarkRead() fiber.Handler
}

type notificationController struct {
	hub     notification.NotificationHub
	service notification.NotificationService
}

func NewNotificationController(hub notification.NotificationHub, notificationService notification.NotificationService) NotificationController {
	return &notificationController{hub: hub, service: notificationService}
}

// Stream godoc
// @Summary      Stream de notificações (SSE)
// @Description  Abre uma conexão Server-Sent Events para o usuário autenticado. O destinatário é sempre o JWT; user_id na query não é aceito.
// @Tags         notifications
// @Produce      text/event-stream
// @Success      200  {string}  string
// @Failure      401  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/notifications/stream [get]
func (n *notificationController) Stream() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userId, _ := c.Locals(middleware.UserIdKey).(string)
		if userId == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing or invalid Authorization header"))
		}

		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("X-Accel-Buffering", "no")

		reqCtx := c.Context()
		ch, cancel := n.hub.Subscribe(userId)
		reqCtx.SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
			defer cancel()
			defer reqCtx.SetConnectionClose()

			clientClosed := make(chan struct{})
			go waitClientDisconnect(reqCtx.Conn(), clientClosed)

			fmt.Fprintf(w, "event: ready\ndata: {\"ok\":true}\n\n")
			if err := w.Flush(); err != nil {
				return
			}
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-clientClosed:
					return
				case msg, ok := <-ch:
					if !ok {
						return
					}
					id := outboxIdFromPayload(msg)
					if id != "" {
						fmt.Fprintf(w, "event: notification\nid: %s\ndata: %s\n\n", id, msg)
					} else {
						fmt.Fprintf(w, "event: notification\ndata: %s\n\n", msg)
					}
					if err := w.Flush(); err != nil {
						return
					}
				case <-ticker.C:
					fmt.Fprintf(w, ": ping\n\n")
					if err := w.Flush(); err != nil {
						return
					}
				}
			}
		}))
		return nil
	}
}

// List godoc
// @Summary      Lista notificações do usuário
// @Description  Inbox paginada do usuário autenticado. Destinatário é sempre o JWT. status=unread (default), read ou all. Não aceita user_id na query.
// @Tags         notifications
// @Produce      json
// @Param        status  query  string  false  "unread, read ou all"  default(unread)
// @Param        page    query  int     false  "Página"
// @Param        limit   query  int     false  "Limite"
// @Success      200  {object}  notificationdto.PageableNotificationDto
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/notifications [get]
func (n *notificationController) List() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userId, _ := c.Locals(middleware.UserIdKey).(string)
		if userId == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing or invalid Authorization header"))
		}
		page, _ := strconv.Atoi(c.Query("page", "1"))
		limit, _ := strconv.Atoi(c.Query("limit", "10"))
		result, err := n.service.List(userId, c.Query("status", notificationdomain.NotificationInboxStatusUnread), page, limit)
		if err != nil {
			logger.Error("error: ", err)
			return c.Status(err.Code).JSON(err)
		}
		return c.Status(fiber.StatusOK).JSON(notificationdto.PageableNotificationDto{}.FromDomain(*result))
	}
}

// MarkRead godoc
// @Summary      Marca uma notificação como lida
// @Description  Idempotente. O id é o bigserial da outbox (o mesmo do SSE). 404 se não for do usuário autenticado ou não for tipo de sininho.
// @Tags         notifications
// @Produce      json
// @Param        id   path      int  true  "ID da notificação (outbox bigserial)"
// @Success      200  {object}  notificationdto.NotificationDtoOut
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/notifications/{id}/read [put]
func (n *notificationController) MarkRead() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userId, _ := c.Locals(middleware.UserIdKey).(string)
		if userId == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(rest_err.NewUnauthorizedError("missing or invalid Authorization header"))
		}
		id, parseErr := strconv.ParseInt(c.Params("id"), 10, 64)
		if parseErr != nil || id <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid path params",
				[]rest_err.Causes{{Field: "id", Message: "id must be a positive integer"}},
			))
		}
		result, err := n.service.MarkRead(id, userId)
		if err != nil {
			logger.Error("error: ", err)
			return c.Status(err.Code).JSON(err)
		}
		return c.Status(fiber.StatusOK).JSON(notificationdto.NotificationDtoOut{}.FromDomain(*result))
	}
}

func waitClientDisconnect(conn net.Conn, closed chan struct{}) {
	if conn == nil {
		return
	}
	defer close(closed)
	buf := make([]byte, 1)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			return
		}
	}
}

func outboxIdFromPayload(msg []byte) string {
	var envelope struct {
		Id json.Number `json:"id"`
	}
	if err := json.Unmarshal(msg, &envelope); err != nil {
		return ""
	}
	return envelope.Id.String()
}
