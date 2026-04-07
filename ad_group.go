package goredditads

type AdGroupID string

func (s AdGroupID) IsZero() bool { return s == "" }

type AdGroup struct {
	ID               AdGroupID         `json:"id"`
	CampaignID       CampaignID        `json:"campaign_id"`
	Name             string            `json:"name"`
	ConfiguredStatus Status            `json:"configured_status,omitempty"`
	BidType          BidType           `json:"bid_type,omitempty"`
	BidValue         float64           `json:"bid_value,omitempty"`
	BidStrategy      BidStrategy       `json:"bid_strategy,omitempty"`
	GoalType         GoalType          `json:"goal_type,omitempty"`
	GoalValue        int64             `json:"goal_value,omitempty"`
	OptimizationGoal OptimizationGoal  `json:"optimization_goal,omitempty"`
	StartTime        string            `json:"start_time,omitempty"`
	EndTime          string            `json:"end_time,omitempty"`
	Targeting        *AdGroupTargeting `json:"targeting,omitempty"`
}

type AdGroupTargeting struct {
	Geolocations              []string        `json:"geolocations,omitempty"`
	ExcludedGeolocations      []string        `json:"excluded_geolocations,omitempty"`
	Communities               []string        `json:"communities,omitempty"`
	ExcludedCommunities       []string        `json:"excluded_communities,omitempty"`
	Keywords                  []string        `json:"keywords,omitempty"`
	ExcludedKeywords          []string        `json:"excluded_keywords,omitempty"`
	Interests                 []string        `json:"interests,omitempty"`
	ExcludedInterests         []string        `json:"excluded_interests,omitempty"`
	CustomAudienceIDs         []string        `json:"custom_audience_ids,omitempty"`
	ExcludedCustomAudienceIDs []string        `json:"excluded_custom_audience_ids,omitempty"`
	Carriers                  []string        `json:"carriers,omitempty"`
	Locations                 []AdPlacement   `json:"locations,omitempty"`
	Devices                   []AdGroupDevice `json:"devices,omitempty"`
	ExpandTargeting           *bool           `json:"expand_targeting,omitempty"`
	Gender                    Gender          `json:"gender,omitempty"`
	Platforms                 []Platform      `json:"platforms,omitempty"`
}

type OptimizationGoal string

const (
	OptimizationGoalAddToCart                         OptimizationGoal = "ADD_TO_CART"
	OptimizationGoalAddToWishlist                     OptimizationGoal = "ADD_TO_WISHLIST"
	OptimizationGoalClicks                            OptimizationGoal = "CLICKS"
	OptimizationGoalLead                              OptimizationGoal = "LEAD"
	OptimizationGoalPageVisit                         OptimizationGoal = "PAGE_VISIT"
	OptimizationGoalPurchase                          OptimizationGoal = "PURCHASE"
	OptimizationGoalSearch                            OptimizationGoal = "SEARCH"
	OptimizationGoalSignUp                            OptimizationGoal = "SIGN_UP"
	OptimizationGoalViewContent                       OptimizationGoal = "VIEW_CONTENT"
	OptimizationGoalMobileConversionInstall           OptimizationGoal = "MOBILE_CONVERSION_INSTALL"
	OptimizationGoalMobileConversionSignUp            OptimizationGoal = "MOBILE_CONVERSION_SIGN_UP"
	OptimizationGoalMobileConversionAddPaymentInfo    OptimizationGoal = "MOBILE_CONVERSION_ADD_PAYMENT_INFO"
	OptimizationGoalMobileConversionAddToCart         OptimizationGoal = "MOBILE_CONVERSION_ADD_TO_CART"
	OptimizationGoalMobileConversionPurchase          OptimizationGoal = "MOBILE_CONVERSION_PURCHASE"
	OptimizationGoalMobileConversionCompletedTutorial OptimizationGoal = "MOBILE_CONVERSION_COMPLETED_TUTORIAL"
	OptimizationGoalMobileConversionLevelAchieved     OptimizationGoal = "MOBILE_CONVERSION_LEVEL_ACHIEVED"
	OptimizationGoalMobileConversionSpendCredits      OptimizationGoal = "MOBILE_CONVERSION_SPEND_CREDITS"
	OptimizationGoalMobileConversionReinstall         OptimizationGoal = "MOBILE_CONVERSION_REINSTALL"
	OptimizationGoalMobileConversionUnlockAchievement OptimizationGoal = "MOBILE_CONVERSION_UNLOCK_ACHIEVEMENT"
	OptimizationGoalMobileConversionStartTrial        OptimizationGoal = "MOBILE_CONVERSION_START_TRIAL"
	OptimizationGoalMobileConversionSubscribe         OptimizationGoal = "MOBILE_CONVERSION_SUBSCRIBE"
	OptimizationGoalMobileConversionOnboardStarted    OptimizationGoal = "MOBILE_CONVERSION_ONBOARD_STARTED"
	OptimizationGoalMobileConversionFirstTimePurchase OptimizationGoal = "MOBILE_CONVERSION_FIRST_TIME_PURCHASE"
	OptimizationGoalVideoView6S                       OptimizationGoal = "VIDEO_VIEW_6S"
	OptimizationGoalVideoView15S                      OptimizationGoal = "VIDEO_VIEW_15S"
	OptimizationGoalLandingPageVisit                  OptimizationGoal = "LANDING_PAGE_VISIT"
)

type Gender string

const (
	GenderFemale Gender = "FEMALE"
	GenderMale   Gender = "MALE"
)

type AdGroupDevice struct {
	OS         DeviceOS   `json:"os,omitempty"`
	Type       DeviceType `json:"type,omitempty"`
	MinVersion string     `json:"min_version,omitempty"`
	MaxVersion string     `json:"max_version,omitempty"`
}

type DeviceOS string

const (
	DeviceOSAndroid DeviceOS = "ANDROID"
	DeviceOSIOS     DeviceOS = "IOS"
)

type DeviceType string

const (
	DeviceTypeDesktop DeviceType = "DESKTOP"
	DeviceTypeMobile  DeviceType = "MOBILE"
)

type Platform string

const (
	PlatformAll           Platform = "ALL"
	PlatformDesktop       Platform = "DESKTOP"
	PlatformDesktopLegacy Platform = "DESKTOP_LEGACY"
	PlatformMobileNative  Platform = "MOBILE_NATIVE"
	PlatformMobileWeb     Platform = "MOBILE_WEB"
	PlatformMobileWeb3X   Platform = "MOBILE_WEB_3X"
	PlatformShredtop      Platform = "SHREDTOP"
)

type AdPlacement string

const (
	AdPlacementFeed         AdPlacement = "FEED"
	AdPlacementCommentsPage AdPlacement = "COMMENTS_PAGE"
)
