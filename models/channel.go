package models

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/parnurzeal/gorequest"
	"github.com/qwildz/webhook-adapter/parser"
	"gorm.io/gorm"
)

type Channel struct {
	gorm.Model
	Token  string
	Name   string
	Lang   string
	Script string `gorm:"type:text"`
	Target string
}

func (c *Channel) Run(ctx context.Context, payload string) (response, contentType string, err error) {
	start := time.Now()

	data, err := c.Transform(ctx, payload)
	elapsedTransform := time.Since(start)

	if err != nil {
		return "", "", err
	}

	responseHeader, response, errs := gorequest.New().
		Post(c.Target).
		Send(data).
		Retry(3, 5*time.Second, http.StatusBadRequest, http.StatusInternalServerError).
		End()
	elapsedSend := time.Since(start)
	contentType = responseHeader.Header.Get("content-type")

	if errs != nil {
		var errStrings []string
		for _, e := range errs {
			errStrings = append(errStrings, e.Error())
		}

		err = errors.New(strings.Join(errStrings, "; "))

		return
	}

	log.Printf("Sent to %s | transform: %s, total: %s", c.Target, elapsedTransform, elapsedSend)

	return
}

func (c *Channel) Transform(ctx context.Context, payload string) (string, error) {
	var data string
	var err error

	switch c.Lang {
	case "lua":
		data, err = parser.RunLua(ctx, c.ID, c.Script, payload)

	case "js":
		data, err = parser.RunJS(ctx, c.ID, c.Script, payload)

	default:
		return "", errors.New("language not supported")
	}

	if err != nil {
		return "", err
	}

	if !json.Valid([]byte(data)) {
		return "", errors.New("invalid json returned from transformer script")
	}

	return data, nil
}
