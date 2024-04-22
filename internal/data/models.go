package data

import (
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrDocNotFound  = errors.New("document not found")
	ErrEditConflict = errors.New("edit conflict")
)

type Models struct {
	Memes MemeModel
}

func NewModels(db *mongo.Client) Models {
	return Models{
		Memes: MemeModel{DB: db},
	}
}
