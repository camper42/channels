package web

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"channels/storage"
)

// alertManagerMessage from:
// https://prometheus.io/docs/alerting/configuration/#webhook_config
// https://github.com/prometheus/alertmanager/blob/66a0ed21bdb0720b4ba083d35acd6ae77fa7b0b5/template/template.go#L227
type alertManagerMessage struct {
	Version           string
	GroupKey          string
	Status            string
	Receiver          string
	GroupLabels       map[string]string
	CommonLabels      map[string]string
	CommonAnnotations map[string]string
	ExternalURL       string
	Alerts            []struct {
		Status       string
		Labels       map[string]string
		Annotations  map[string]string
		StartsAt     time.Time
		EndsAt       time.Time
		GeneratorURL string
		Fingerprint  string
	}
}

// webhookAlertManager handles request from alertmanager as a webhook
func (e *env) webhookAlertManager(c *gin.Context) {
	caller, ok := e.checkToken(c)
	if !ok {
		return
	}
	if len(caller.Caps) != 1 {
		c.AbortWithStatusJSON(500, gin.H{"error": "caps invalid"})
		return
	}

	var msg alertManagerMessage
	if err := c.BindJSON(&msg); err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": err.Error()})
		return
	}
	var text string
	switch msg.Status {
	case "firing":
		text = "🔥"
	case "resolved":
		text = "✅"
	}

	text += fmt.Sprintf("[%s:%s] %s %s: %s\n",
		msg.Status, msg.Version,
		msg.GroupLabels["alertname"],
		msg.CommonLabels["severity"],
		msg.CommonAnnotations["summary"],
	)
	var labels []string
	for k, v := range msg.CommonLabels {
		if k == "alertname" || k == "severity" {
			continue
		}
		labels = append(labels, k+"="+v)
	}
	text += "labels{" + strings.Join(labels, ",") + "}"

	m := storage.Message{
		From:      caller.Name,
		To:        caller.Caps[0],
		Text:      text,
		Timestamp: time.Now().UnixNano(),
	}

	if err := e.store.Save(&m); err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "success"})
}