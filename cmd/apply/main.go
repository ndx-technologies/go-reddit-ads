package cmdapply

import (
	"context"
	"flag"
	"log"
	"net/http"

	goredditads "github.com/ndx-technologies/go-reddit-ads"
	"github.com/ndx-technologies/jsonx"
)

const DocShort = "diff two configs and apply patches to API"

const doc = `Reddit Ads Apply

Compute diff between -from (current live state) and -to (desired state),
then apply patches to Reddit Ads API:

  update  — entity present in both -from and -to with differing fields (PATCH with -to values)
  pause   — entity in -from but missing from -to (sets configured_status=PAUSED)
  skip    — entity with no ID in -to, or entity with no changes

If -from is omitted, only updates are performed.

`

func Run(args []string) {
	var (
		secretsPath string
		fromPath    string
		toPath      string
		dryRun      bool
	)
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
	fs.StringVar(&toPath, "to", "reddit-ads/config.json", "desired state config JSON file path")
	fs.StringVar(&fromPath, "from", "", "current live state config JSON file path (optional)")
	fs.StringVar(&secretsPath, "secrets", "reddit-ads/secrets.json", "secrets JSON file path")
	fs.BoolVar(&dryRun, "dry-run", false, "print what would be done without calling API")
	fs.Usage = func() { fs.Output().Write([]byte(doc)); fs.PrintDefaults() }
	fs.Parse(args)

	to, err := jsonx.FromFile[goredditads.JSONDB](toPath)
	if err != nil {
		log.Fatalf("loading -to config: %v", err)
	}

	var from goredditads.JSONDB
	if fromPath != "" {
		from, err = jsonx.FromFile[goredditads.JSONDB](fromPath)
		if err != nil {
			log.Fatalf("loading -from config: %v", err)
		}
	}
	from.Init()

	secrets, err := jsonx.FromFile[goredditads.RedditHTTPClientSecrets](secretsPath)
	if err != nil {
		log.Fatalf("loading secrets: %v", err)
	}

	ctx := context.Background()

	client := goredditads.RedditHTTPClient{
		Config:     goredditads.RedditHTTPClientConfig{AdAccountID: to.AdAccountID}.WithDefaults(),
		Secrets:    secrets,
		HTTPClient: http.DefaultClient,
	}

	if !dryRun {
		if secrets.AppToken == "" {
			token, err := client.FetchAccessTokenWithCode(ctx, secrets.AppAuthCode, to.AppRedirectURI)
			if err != nil {
				log.Fatalf("getting access token: %v", err)
			}
			client.Secrets.AppToken = token
			if err := client.Secrets.Save(secretsPath); err != nil {
				log.Fatalf("saving secrets: %v", err)
			}
		}
	}

	// index IDs seen in -to to detect removals
	seenCampaigns := make(map[goredditads.CampaignID]struct{})
	seenAdGroups := make(map[goredditads.AdGroupID]struct{})
	seenAds := make(map[goredditads.AdID]struct{})

	for ci := range to.Campaigns {
		cn := &to.Campaigns[ci]

		if cn.ID.IsZero() || from.GetCampaign(cn.ID) == nil {
			log.Printf("skip campaign %q — no ID", cn.Name)
		} else {
			// update only if changed
			if cn.Campaign.IsEqual(from.GetCampaign(cn.ID).Campaign) {
				log.Printf("skip campaign %s (%s) — no change", cn.ID, cn.Name)
			} else if dryRun {
				log.Printf("[dry-run] update campaign %s (%s)", cn.ID, cn.Name)
			} else if err := client.UpdateCampaign(ctx, cn.Campaign); err != nil {
				log.Printf("error updating campaign %s: %v", cn.ID, err)
			} else {
				log.Printf("updated campaign %s (%s)", cn.ID, cn.Name)
			}
		}
		seenCampaigns[cn.ID] = struct{}{}

		for ai := range cn.AdGroups {
			an := &cn.AdGroups[ai]
			an.CampaignID = cn.ID

			if an.ID.IsZero() || from.GetAdGroup(an.ID) == nil {
				log.Printf("skip ad group %q — no ID", an.Name)
			} else {
				// update only if changed
				if an.AdGroup.IsEqual(from.GetAdGroup(an.ID).AdGroup) {
					log.Printf("skip ad group %s (%s) — no change", an.ID, an.Name)
				} else if dryRun {
					log.Printf("[dry-run] update ad group %s (%s)", an.ID, an.Name)
				} else if err := client.UpdateAdGroup(ctx, an.AdGroup); err != nil {
					log.Printf("error updating ad group %s: %v", an.ID, err)
				} else {
					log.Printf("updated ad group %s (%s)", an.ID, an.Name)
				}
			}
			seenAdGroups[an.ID] = struct{}{}

			for ki := range an.Ads {
				a := &an.Ads[ki]
				a.AdGroupID = an.ID
				a.CampaignID = cn.ID

				if a.ID.IsZero() || from.GetAd(a.ID) == nil {
					log.Printf("skip ad %q — no ID", a.Name)
				} else {
					if a.IsEqual(*from.GetAd(a.ID)) {
						log.Printf("skip ad %s (%s) — no change", a.ID, a.Name)
					} else if dryRun {
						log.Printf("[dry-run] update ad %s (%s)", a.ID, a.Name)
					} else if err := client.UpdateAd(ctx, *a); err != nil {
						log.Printf("error updating ad %s: %v", a.ID, err)
					} else {
						log.Printf("updated ad %s (%s)", a.ID, a.Name)
					}
				}
				seenAds[a.ID] = struct{}{}
			}
		}
	}

	// pause entities in -from not present in -to
	if fromPath != "" {
		paused := goredditads.Paused
		for _, fcn := range from.Campaigns {
			if _, ok := seenCampaigns[fcn.ID]; ok {
				continue
			}
			if dryRun {
				log.Printf("[dry-run] pause campaign %s (%s)", fcn.ID, fcn.Name)
			} else if err := client.UpdateCampaign(ctx, goredditads.Campaign{ID: fcn.ID, ConfiguredStatus: paused}); err != nil {
				log.Printf("error pausing campaign %s: %v", fcn.ID, err)
			} else {
				log.Printf("paused campaign %s (%s)", fcn.ID, fcn.Name)
			}
			for _, fan := range fcn.AdGroups {
				if dryRun {
					log.Printf("[dry-run] pause ad group %s (%s)", fan.ID, fan.Name)
				} else if err := client.UpdateAdGroup(ctx, goredditads.AdGroup{ID: fan.ID, ConfiguredStatus: paused}); err != nil {
					log.Printf("error pausing ad group %s: %v", fan.ID, err)
				} else {
					log.Printf("paused ad group %s (%s)", fan.ID, fan.Name)
				}
				for _, fa := range fan.Ads {
					if dryRun {
						log.Printf("[dry-run] pause ad %s (%s)", fa.ID, fa.Name)
					} else if err := client.UpdateAd(ctx, goredditads.Ad{ID: fa.ID, ConfiguredStatus: paused}); err != nil {
						log.Printf("error pausing ad %s: %v", fa.ID, err)
					} else {
						log.Printf("paused ad %s (%s)", fa.ID, fa.Name)
					}
				}
			}
		}
		for _, fan := range from.Campaigns {
			for _, ag := range fan.AdGroups {
				if _, ok := seenAdGroups[ag.ID]; ok {
					continue
				}
				if _, ok := seenCampaigns[ag.CampaignID]; !ok {
					continue // already paused via campaign
				}
				if dryRun {
					log.Printf("[dry-run] pause ad group %s (%s)", ag.ID, ag.Name)
				} else if err := client.UpdateAdGroup(ctx, goredditads.AdGroup{ID: ag.ID, ConfiguredStatus: paused}); err != nil {
					log.Printf("error pausing ad group %s: %v", ag.ID, err)
				} else {
					log.Printf("paused ad group %s (%s)", ag.ID, ag.Name)
				}
				for _, fa := range ag.Ads {
					if dryRun {
						log.Printf("[dry-run] pause ad %s (%s)", fa.ID, fa.Name)
					} else if err := client.UpdateAd(ctx, goredditads.Ad{ID: fa.ID, ConfiguredStatus: paused}); err != nil {
						log.Printf("error pausing ad %s: %v", fa.ID, err)
					} else {
						log.Printf("paused ad %s (%s)", fa.ID, fa.Name)
					}
				}
			}
			for _, ag := range fan.AdGroups {
				for _, fa := range ag.Ads {
					if _, ok := seenAds[fa.ID]; ok {
						continue
					}
					if _, ok := seenAdGroups[fa.AdGroupID]; !ok {
						continue // already paused via ad group
					}
					if dryRun {
						log.Printf("[dry-run] pause ad %s (%s)", fa.ID, fa.Name)
					} else if err := client.UpdateAd(ctx, goredditads.Ad{ID: fa.ID, ConfiguredStatus: paused}); err != nil {
						log.Printf("error pausing ad %s: %v", fa.ID, err)
					} else {
						log.Printf("paused ad %s (%s)", fa.ID, fa.Name)
					}
				}
			}
		}
	}

	if dryRun {
		return
	}
}
