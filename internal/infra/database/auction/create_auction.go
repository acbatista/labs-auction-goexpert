package auction

import (
	"context"
	"fmt"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}

type AuctionRepository struct {
	Collection      *mongo.Collection
	auctionInterval time.Duration
	stopChan        chan struct{}
	wg              sync.WaitGroup
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	ar := &AuctionRepository{
		Collection:      database.Collection("auctions"),
		auctionInterval: getAuctionInterval(),
		stopChan:        make(chan struct{}),
	}

	ar.startAuctionCloser()
	return ar
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {

	// Ensure auction has a valid timestamp
	if auctionEntity.Timestamp.IsZero() {
		auctionEntity.Timestamp = time.Now()
	}

	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auction_entity.Active, // Ensure auction starts as active
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}

	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	// Trigger immediate check for expired auctions
	go ar.closeExpiredAuctions(context.Background())

	return nil
}

func (ar *AuctionRepository) startAuctionCloser() {
	ar.wg.Add(1)
	go func() {
		defer ar.wg.Done()
		// Check every minute for expired auctions
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		logger.Info("Starting auction closer routine")
		// Initial check on startup
		ar.closeExpiredAuctions(context.Background())

		for {
			select {
			case <-ar.stopChan:
				logger.Info("Stopping auction closer routine")
				return
			case <-ticker.C:
				ar.closeExpiredAuctions(context.Background())
			}
		}
	}()
}

func (ar *AuctionRepository) closeExpiredAuctions(ctx context.Context) {
	now := time.Now()
	expirationTime := now.Add(-ar.auctionInterval).Unix()

	filter := bson.M{
		"status":    auction_entity.Active,
		"timestamp": bson.M{"$lte": expirationTime},
	}

	update := bson.M{
		"$set": bson.M{
			"status": auction_entity.Completed,
		},
	}

	result, err := ar.Collection.UpdateMany(ctx, filter, update)
	if err != nil {
		logger.Error("Error trying to close expired auctions", err)
		return
	}

	if result.ModifiedCount > 0 {
		logger.Info(
			fmt.Sprintf("Closed expired auctions - Modified count: %d",
				result.ModifiedCount),
		)
	}
}

func (ar *AuctionRepository) Stop() {
	close(ar.stopChan)
	ar.wg.Wait()
}

func getAuctionInterval() time.Duration {
	auctionInterval := os.Getenv("AUCTION_INTERVAL")
	if auctionInterval == "" {
		logger.Info("AUCTION_INTERVAL not set, using default of 5 minutes")
		return 5 * time.Minute
	}

	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		logger.Error("Invalid AUCTION_INTERVAL format, using default of 5 minutes", err)
		return 5 * time.Minute
	}

	if duration < time.Minute {
		logger.Info("AUCTION_INTERVAL too short, using minimum of 1 minute")
		return time.Minute
	}

	return duration
}
