package goredditads

import "time"

type CampaignID string

func (s CampaignID) IsZero() bool { return s == "" }

type Campaign struct {
	ID                           CampaignID  `json:"id"`
	Name                         string      `json:"name"`
	ConfiguredStatus             Status      `json:"configured_status,omitempty"`
	Objective                    Objective   `json:"objective,omitempty"`
	GoalType                     GoalType    `json:"goal_type,omitempty"`
	GoalValue                    int64       `json:"goal_value,omitempty"`
	BidStrategy                  BidStrategy `json:"bid_strategy,omitempty"`
	BidType                      BidType     `json:"bid_type,omitempty"`
	BidValue                     float64     `json:"bid_value,omitempty"`
	SpendCap                     int64       `json:"spend_cap,omitempty"`
	IsCampaignBudgetOptimization *bool       `json:"is_campaign_budget_optimization,omitempty"`
	StartTime                    time.Time   `json:"start_time,omitzero"`
	EndTime                      time.Time   `json:"end_time,omitzero"`
}

type Status string

const (
	Active   Status = "ACTIVE"
	Paused   Status = "PAUSED"
	Archived Status = "ARCHIVED"
	Deleted  Status = "DELETED"
)

type Objective string

const (
	ObjectiveAppInstalls              Objective = "APP_INSTALLS"
	ObjectiveCatalogSales             Objective = "CATALOG_SALES"
	ObjectiveClicks                   Objective = "CLICKS"
	ObjectiveConversions              Objective = "CONVERSIONS"
	ObjectiveImpressions              Objective = "IMPRESSIONS"
	ObjectiveLeadGeneration           Objective = "LEAD_GENERATION"
	ObjectiveVideoViewableImpressions Objective = "VIDEO_VIEWABLE_IMPRESSIONS"
)

type GoalType string

const (
	GoalTypeDailySpend    GoalType = "DAILY_SPEND"
	GoalTypeLifetimeSpend GoalType = "LIFETIME_SPEND"
)

type BidStrategy string

const (
	BidStrategyBidless        BidStrategy = "BIDLESS"
	BidStrategyManualBidding  BidStrategy = "MANUAL_BIDDING"
	BidStrategyMaximizeVolume BidStrategy = "MAXIMIZE_VOLUME"
	BidStrategyTargetCPX      BidStrategy = "TARGET_CPX"
)

type BidType string

const (
	BidTypeCPC  BidType = "CPC"
	BidTypeCPM  BidType = "CPM"
	BidTypeCPV  BidType = "CPV"
	BidTypeCPV6 BidType = "CPV6"
)
