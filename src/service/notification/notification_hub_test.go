package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotificationHub_PublishToSubscribers(t *testing.T) {
	hub := NewNotificationHub()
	ch1, cancel1 := hub.Subscribe("u1")
	ch2, cancel2 := hub.Subscribe("u1")
	defer cancel1()
	defer cancel2()

	hub.Publish("u1", []byte(`{"ok":true}`))
	assert.Equal(t, []byte(`{"ok":true}`), <-ch1)
	assert.Equal(t, []byte(`{"ok":true}`), <-ch2)
}

func TestNotificationHub_CancelRemovesChannel(t *testing.T) {
	hub := NewNotificationHub()
	ch, cancel := hub.Subscribe("u1")
	cancel()
	hub.Publish("u1", []byte("x"))
	_, ok := <-ch
	assert.False(t, ok)
}
