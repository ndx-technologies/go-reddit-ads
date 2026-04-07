package timeline

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ndx-technologies/fmtx"
	goredditads "github.com/ndx-technologies/go-reddit-ads"
	"github.com/ndx-technologies/jsonx"
)

const DocShort = "daily performance timeline"

const doc = `Reddit Ads Timeline

Print daily performance timeline from Reddit Ads API.
`

func Run(args []string) {
	fs := flag.NewFlagSet("timeline", flag.ExitOnError)
	var (
		configPath     string
		secretsPath    string
		campaignIDsStr string
		adGroupIDsStr  string
		adIDsStr       string
		from, until    time.Time
	)
	fs.StringVar(&configPath, "config", "reddit-ads/config.json", "config JSON file path")
	fs.StringVar(&secretsPath, "secrets", "reddit-ads/secrets.json", "secrets JSON file path")
	fs.BoolVar(&fmtx.EnableColor, "color", os.Getenv("NO_COLOR") == "", "colorize output")
	fs.StringVar(&campaignIDsStr, "campaign-ids", "", "comma-separated campaign IDs to filter")
	fs.StringVar(&adGroupIDsStr, "adgroup-ids", "", "comma-separated ad group IDs to filter")
	fs.StringVar(&adIDsStr, "ad-ids", "", "comma-separated ad IDs to filter")
	fs.Func("from", "start date (e.g. 2025-01-01)", func(s string) (err error) {
		from, err = time.Parse(time.DateOnly, s)
		return
	})
	fs.Func("until", "end date inclusive (e.g. 2025-12-31)", func(s string) (err error) {
		until, err = time.Parse(time.DateOnly, s)
		return
	})
	fs.Usage = func() { fs.Output().Write([]byte(doc)); fs.PrintDefaults() }
	fs.Parse(args)

	if from.IsZero() {
		from = time.Now().UTC().AddDate(0, -1, 0).Truncate(24 * time.Hour)
	}
	if until.IsZero() {
		until = time.Now().UTC().Truncate(24 * time.Hour)
	}

	db, err := jsonx.FromFile[goredditads.JSONDB](configPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}
	db.Init()

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

	if client.Secrets.AppToken == "" {
		token, err := client.FetchAccessTokenWithCode(ctx, secrets.AppAuthCode, db.AppRedirectURI)
		if err != nil {
			log.Fatalf("getting access token: %v", err)
		}
		client.Secrets.AppToken = token
		if err := client.Secrets.Save(secretsPath); err != nil {
			log.Fatalf("saving secrets: %v", err)
		}
	}

	breakdowns := []goredditads.ReportBreakdown{goredditads.ReportBreakdownDate}
	if campaignIDsStr != "" {
		breakdowns = append(breakdowns, goredditads.ReportBreakdownCampaignID)
	}
	if adGroupIDsStr != "" {
		breakdowns = append(breakdowns, goredditads.ReportBreakdownAdGroupID)
	}
	if adIDsStr != "" {
		breakdowns = append(breakdowns, goredditads.ReportBreakdownAdID)
	}

	metrics, err := client.FetchReport(ctx, from, until, breakdowns, []goredditads.ReportField{goredditads.Impressions, goredditads.Clicks, goredditads.CTR, goredditads.ECPM, goredditads.CPC, goredditads.Spend})
	if err != nil {
		log.Fatalf("fetching report: %v", err)
	}

	if campaignIDsStr != "" {
		keep := make(map[goredditads.CampaignID]bool)
		for id := range strings.SplitSeq(campaignIDsStr, ",") {
			keep[goredditads.CampaignID(strings.TrimSpace(id))] = true
		}
		filtered := metrics[:0]
		for _, m := range metrics {
			if keep[m.CampaignID] {
				filtered = append(filtered, m)
			}
		}
		metrics = filtered
	}

	if adGroupIDsStr != "" {
		keep := make(map[goredditads.AdGroupID]bool)
		for id := range strings.SplitSeq(adGroupIDsStr, ",") {
			keep[goredditads.AdGroupID(strings.TrimSpace(id))] = true
		}
		filtered := metrics[:0]
		for _, m := range metrics {
			if keep[m.AdGroupID] {
				filtered = append(filtered, m)
			}
		}
		metrics = filtered
	}

	if adIDsStr != "" {
		keep := make(map[goredditads.AdID]bool)
		for id := range strings.SplitSeq(adIDsStr, ",") {
			keep[goredditads.AdID(strings.TrimSpace(id))] = true
		}
		filtered := metrics[:0]
		for _, m := range metrics {
			if keep[m.AdID] {
				filtered = append(filtered, m)
			}
		}
		metrics = filtered
	}

	type agg struct {
		impressions int64
		clicks      int64
		spendMicro  int64
		cpcMicro    float64
		ecpmMicro   float64
		n           int
	}

	byDay := make(map[time.Time]agg)
	var total agg
	for _, m := range metrics {
		day := m.Date.UTC().Truncate(24 * time.Hour)
		a := byDay[day]
		a.impressions += m.Impressions
		a.clicks += m.Clicks
		a.spendMicro += m.SpendMicro
		a.cpcMicro += m.CPCMicro
		a.ecpmMicro += m.ECPMMicro
		a.n++
		byDay[day] = a

		total.impressions += m.Impressions
		total.clicks += m.Clicks
		total.spendMicro += m.SpendMicro
		total.cpcMicro += m.CPCMicro
		total.ecpmMicro += m.ECPMMicro
		total.n++
	}

	days := make([]time.Time, 0, len(byDay))
	for d := range byDay {
		days = append(days, d)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].After(days[j]) })

	totalCTR := float64(0)
	if total.impressions > 0 {
		totalCTR = float64(total.clicks) / float64(total.impressions)
	}

	maxClicks := int64(0)
	for _, a := range byDay {
		if a.clicks > maxClicks {
			maxClicks = a.clicks
		}
	}

	tw := fmtx.TableWriter{
		Indent: "  ",
		Out:    os.Stdout,
		Cols: []fmtx.TablCol{
			{Header: "Date(UTC)", Width: 10},
			{Header: "Spend(USD)", Width: 10, Alignment: fmtx.AlignRight},
			{Header: "Clicks", Width: 7, Alignment: fmtx.AlignRight},
			{Header: "Clicks", Width: 12},
			{Header: "Impr", Width: 8, Alignment: fmtx.AlignRight},
			{Header: "CTR", Width: 7, Alignment: fmtx.AlignRight},
			{Header: "eCPM", Width: 7, Alignment: fmtx.AlignRight},
			{Header: "CPC", Width: 7, Alignment: fmtx.AlignRight},
		},
	}
	tw.WriteHeader()
	tw.WriteHeaderLine()

	for _, day := range days {
		a := byDay[day]

		ctr := float64(0)
		if a.impressions > 0 {
			ctr = float64(a.clicks) / float64(a.impressions)
		}
		ecpm := float64(0)
		if a.n > 0 {
			ecpm = a.ecpmMicro / float64(a.n) / 1_000_000
		}
		cpc := float64(0)
		if a.n > 0 {
			cpc = a.cpcMicro / float64(a.n) / 1_000_000
		}

		clicksS := strconv.FormatInt(a.clicks, 10)
		clicksColor := fmtx.ColorizeDist(clicksS, float64(a.clicks), []float64{float64(maxClicks) * 0.5, float64(maxClicks) * 0.8}, []string{fmtx.Red, fmtx.Yellow, fmtx.Green})

		ctrStr := fmtx.DimS(strconv.FormatFloat(ctr*100, 'f', 2, 64) + "%")
		if a.clicks > 0 && totalCTR > 0 {
			ctrStr = fmtx.ColorizeDist(strconv.FormatFloat(ctr*100, 'f', 2, 64)+"%", ctr, []float64{totalCTR * 0.7, totalCTR}, []string{fmtx.Red, fmtx.Yellow, fmtx.Green})
		}

		tw.WriteRow(
			day.Format(time.DateOnly),
			strconv.FormatFloat(float64(a.spendMicro)/1_000_000, 'f', 2, 64),
			clicksColor,
			fmtx.VolumeBar(a.clicks, maxClicks, 12),
			strconv.FormatInt(a.impressions, 10),
			ctrStr,
			strconv.FormatFloat(ecpm, 'f', 2, 64),
			strconv.FormatFloat(cpc, 'f', 2, 64),
		)
	}

	fmt.Println()
}
