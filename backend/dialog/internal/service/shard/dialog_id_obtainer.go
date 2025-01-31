package shard

import (
	"context"
	"fmt"

	"github.com/olga-larina/otus-highload/dialog/internal/model"
	"github.com/spaolacci/murmur3"
)

type DialogIdObtainer struct {
}

func NewDialogIdObtainer() *DialogIdObtainer {
	return &DialogIdObtainer{}
}

func (d *DialogIdObtainer) ObtainDialogId(ctx context.Context, firstUserId *model.UserId, secondUserId *model.UserId) (*model.DialogId, error) {
	var combinedId string
	if *firstUserId > *secondUserId {
		combinedId = *secondUserId + *firstUserId
	} else {
		combinedId = *firstUserId + *secondUserId
	}
	h1, h2 := murmur3.Sum128([]byte(combinedId))
	hash := fmt.Sprintf("%016x%016x", h1, h2)
	return &hash, nil
}
