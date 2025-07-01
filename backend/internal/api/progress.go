
package api

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"time"
)

func ProgressUpdates(c *websocket.Conn) {
	for {
		// Simulate progress updates
		time.Sleep(2 * time.Second)
		err := c.WriteMessage(fiber.TextMessage, []byte(fmt.Sprintf("Processing... %d%%", time.Now().Second())))
		if err != nil {
			break
		}
	}
}
