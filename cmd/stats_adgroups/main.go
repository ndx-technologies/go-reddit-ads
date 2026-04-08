package statsadgroups

import (
	"context"
	"flag"
	"io"
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

const DocShort = "ad groups stats (Impressions, Clicks, CTR, eCPM, CPC, Spend)"

const doc = `Reddit Ads Ad Groups Stats

Print per-ad-group performance report from Reddit Ads API, grouped by campaign.
`

type agg struct {
	impressions int64
	clicks      int64
	spendMicro  int64
	cpcMicro    float64
	ecpmMicro   float64
	n           int
}

func printStats(w io.StringWriter, showID bool, metrics []goredditads.ReportMetric, db goredditads.JSONDB) {
	fmtx.HeaderTo(w, "AD GROUP STATS")

	byCamp := make(map[goredditads.CampaignID]map[goredditads.AdGroupID]agg)
	campAgg := make(map[goredditads.CampaignID]agg)
	var total agg

	for _, m := range metrics {
		if byCamp[m.CampaignID] == nil {
			byCamp[m.CampaignID] = make(map[goredditads.AdGroupID]agg)
		}
		a := byCamp[m.CampaignID][m.AdGroupID]
		a.impressions += m.Impressions
		a.clicks += m.Clicks
		a.spendMicro += m.SpendMicro
		a.cpcMicro += m.CPCMicro
		a.ecpmMicro += m.ECPMMicro
		a.n++
		byCamp[m.CampaignID][m.AdGroupID] = a

		ca := campAgg[m.CampaignID]
		ca.impressions += m.Impressions
		ca.clicks += m.Clicks
		ca.spendMicro += m.SpendMicro
		ca.n++
		campAgg[m.CampaignID] = ca

		total.impressions += m.Impressions
		total.clicks += m.Clicks
		total.spendMicro += m.SpendMicro
		total.n++
	}

	camps := make([]goredditads.CampaignID, 0, len(byCamp))
	for k := range byCamp {
		camps = append(camps, k)
	}
	sort.Slice(camps, func(i, j int) bool {
		return campAgg[camps[i]].spendMicro > campAgg[camps[j]].spendMicro
	})

	for _, cid := range camps {
		groups := byCamp[cid]

		campName := string(cid)
		if cn := db.GetCampaign(cid); cn != nil {
			campName = cn.Name
		}
		w.WriteString("\n")
		w.WriteString(" " + campName)
		if showID {
			w.WriteString(" " + fmtx.DimS("("+string(cid)+")"))
		}
		w.WriteString("\n")

		agList := make([]goredditads.AdGroupID, 0, len(groups))
		for k := range groups {
			agList = append(agList, k)
		}
		sort.Slice(agList, func(i, j int) bool { return groups[agList[i]].spendMicro > groups[agList[j]].spendMicro })

		maxImpressions := int64(0)
		for _, ag := range agList {
			if groups[ag].impressions > maxImpressions {
				maxImpressions = groups[ag].impressions
			}
		}

		campTotalClicks := campAgg[cid].clicks
		campTotalImps := campAgg[cid].impressions
		campCTR := float64(0)
		if campTotalImps > 0 {
			campCTR = float64(campTotalClicks) / float64(campTotalImps)
		}

		tw := fmtx.TableWriter{
			Indent: "    ",
			Out:    w,
			Cols: []fmtx.TablCol{
				{Header: "Ad Group", Width: 28},
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

		ca := campAgg[cid]
		campECPM := float64(0)
		campCPC := float64(0)
		// campAgg doesn't accumulate ecpm/cpc – get them from group sums
		for _, a := range groups {
			campECPM += a.ecpmMicro
			campCPC += a.cpcMicro
		}
		campN := 0
		for _, a := range groups {
			campN += a.n
		}
		if campN > 0 {
			campECPM = campECPM / float64(campN) / 1_000_000
			campCPC = campCPC / float64(campN) / 1_000_000
		}

		tw.WriteHeader()
		subheader := []string{
			"",
			strconv.FormatInt(ca.impressions, 10),
			"",
			strconv.FormatInt(ca.clicks, 10),
			strconv.FormatFloat(campCTR*100, 'f', 2, 64) + "%",
			"",
			strconv.FormatFloat(campECPM, 'f', 2, 64),
			strconv.FormatFloat(campCPC, 'f', 2, 64),
			strconv.FormatFloat(float64(ca.spendMicro)/1_000_000, 'f', 2, 64),
		}
		if showID {
			subheader = append([]string{""}, subheader...)
		}
		tw.WriteSubHeader(subheader...)
		tw.WriteHeaderLine()

		for _, agid := range agList {
			a := groups[agid]
			ecpm := float64(0)
			if a.n > 0 {
				ecpm = a.ecpmMicro / float64(a.n) / 1_000_000
			}
			cpc := float64(0)
			if a.n > 0 {
				cpc = a.cpcMicro / float64(a.n) / 1_000_000
			}
			spend := float64(a.spendMicro) / 1_000_000

			ctrStr := fmtx.DimS("-")
			if a.impressions > 0 {
				if a.clicks == 0 {
					ctrStr = fmtx.RedS("0.00%")
				} else {
					ctr := float64(a.clicks) / float64(a.impressions)
					ctrStr = fmtx.ColorizeDist(strconv.FormatFloat(ctr*100, 'f', 2, 64)+"%", ctr, []float64{campCTR * 0.7, campCTR}, []string{fmtx.Red, fmtx.Yellow, fmtx.Green})
				}
			}

			z, zOK := goredditads.CTRZScore(a.clicks, a.impressions, campTotalClicks, campTotalImps)

			name := string(agid)
			if ag := db.GetAdGroup(agid); ag != nil {
				name = ag.Name
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
				row = append([]string{fmtx.DimS(string(agid))}, row...)
			}
			tw.WriteRow(row...)
		}
	}
	w.WriteString("\n")
}

func Run(args []string) {
	fs := flag.NewFlagSet("stats adgroups", flag.ExitOnError)
	var (
		configPath     string
		secretsPath    string
		campaignIDsStr string
		adGroupIDsStr  string
		showID         bool
		from, until    time.Time
	)
	fs.StringVar(&configPath, "config", "reddit-ads/config.json", "config JSON file path")
	fs.StringVar(&secretsPath, "secrets", "reddit-ads/secrets.json", "secrets JSON file path")
	fs.BoolVar(&showID, "id", false, "show IDs")
	fs.BoolVar(&fmtx.EnableColor, "color", os.Getenv("NO_COLOR") == "", "colorize output")
	fs.StringVar(&campaignIDsStr, "campaign-ids", "", "comma-separated campaign IDs to filter")
	fs.StringVar(&adGroupIDsStr, "adgroup-ids", "", "comma-separated ad group IDs to filter")
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
		[]goredditads.ReportBreakdown{goredditads.ReportBreakdownCampaignID, goredditads.ReportBreakdownAdGroupID},
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

	printStats(os.Stdout, showID, metrics, db)
}
