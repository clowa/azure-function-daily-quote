package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/clowa/az-func-daily-quote/src/lib/config"
	quotable "github.com/clowa/az-func-daily-quote/src/lib/quotableSdk"
	log "github.com/sirupsen/logrus"
)

// Struct representing the structure returned from the quotable API
type Quote struct {
	Id           string   `json:"id"`
	Content      string   `json:"content"`
	Author       string   `json:"author"`
	AuthorSlug   string   `json:"authorSlug"`
	Length       int      `json:"length"`
	Tags         []string `json:"tags"`
	CreationDate string   `json:"creationDate"`
}

func (q *Quote) Load(i *quotable.QuoteResponse) {
	q.Id = i.Id
	q.Content = i.Content
	q.Author = i.Author
	q.AuthorSlug = i.AuthorSlug
	q.Length = i.Length
	q.Tags = i.Tags

	now := time.Now()
	q.CreationDate = fmt.Sprintf("%d-%d-%d", now.Year(), int(now.Month()), now.Day())
}

func writeQuoteToDatabase(q *Quote) error {
	config := config.GetConfig()

	credential, err := azidentity.NewManagedIdentityCredential(nil)
	if err != nil {
		return err
	}

	client, err := azcosmos.NewClient(config.CosmosHost, credential, nil)
	if err != nil {
		return err
	}

	database, err := client.NewDatabase(config.CosmosDatabase)
	if err != nil {
		return err
	}

	container, err := database.NewContainer(config.CosmosContainer)
	if err != nil {
		return err
	}

	partitionKey := azcosmos.NewPartitionKeyString(q.AuthorSlug)

	bytes, err := json.Marshal(q)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err = container.UpsertItem(ctx, partitionKey, bytes, nil) // ToDo: change to CreateItem()
	if err != nil {
		return err
	}

	return nil
}

func QuoteOfTheDayHandler(w http.ResponseWriter, r *http.Request) {
	quotes, err := quotable.GetRandomQuote(quotable.GetRandomQuoteQueryParams{Limit: 1, Tags: []string{"technology"}})
	if err != nil {
		handleWarn(w, err)
	}
	quoteOfTheDay := quotes[0]

	log.Infof("Quote of the day: %s by %s", quoteOfTheDay.Content, quoteOfTheDay.Author)

	// Write quote to database
	databaseQuote := Quote{}
	databaseQuote.Load(&quoteOfTheDay)
	err = writeQuoteToDatabase(&databaseQuote)
	if err != nil {
		log.Warnf("Error writing quote to database: %s", err)
	}

	// Write response
	responseBodyBytes := new(bytes.Buffer)
	json.NewEncoder(responseBodyBytes).Encode(quoteOfTheDay)
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBodyBytes.Bytes())
	w.WriteHeader(http.StatusOK)
}
