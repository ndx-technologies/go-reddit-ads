package statscampaigns

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ndx-technologies/fmtx"
	goredditads "github.com/ndx-technologies/go-reddit-ads"
	"github.com/ndx-technologies/jsonx"
)

const DocShort = "campaigns stats (Impressions, Clicks, CTR, eCPM, CPC, Spend)"

const doc = `Reddit Ads Campaigns Stats

Print per-campaign performance report from Reddit Ads API.
`

func printStats(w io.StringWriter, showID bool, metrics []goredditads.ReportMetric, db goredditads.JSONDB) {
	fmtx.HeaderTo(w, "CAMPAIGN STATS")

	type agg struct {
		impressions int64
		clicks      int64
		spendMicro  int64
		cpcMicro    float64
		ecpmMicro   float64
		n           int
	}

	byCampaign := make(map[goredditads.CampaignID]agg)
	var total agg
	for _, m := range metrics {
		a := byCampaign[m.CampaignID]
		a.impressions += m.Impressions
		a.clicks += m.Clicks
		a.spendMicro += m.SpendMicro
		a.cpcMicro += m.CPCMicro
		a.ecpmMicro += m.ECPMMicro
		a.n++
		byCampaign[m.CampaignID] = a

		total.impressions += m.Impressions
		total.clicks += m.Clicks
		total.spendMicro += m.SpendMicro
		total.cpcMicro += m.CPCMicro
		total.ecpmMicro += m.ECPMMicro
		total.n++
	}

	campaigns := slices.Collect(func(yield func(goredditads.CampaignID) bool) {
		for k := range byCampaign {
			if !yield(k) {
				return
			}
		}
	})
	sort.Slice(campaigns, func(i, j int) bool {
		return byCampaign[campaigns[i]].spendMicro > byCampaign[campaigns[j]].spendMicro
	})

	totalCTR := float64(0)
	if total.impressions > 0 {
		totalCTR = float64(total.clicks) / float64(total.impressions)
	}
	totalECPM := float64(0)
	if total.n > 0 {
		totalECPM = total.ecpmMicro / float64(total.n) / 1_000_000
	}
	totalCPC := float64(0)
	if total.n > 0 {
		totalCPC = total.cpcMicro / float64(total.n) / 1_000_000
	}

	maxImpressions := int64(0)
	for _, a := range byCampaign {
		if a.impressions > maxImpressions {
			maxImpressions = a.impressions
		}
	}

	tw := fmtx.TableWriter{
		Indent: "  ",
		Out:    w,
		Cols: []fmtx.TablCol{
			{Header: "Campaign", Width: 32},
			{Header: "Impressions", Width: 12, Alignment: fmtx.AlignRight},
			{Header: "Impr", Width: 12},
			{Header: "Clicks", Width: 8, Alignment: fmtx.AlignRight},
			{Header: "CTR", Width: 7, Alignment: fmtx.AlignRight},
			{Header: "CTR Conf", Width: 8, Alignment: fmtx.AlignRight},
			{Header: "eCPM", Width: 7, Alignment: fmtx.AlignRight},
			{Header: "CPC", Width: 7, Alignment: fmtx.AlignRight},
			{Header: "Spend(USD)", Width: 11, Alignment: fmtx.AlignRight},
		},
	}
	if showID {
		tw.Cols = append([]fmtx.TablCol{{Header: "ID", Width: 20}}, tw.Cols...)
	}

	tw.WriteHeader()
	subheader := []string{
		"",
		strconv.FormatInt(total.impressions, 10),
		"",
		strconv.FormatInt(total.clicks, 10),
		strconv.FormatFloat(totalCTR*100, 'f', 2, 64) + "%",
		"",
		strconv.FormatFloat(totalECPM, 'f', 2, 64),
		strconv.FormatFloat(totalCPC, 'f', 2, 64),
		strconv.FormatFloat(float64(total.spendMicro)/1_000_000, 'f', 2, 64),
	}
	if showID {
		subheader = append([]string{""}, subheader...)
	}
	tw.WriteSubHeader(subheader...)
	tw.WriteHeaderLine()

	for _, cid := range campaigns {
		a := byCampaign[cid]
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
		spend := float64(a.spendMicro) / 1_000_000

		ctrStr := fmtx.DimS(strconv.FormatFloat(ctr*100, 'f', 2, 64) + "%")
		if a.clicks > 0 {
			ctrStr = fmtx.ColorizeDist(strconv.FormatFloat(ctr*100, 'f', 2, 64)+"%", ctr, []float64{totalCTR * 0.7, totalCTR}, []string{fmtx.Red, fmtx.Yellow, fmtx.Green})
		}

		z, zOK := goredditads.CTRZScore(a.clicks, a.impressions, total.clicks, total.impressions)

		name := string(cid)
		if cn := db.GetCampaign(cid); cn != nil {
			name = cn.Name
		}

		row := []string{
			name,
			strconv.FormatInt(a.impressions, 10),
			fmtx.VolumeBar(a.impressions, maxImpressions, 12),
			strconv.FormatInt(a.clicks, 10),
			ctrStr,
			goredditads.CTRSigStr(z, zOK),
			strconv.FormatFloat(ecpm, 'f', 2, 64),
			strconv.FormatFloat(cpc, 'f', 2, 64),
			strconv.FormatFloat(spend, 'f', 2, 64),
		}
		if showID {
			row = append([]string{fmtx.DimS(string(cid))}, row...)
		}
		tw.WriteRow(row...)
	}
	w.WriteString("\n")
}

func Run(args []string) {
	fs := flag.NewFlagSet("stats campaigns", flag.ExitOnError)
	var (
		configPath     string
		secretsPath    string
		campaignIDsStr string
		showID         bool
		from, until    time.Time
	)
	fs.StringVar(&configPath, "config", "reddit-ads/config.json", "config JSON file path")
	fs.StringVar(&secretsPath, "secrets", "reddit-ads/secrets.json", "secrets JSON file path")
	fs.BoolVar(&showID, "id", false, "show IDs")
	fs.BoolVar(&fmtx.EnableColor, "color", os.Getenv("NO_COLOR") == "", "colorize output")
	fs.StringVar(&campaignIDsStr, "campaign-ids", "", "comma-separated campaign IDs to filter")
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

	metrics, err := client.FetchReport(ctx, from, until,
		[]goredditads.ReportBreakdown{goredditads.ReportBreakdownCampaignID},
		[]goredditads.ReportField{goredditads.Impressions, goredditads.Clicks, goredditads.CTR, goredditads.ECPM, goredditads.CPC, goredditads.Spend},
	)
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

	printStats(os.Stdout, showID, metrics, db)
}
