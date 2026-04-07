package cmdapply

import (
	"context"
	"flag"
	"log"
	"net/http"

	goredditads "github.com/ndx-technologies/go-reddit-ads"
	"github.com/ndx-technologies/jsonx"
)

const DocShort = "apply JSON config state to API"

const doc = `Reddit Ads Apply

Apply campaigns, ad groups, and ads from config file to Reddit Ads API.

`

func Run(args []string) {
	var (
		secretsPath string
		configPath  string
	)
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
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

	client := goredditads.RedditHTTPClient{
		Config:     goredditads.RedditHTTPClientConfig{AdAccountID: db.AdAccountID}.WithDefaults(),
		Secrets:    secrets,
		HTTPClient: http.DefaultClient,
	}

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

	for _, cn := range db.Campaigns {
		if err := client.UpdateCampaign(ctx, cn.Campaign); err != nil {
			log.Printf("error updating campaign %s: %v", cn.ID, err)
		} else {
			log.Printf("updated campaign %s (%s)", cn.ID, cn.Name)
		}
		for _, an := range cn.AdGroups {
			if err := client.UpdateAdGroup(ctx, an.AdGroup); err != nil {
				log.Printf("error updating ad group %s: %v", an.ID, err)
			} else {
				log.Printf("updated ad group %s (%s)", an.ID, an.Name)
			}
			for _, a := range an.Ads {
				if err := client.UpdateAd(ctx, a); err != nil {
					log.Printf("error updating ad %s: %v", a.ID, err)
				} else {
					log.Printf("updated ad %s (%s)", a.ID, a.Name)
				}
			}
		}
	}
}
