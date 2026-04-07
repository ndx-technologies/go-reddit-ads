package goredditads

import (
	"math"
	"time"
)

// DateOnly wraps time.Time and unmarshals JSON strings in "2006-01-02" format.
type DateOnly struct{ time.Time }

func (d *DateOnly) UnmarshalJSON(b []byte) error {
	s := string(b)
	if len(s) >= 2 {
		s = s[1 : len(s)-1] // strip quotes
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}

// ReportMetric is one aggregated row returned by POST /ad_accounts/{id}/reports.
// spend, cpc, ecpm are raw microcurrency values; call Spend(), CPC(), ECPM() to get real values.
type ReportMetric struct {
	Date        DateOnly   `json:"date"`
	CampaignID  CampaignID `json:"campaign_id"`
	AdGroupID   AdGroupID  `json:"ad_group_id"`
	AdID        AdID       `json:"ad_id"`
	Impressions int64      `json:"impressions"`
	Clicks      int64      `json:"clicks"`
	CTR         float64    `json:"ctr"`   // ratio, e.g. 0.05 = 5%
	SpendMicro  int64      `json:"spend"` // microcurrency, ÷1,000,000
	CPCMicro    float64    `json:"cpc"`   // microcurrency, ÷1,000,000
	ECPMMicro   float64    `json:"ecpm"`  // microcurrency, ÷1,000,000
}

func (m ReportMetric) Spend() float64 { return float64(m.SpendMicro) / 1_000_000 }
func (m ReportMetric) CPC() float64   { return m.CPCMicro / 1_000_000 }
func (m ReportMetric) ECPM() float64  { return m.ECPMMicro / 1_000_000 }

type ReportBreakdown string

const (
	ReportBreakdownAdAccountID ReportBreakdown = "AD_ACCOUNT_ID"
	ReportBreakdownAdGroupID   ReportBreakdown = "AD_GROUP_ID"
	ReportBreakdownAdID        ReportBreakdown = "AD_ID"
	ReportBreakdownCampaignID  ReportBreakdown = "CAMPAIGN_ID"
	ReportBreakdownCountry     ReportBreakdown = "COUNTRY"
	ReportBreakdownDate        ReportBreakdown = "DATE"
	ReportBreakdownHour        ReportBreakdown = "HOUR"
	ReportBreakdownGender      ReportBreakdown = "GENDER"
	ReportBreakdownInterest    ReportBreakdown = "INTEREST"
	ReportBreakdownKeyword     ReportBreakdown = "KEYWORD"
	ReportBreakdownPlacement   ReportBreakdown = "PLACEMENT"
	ReportBreakdownOSType      ReportBreakdown = "OS_TYPE"
	ReportBreakdownRegion      ReportBreakdown = "REGION"
	ReportBreakdownCommunity   ReportBreakdown = "COMMUNITY"
)

type ReportField string

const (
	Impressions ReportField = "IMPRESSIONS"
	Clicks      ReportField = "CLICKS"
	CTR         ReportField = "CTR"
	ECPM        ReportField = "ECPM"
	CPC         ReportField = "CPC"
	Spend       ReportField = "SPEND"
)

// CTRZScore computes the two-proportion z-score comparing this row's CTR against a reference total.
func CTRZScore(clicks, imps, totalClicks, totalImps int64) (float64, bool) {
	if imps <= 0 || totalImps <= 0 {
		return 0, false
	}
	p1 := float64(clicks) / float64(imps)
	p2 := float64(totalClicks) / float64(totalImps)
	pHat := float64(clicks+totalClicks) / float64(imps+totalImps)
	se := math.Sqrt(pHat * (1 - pHat) * (1.0/float64(imps) + 1.0/float64(totalImps)))
	if se <= 0 {
		return 0, false
	}
	return (p1 - p2) / se, true
}
