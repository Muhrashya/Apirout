package targets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoDBConnector handles MongoDB target database operations
type MongoDBConnector struct{}

// NewMongoDBConnector creates a new MongoDB connector
func NewMongoDBConnector() *MongoDBConnector {
	return &MongoDBConnector{}
}

// BuildConnectionString constructs a standard MongoDB connection URI
func (c *MongoDBConnector) BuildConnectionString(host string, port int, dbname, username, password, authSource string) string {
	// If host is already a full URI, return as-is
	if strings.HasPrefix(host, "mongodb://") || strings.HasPrefix(host, "mongodb+srv://") {
		return host
	}

	if port <= 0 {
		port = 27017
	}

	if authSource == "" {
		authSource = "admin"
	}

	if username != "" && password != "" {
		encodedUser := url.QueryEscape(username)
		encodedPassword := url.QueryEscape(password)
		return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=%s&connectTimeoutMS=5000",
			encodedUser, encodedPassword, host, port, dbname, authSource)
	}

	return fmt.Sprintf("mongodb://%s:%d/%s?connectTimeoutMS=5000", host, port, dbname)
}

// TestConnection tests a MongoDB connection and returns ping latency
func (c *MongoDBConnector) TestConnection(connString string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()

	clientOpts := options.Client().ApplyURI(connString)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return 0, fmt.Errorf("failed to initialize MongoDB client: %w", err)
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	// Ping the primary node
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return 0, fmt.Errorf("MongoDB connection ping failed: %w", err)
	}

	latency := time.Since(start)
	return latency, nil
}

// Execute runs a document query on a MongoDB collection
func (c *MongoDBConnector) Execute(ctx context.Context, connString, query string, params map[string]interface{}, limit int) ([]Row, error) {
	if limit <= 0 {
		limit = 100
	}

	clientOpts := options.Client().ApplyURI(connString)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	// Extract database name from connection string
	cs, err := options.Client().ApplyURI(connString).GetURI(), error(nil)
	_ = cs
	dbName := "test"
	if parsed, err := url.Parse(connString); err == nil && len(parsed.Path) > 1 {
		dbName = strings.TrimPrefix(parsed.Path, "/")
	}

	collectionName := strings.TrimSpace(query)
	if collectionName == "" {
		collectionName = "items"
	}

	coll := client.Database(dbName).Collection(collectionName)
	findOpts := options.Find().SetLimit(int64(limit))

	filter := bson.M{}
	if len(params) > 0 {
		for k, v := range params {
			filter[k] = v
		}
	}

	cursor, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("MongoDB query failed on collection %s: %w", collectionName, err)
	}
	defer cursor.Close(ctx)

	var results []Row
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		var row Row
		for k, v := range doc {
			// Convert bson.ObjectID or complex types to string representation if needed
			if b, err := json.Marshal(v); err == nil {
				var val interface{}
				_ = json.Unmarshal(b, &val)
				row = append(row, Column{Name: k, Value: val})
			} else {
				row = append(row, Column{Name: k, Value: fmt.Sprintf("%v", v)})
			}
		}
		results = append(results, row)
	}

	return results, cursor.Err()
}
