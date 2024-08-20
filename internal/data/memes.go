package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/michaelgov-ctrl/memebase/internal/validator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	dbName         = "memebase"
	collectionName = "memes"
)

type MemeModel struct {
	DB *mongo.Client
}

type Meme struct {
	ID      string    `json:"id" bson:"_id"` //"go.mongodb.org/mongo-driver/bson/primitive" ID     primitive.ObjectID
	Created time.Time `json:"created" bson:"created"`
	Artist  string    `json:"artist" bson:"artist"`
	Title   string    `json:"title" bson:"title"`
	B64     string    `json:"b64" bson:"b64"`
	Version int32     `json:"version" bson:"version"`
}

func (m *Meme) ToEditMeme() *MongoEditMeme {
	updateMeme := &MongoEditMeme{}

	updateMeme.Created = m.Created

	if m.Artist != "" {
		updateMeme.Artist = m.Artist
	}

	if m.Title != "" {
		updateMeme.Title = m.Title
	}

	if m.B64 != "" {
		updateMeme.B64 = m.B64
	}

	if m.Version != 0 {
		updateMeme.Version = m.Version
	}

	return updateMeme
}

func ValidateMeme(v *validator.Validator, meme *Meme) {
	v.Check(meme.Artist != "", "artist", "must be provided")
	v.Check(len(meme.Artist) <= 64, "artist", "must not be more than 64 bytes long")
	v.Check(meme.Title != "", "title", "must be provided")
	v.Check(len(meme.Title) <= 64, "title", "must not be more than 64 bytes long")
	v.Check(meme.B64 != "", "b64", "must be provided")
}

type MongoEditMeme struct {
	Artist  string    `bson:"artist"`
	Created time.Time `bson:"created"`
	Title   string    `bson:"title"`
	B64     string    `bson:"b64"`
	Version int32     `bson:"version"`
}

func (m MemeModel) Insert(meme *Meme) error {
	meme.Created, meme.Version = time.Now(), 1

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := m.DB.Database(dbName).Collection(collectionName).InsertOne(ctx, meme.ToEditMeme())
	if err != nil {
		return err
	}

	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		panic(fmt.Sprintf("insert operation returned unexpected value %v", id))
	}

	meme.ID = id.Hex()

	return nil
}

func (m MemeModel) Get(id string) (*Meme, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrDocNotFound
	}

	var meme Meme

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := m.DB.Database(dbName).Collection(collectionName).FindOne(ctx, bson.D{{"_id", objID}}).Decode(&meme); err != nil {
		switch {
		case errors.Is(err, mongo.ErrNoDocuments):
			return nil, ErrDocNotFound
		default:
			return nil, err
		}
	}

	meme.ID = id
	return &meme, nil
}

func (m MemeModel) GetAll(artist, title string, filters Filters) ([]*Meme, Metadata, error) {
	match, metadata := GetAllFilter(artist, title), Metadata{}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	count, err := m.DB.Database(dbName).Collection(collectionName).CountDocuments(ctx, match)
	if err != nil {
		return nil, metadata, err
	}

	pipeline := filters.GetAllAggregationPipeline(match)
	cursor, err := m.DB.Database(dbName).Collection(collectionName).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, metadata, err
	}

	defer cursor.Close(context.TODO())

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	memes := []*Meme{}
	if err := cursor.All(ctx, &memes); err != nil {
		return nil, metadata, err
	}

	if err := cursor.Err(); err != nil {
		return nil, metadata, err
	}

	metadata.Calculate(len(memes), int(count), filters.Page, filters.PageSize)

	return memes, metadata, nil
}

func (m MemeModel) GetRandom() (*Meme, error) {
	aggStage := bson.D{{"$sample", bson.D{{"size", 1}}}}
	opts := options.Aggregate().SetMaxTime(3 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	cursor, err := m.DB.Database(dbName).Collection(collectionName).Aggregate(ctx, mongo.Pipeline{aggStage}, opts)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(context.TODO())

	memes := []*Meme{}
	if err := cursor.All(ctx, &memes); err != nil {
		return nil, err
	}

	return memes[0], nil
}

func (m MemeModel) Update(meme *Meme) error {
	objID, err := primitive.ObjectIDFromHex(meme.ID)
	if err != nil {
		return ErrDocNotFound
	}

	filter := bson.D{{"_id", objID}, {"version", meme.Version}}

	meme.Version += 1
	updateMeme := meme.ToEditMeme()
	update := bson.M{"$set": updateMeme}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := m.DB.Database(dbName).Collection(collectionName).UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return ErrEditConflict
	}

	return nil
}

func (m MemeModel) Delete(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrDocNotFound
	}

	filter := bson.D{{"_id", objID}}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := m.DB.Database(dbName).Collection(collectionName).DeleteOne(ctx, filter, nil)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return ErrDocNotFound
	}

	return nil
}

func GetAllFilter(artist, title string) bson.M {
	filter := bson.M{}

	if artist != "" {
		filter["artist"] = artist
	}

	if title != "" {
		filter["title"] = bson.M{"$regex": primitive.Regex{Pattern: title, Options: "i"}}
	}

	return filter
}

func (f *Filters) GetAllAggregationPipeline(match bson.M) mongo.Pipeline {
	matchStage := bson.D{{"$match", match}}

	sortField, sortDirection := f.sortField(), f.sortDirection()
	sortStage := bson.D{{"$sort", bson.D{{sortField, sortDirection}, {"_id", 1}}}}

	skipStage := bson.D{{"$skip", f.offset()}}

	limitStage := bson.D{{"$limit", f.limit()}}

	return mongo.Pipeline{matchStage, sortStage, skipStage, limitStage}
}
