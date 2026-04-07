package statsads

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

const DocShort = "ads stats (Impressions, Clicks, CTR, eCPM, CPC, Spend)"

const doc = `Reddit Ads Ads Stats

Print per-ad performance report from Reddit Ads API, grouped by campaign and ad group.
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
	fmtx.HeaderTo(w, "AD STATS")

	type adGroupKey struct {
		campaignID goredditads.CampaignID
		adGroupID  goredditads.AdGroupID
	}

	byAd := make(map[goredditads.AdID]agg)
	adParent := make(map[goredditads.AdID]adGroupKey)
	byAdGroup := make(map[adGroupKey]agg)
	byCamp := make(map[goredditads.CampaignID]agg)

	for _, m := range metrics {
		key := adGroupKey{m.CampaignID, m.AdGroupID}

		a := byAd[m.AdID]
		a.impressions += m.Impressions
		a.clicks += m.Clicks
		a.spendMicro += m.SpendMicro
		a.cpcMicro += m.CPCMicro
		a.ecpmMicro += m.ECPMMicro
		a.n++
		byAd[m.AdID] = a
		adParent[m.AdID] = key

		ag := byAdGroup[key]
		ag.impressions += m.Impressions
		ag.clicks += m.Clicks
		ag.spendMicro += m.SpendMicro
		ag.n++
		byAdGroup[key] = ag

		ca := byCamp[m.CampaignID]
		ca.impressions += m.Impressions
		ca.clicks += m.Clicks
		ca.spendMicro += m.SpendMicro
		ca.n++
		byCamp[m.CampaignID] = ca
	}

	camps := make([]goredditads.CampaignID, 0, len(byCamp))
	for k := range byCamp {
		camps = append(camps, k)
	}
	sort.Slice(camps, func(i, j int) bool {
		return byCamp[camps[i]].spendMicro > byCamp[camps[j]].spendMicro
	})

	campGroups := make(map[goredditads.CampaignID][]adGroupKey)
	for key := range byAdGroup {
		campGroups[key.campaignID] = append(campGroups[key.campaignID], key)
	}
	for cid := range campGroups {
		sort.Slice(campGroups[cid], func(i, j int) bool {
			return byAdGroup[campGroups[cid][i]].spendMicro > byAdGroup[campGroups[cid][j]].spendMicro
		})
	}

	for _, cid := range camps {
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

		for _, agKey := range campGroups[cid] {
			agAgg := byAdGroup[agKey]
			agCTR := float64(0)
			if agAgg.impressions > 0 {
				agCTR = float64(agAgg.clicks) / float64(agAgg.impressions)
			}

			agName := string(agKey.adGroupID)
			if ag := db.GetAdGroup(agKey.adGroupID); ag != nil {
				agName = ag.Name
			}
			w.WriteString("   " + agName)
			if showID {
				w.WriteString(" " + fmtx.DimS("("+string(agKey.adGroupID)+")"))
			}
			w.WriteString("\n")

			var adList []goredditads.AdID
			for aid, key := range adParent {
				if key == agKey {
					adList = append(adList, aid)
				}
			}
			sort.Slice(adList, func(i, j int) bool {
				return byAd[adList[i]].spendMicro > byAd[adList[j]].spendMicro
			})

			maxImpressions := int64(0)
			for _, aid := range adList {
				if byAd[aid].impressions > maxImpressions {
					maxImpressions = byAd[aid].impressions
				}
			}

			tw := fmtx.TableWriter{
				Indent: "      ",
				Out:    w,
				Cols: []fmtx.TablCol{
					{Header: "Ad", Width: 32},
					{Header: "Impressions", Width: 11, Alignment: fmtx.AlignRight},
					{Header: "Impr", Width: 10},
					{Header: "Clicks", Width: 8, Alignment: fmtx.AlignRight},
					{Header: "CTR", Width: 7, Alignment: fmtx.AlignRight},
					{Header: "CTR sig", Width: 8, Alignment: fmtx.AlignRight},
					{Header: "eCPM", Width: 7, Alignment: fmtx.AlignRight},
					{Header: "CPC", Width: 7, Alignment: fmtx.AlignRight},
					{Header: "Spend(USD)", Width: 11, Alignment: fmtx.AlignRight},
				},
			}
			if showID {
				tw.Cols = append([]fmtx.TablCol{{Header: "ID", Width: 20}}, tw.Cols...)
			}

			agECPM, agCPC := float64(0), float64(0)
			agN := 0
			for _, aid := range adList {
				agECPM += byAd[aid].ecpmMicro
				agCPC += byAd[aid].cpcMicro
				agN += byAd[aid].n
			}
			if agN > 0 {
				agECPM = agECPM / float64(agN) / 1_000_000
				agCPC = agCPC / float64(agN) / 1_000_000
			}

			tw.WriteHeader()
			subheader := []string{
				"",
				strconv.FormatInt(agAgg.impressions, 10),
				"",
				strconv.FormatInt(agAgg.clicks, 10),
				strconv.FormatFloat(agCTR*100, 'f', 2, 64) + "%",
				"",
				strconv.FormatFloat(agECPM, 'f', 2, 64),
				strconv.FormatFloat(agCPC, 'f', 2, 64),
				strconv.FormatFloat(float64(agAgg.spendMicro)/1_000_000, 'f', 2, 64),
			}
			if showID {
				subheader = append([]string{""}, subheader...)
			}
			tw.WriteSubHeader(subheader...)
			tw.WriteHeaderLine()

			for _, aid := range adList {
				a := byAd[aid]
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
					ctrStr = fmtx.ColorizeDist(strconv.FormatFloat(ctr*100, 'f', 2, 64)+"%", ctr, []float64{agCTR * 0.7, agCTR}, []string{fmtx.Red, fmtx.Yellow, fmtx.Green})
				}

				z, zOK := goredditads.CTRZScore(a.clicks, a.impressions, agAgg.clicks, agAgg.impressions)

				name := string(aid)
				if ad := db.GetAd(aid); ad != nil {
					name = ad.Name
				}

				row := []string{
					name,
					strconv.FormatInt(a.impressions, 11),
					fmtx.VolumeBar(a.impressions, maxImpressions, 10),
					strconv.FormatInt(a.clicks, 10),
					ctrStr,
					goredditads.CTRSigStr(z, zOK),
					strconv.FormatFloat(ecpm, 'f', 2, 64),
					strconv.FormatFloat(cpc, 'f', 2, 64),
					strconv.FormatFloat(spend, 'f', 2, 64),
				}
				if showID {
					row = append([]string{fmtx.DimS(string(aid))}, row...)
				}
				tw.WriteRow(row...)
			}
		}
	}
	w.WriteString("\n")
}

func Run(args []string) {
	fs := flag.NewFlagSet("stats ads", flag.ExitOnError)
	var (
		configPath     string
		secretsPath    string
		campaignIDsStr string
		adGroupIDsStr  string
		adIDsStr       string
		showID         bool
		from, until    time.Time
	)
	fs.StringVar(&configPath, "config", "reddit-ads/config.json", "config JSON file path")
	fs.StringVar(&secretsPath, "secrets", "reddit-ads/secrets.json", "secrets JSON file path")
	fs.BoolVar(&showID, "id", false, "show IDs")
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

	metrics, err := client.FetchReport(ctx, from, until,
		[]goredditads.ReportBreakdown{goredditads.ReportBreakdownCampaignID, goredditads.ReportBreakdownAdGroupID, goredditads.ReportBreakdownAdID},
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

	printStats(os.Stdout, showID, metrics, db)
}
