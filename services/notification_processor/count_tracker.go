package notification_processor

import (
	"context"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	countCollection = "notification_counts"
	trackerBufSize  = 512
)

type countEvent struct {
	ChannelType string
	Status      string
}

type CountTracker struct {
	events chan countEvent
	mongo  *mongo.Client
	logger *utility.Logger
}

func NewCountTracker(mongoClient *mongo.Client, logger *utility.Logger) *CountTracker {
	return &CountTracker{
		events: make(chan countEvent, trackerBufSize),
		mongo:  mongoClient,
		logger: logger,
	}
}

func (ct *CountTracker) Track(channelType, status string) {
	if ct == nil || ct.mongo == nil {
		return
	}
	select {
	case ct.events <- countEvent{ChannelType: channelType, Status: status}:
	default:
	}
}

func (ct *CountTracker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-ct.events:
			if err := ct.increment(ev); err != nil {
				ct.logger.Error("count tracker: mongo upsert failed: %v", err)
			}
		}
	}
}

func (ct *CountTracker) increment(ev countEvent) error {
	tCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	today := time.Now().UTC().Format("2006-01-02")
	col := ct.mongo.Database(config.Config.MongoDB.DB_Name).Collection(countCollection)

	filter := bson.M{"date": today, "channel_type": ev.ChannelType}
	update := bson.M{"$inc": bson.M{"total": 1, ev.Status: 1}}

	_, err := col.UpdateOne(tCtx, filter, update, options.Update().SetUpsert(true))
	return err
}

func EnsureCountIndex(mongoClient *mongo.Client) error {
	if mongoClient == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	col := mongoClient.Database(config.Config.MongoDB.DB_Name).Collection(countCollection)
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "date", Value: 1}, {Key: "channel_type", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}
