package converter

import (
	"encoding/json"

	"github.com/olga-larina/otus-highload/social/internal/model"
)

type SubscriberStringConverter struct {
}

func NewSubscriberStringConverter() *SubscriberStringConverter {
	return &SubscriberStringConverter{}
}

func (p *SubscriberStringConverter) ConvertToString(value any) (string, error) {
	bytes, err := json.Marshal(value.([]model.UserId))
	return string(bytes), err
}

func (p *SubscriberStringConverter) ConvertFromString(valueStr string) (any, error) {
	var subscribers []model.UserId
	err := json.Unmarshal([]byte(valueStr), &subscribers)
	return subscribers, err
}
