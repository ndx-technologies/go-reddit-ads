package goredditads

type AdID string

func (s AdID) IsZero() bool { return s == "" }

type PostID string

func (s PostID) String() string { return string(s) }

type Ad struct {
	ID               AdID       `json:"id"`
	AdGroupID        AdGroupID  `json:"ad_group_id"`
	CampaignID       CampaignID `json:"campaign_id"`
	Name             string     `json:"name"`
	ConfiguredStatus Status     `json:"configured_status,omitempty"`
	ClickURL         string     `json:"click_url,omitempty"`
	PostID           PostID     `json:"post_id,omitempty"`
	PostURL          string     `json:"post_url,omitempty"`
	Post             *Post      `json:"post,omitempty"`
}
