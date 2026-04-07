package cmdauth

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"strconv"

	goredditads "github.com/ndx-technologies/go-reddit-ads"
	"github.com/ndx-technologies/jsonx"
)

const DocShort = "authorize and store access token"

const doc = `Reddit Ads Auth

Exchange authorization code for access token and store in secrets file.
If app_auth_code is empty, prints the authorization URL to visit.

Flags:
`

func Run(args []string) {
	var (
		secretsPath string
		configPath  string
	)
	fs := flag.NewFlagSet("auth", flag.ExitOnError)
	fs.StringVar(&configPath, "config", "reddit-ads/config.json", "config JSON file path")
	fs.StringVar(&secretsPath, "secrets", "reddit-ads/secrets.json", "secrets JSON file path")
	fs.Usage = func() { fs.Output().Write([]byte(doc)); fs.PrintDefaults() }
	fs.Parse(args)

	db, err := jsonx.FromFile[goredditads.JSONDB](configPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	s, err := jsonx.FromFile[goredditads.RedditHTTPClientSecrets](secretsPath)
	if err != nil {
		log.Fatalf("loading secrets: %v", err)
	}

	client := goredditads.RedditHTTPClient{
		Config:     goredditads.RedditHTTPClientConfig{AppID: db.AppID}.WithDefaults(),
		Secrets:    s,
		HTTPClient: http.DefaultClient,
	}

	if s.AppAuthCode == "" {
		state := rand.Int()
		log.Fatal("visit to authorize:\n" + client.AuthorizeURL(db.AppRedirectURI, strconv.Itoa(state)) + "\n\nthen set app_auth_code in " + secretsPath)
	}

	ctx := context.Background()
	token, err := client.FetchAccessTokenWithCode(ctx, s.AppAuthCode, db.AppRedirectURI)
	if err != nil {
		log.Fatalf("exchanging code: %v", err)
	}

	s.AppToken = token
	if err := s.Save(secretsPath); err != nil {
		log.Fatalf("saving secrets: %v", err)
	}

	log.Printf("access token saved to %s", secretsPath)
}
