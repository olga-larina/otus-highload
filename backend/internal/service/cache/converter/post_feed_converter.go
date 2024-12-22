package converter

import (
	"encoding/json"

	"github.com/olga-larina/otus-highload/backend/internal/model"
)

type PostFeedStringConverter struct {
}

func NewPostFeedStringConverter() *PostFeedStringConverter {
	return &PostFeedStringConverter{}
}

func (p *PostFeedStringConverter) ConvertToString(value any) (string, error) {
	bytes, err := json.Marshal(value.([]model.PostExtended))
	return string(bytes), err
}

func (p *PostFeedStringConverter) ConvertFromString(valueStr string) (any, error) {
	var postFeed []model.PostExtended
	err := json.Unmarshal([]byte(valueStr), &postFeed)
	return postFeed, err
}
