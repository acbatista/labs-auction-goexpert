package auction

import (
	"context"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupTestDB(t *testing.T) (*mongo.Database, func()) {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatal(err)
	}

	dbName := "auction_test_" + time.Now().Format("20060102150405")
	db := client.Database(dbName)

	return db, func() {
		if err := db.Drop(ctx); err != nil {
			t.Error(err)
		}
		if err := client.Disconnect(ctx); err != nil {
			t.Error(err)
		}
	}
}

func TestAutomaticAuctionClosing(t *testing.T) {
	// Set a short auction interval for testing
	os.Setenv("AUCTION_INTERVAL", "2s")
	defer os.Unsetenv("AUCTION_INTERVAL")

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAuctionRepository(db)
	defer repo.Stop()

	ctx := context.Background()

	// Create a test auction
	auction := &auction_entity.Auction{
		Id:          "test-auction",
		ProductName: "Test Product",
		Category:    "Test Category",
		Description: "Test Description",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		Timestamp:   time.Now().Add(-3 * time.Second), // Set timestamp in the past
	}

	err := repo.CreateAuction(ctx, auction)
	assert.Nil(t, err)

	// Wait for the auction to be closed automatically
	time.Sleep(3 * time.Second)

	// Check if the auction was closed
	var result AuctionEntityMongo
	mongoErr := repo.Collection.FindOne(ctx, bson.M{"_id": auction.Id}).Decode(&result)
	assert.NoError(t, mongoErr)
	assert.Equal(t, auction_entity.Completed, result.Status)
}

func TestAuctionClosingWithValidInterval(t *testing.T) {
	// Test with valid interval
	os.Setenv("AUCTION_INTERVAL", "5m")
	defer os.Unsetenv("AUCTION_INTERVAL")

	interval := getAuctionInterval()
	assert.Equal(t, 5*time.Minute, interval)
}

func TestAuctionClosingWithInvalidInterval(t *testing.T) {
	// Test with invalid interval
	os.Setenv("AUCTION_INTERVAL", "invalid")
	defer os.Unsetenv("AUCTION_INTERVAL")

	interval := getAuctionInterval()
	assert.Equal(t, 5*time.Minute, interval)
}

func TestAuctionClosingWithShortInterval(t *testing.T) {
	// Test with interval shorter than 1 minute
	os.Setenv("AUCTION_INTERVAL", "30s")
	defer os.Unsetenv("AUCTION_INTERVAL")

	interval := getAuctionInterval()
	assert.Equal(t, time.Minute, interval)
}
