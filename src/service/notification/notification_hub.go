package notification

import (
	"sync"

	"github.com/ajuda-dev/backend/src/config/logger"
	"go.uber.org/zap"
)

type NotificationHub interface {
	Subscribe(userId string) (ch <-chan []byte, cancel func())
	Publish(userId string, payload []byte)
}

type notificationHub struct {
	mu   sync.RWMutex
	subs map[string][]chan []byte
}

func NewNotificationHub() NotificationHub {
	return &notificationHub{subs: make(map[string][]chan []byte)}
}

func (h *notificationHub) Subscribe(userId string) (<-chan []byte, func()) {
	ch := make(chan []byte, 8)
	h.mu.Lock()
	h.subs[userId] = append(h.subs[userId], ch)
	h.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			list := h.subs[userId]
			for i, existing := range list {
				if existing == ch {
					h.subs[userId] = append(list[:i], list[i+1:]...)
					break
				}
			}
			if len(h.subs[userId]) == 0 {
				delete(h.subs, userId)
			}
			close(ch)
		})
	}
	return ch, cancel
}

func (h *notificationHub) Publish(userId string, payload []byte) {
	h.mu.RLock()
	list := append([]chan []byte(nil), h.subs[userId]...)
	h.mu.RUnlock()
	for _, ch := range list {
		select {
		case ch <- payload:
		default:
			logger.Info("notification hub drop", zap.String("user_id", userId))
		}
	}
}
