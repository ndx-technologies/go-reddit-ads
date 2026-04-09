package cmdfetch

import (
	"context"
	"flag"
	"log"
	"net/http"

	goredditads "github.com/ndx-technologies/go-reddit-ads"
	"github.com/ndx-technologies/jsonx"
)

const DocShort = "fetch state from API into JSON config"

const doc = `Reddit Ads Fetch

Fetch campaigns, ad groups, and ads from Reddit Ads API and save to config file.

`

func Run(args []string) {
	var (
		secretsPath string
		configPath  string
	)
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	fs.StringVar(&configPath, "config", "reddit-ads/config.json", "config JSON file path")
	fs.StringVar(&secretsPath, "secrets", "reddit-ads/secrets.json", "secrets JSON file path")
	fs.Usage = func() { fs.Output().Write([]byte(doc)); fs.PrintDefaults() }
	fs.Parse(args)

	db, err := jsonx.FromFile[goredditads.JSONDB](configPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	secrets, err := jsonx.FromFile[goredditads.RedditHTTPClientSecrets](secretsPath)
	if err != nil {
		log.Fatalf("loading secrets: %v", err)
	}

	ctx := context.Background()

	config := goredditads.RedditHTTPClientConfig{
		AdAccountID: db.AdAccountID,
	}
	config = config.WithDefaults()

	client := goredditads.RedditHTTPClient{Config: config, Secrets: secrets, HTTPClient: http.DefaultClient}

	if secrets.AppToken == "" {
		token, err := client.FetchAccessTokenWithCode(ctx, secrets.AppAuthCode, db.AppRedirectURI)
		if err != nil {
			log.Fatalf("getting access token: %v", err)
		}
		client.Secrets.AppToken = token
		if err := client.Secrets.Save(secretsPath); err != nil {
			log.Fatalf("saving secrets: %v", err)
		}
	}

	account, err := client.FetchAdAccount(ctx)
	if err != nil {
		log.Fatalf("fetching ad account: %v", err)
	}
	db.Currency = account.Currency
	db.ExcludedCommunities = account.ExcludedCommunities
	db.ExcludedKeywords = account.ExcludedKeywords

	campaigns, err := client.FetchCampaigns(ctx)
	if err != nil {
		log.Fatalf("fetching campaigns: %v", err)
	}

	adGroups, err := client.FetchAdGroups(ctx)
	if err != nil {
		log.Fatalf("fetching ad groups: %v", err)
	}

	ads, err := client.FetchAds(ctx)
	if err != nil {
		log.Fatalf("fetching ads: %v", err)
	}

	postIDs := map[goredditads.PostID][]int{}
	for i, a := range ads {
		if a.PostID != "" {
			postIDs[a.PostID] = append(postIDs[a.PostID], i)
		}
	}
	for postID, idxs := range postIDs {
		post, err := client.FetchPost(ctx, postID)
		if err != nil {
			log.Printf("fetching post %s: %v", postID, err)
			continue
		}
		for _, i := range idxs {
			p := post
			ads[i].Post = &p
		}
	}

	db.Init()
	for _, c := range campaigns {
		db.UpsertCampaign(c)
	}
	for _, ag := range adGroups {
		db.UpsertAdGroup(ag)
	}
	for _, a := range ads {
		db.UpsertAd(a)
	}

	if err := db.Save(configPath); err != nil {
		log.Fatalf("saving config: %v", err)
	}

	log.Printf("saved %d campaigns, %d ad groups, %d ads to %s", len(campaigns), len(adGroups), len(ads), configPath)
}
