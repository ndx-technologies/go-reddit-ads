package goredditads

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"

	"github.com/nikolaydubina/fpmoney"
)

type JSONDB struct {
	AppID          string           `json:"app_id"`
	AdAccountID    string           `json:"ad_account_id"`
	AppRedirectURI string           `json:"app_redirect_uri"`
	Currency       fpmoney.Currency `json:"currency,omitempty"`
	Campaigns      []CampaignNode   `json:"campaigns"`

	campaignByID map[CampaignID]*CampaignNode
	adGroupByID  map[AdGroupID]*AdGroupNode
	adByID       map[AdID]*Ad
}

func (db JSONDB) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.MarshalWrite(f, db, jsontext.WithIndent("    "))
}

type CampaignNode struct {
	Campaign
	AdGroups []AdGroupNode `json:"ad_groups"`
}

type AdGroupNode struct {
	AdGroup
	Ads []Ad `json:"ads"`
}

func (db *JSONDB) Init() {
	db.campaignByID = make(map[CampaignID]*CampaignNode, len(db.Campaigns))
	db.adGroupByID = make(map[AdGroupID]*AdGroupNode)
	db.adByID = make(map[AdID]*Ad)
	for i := range db.Campaigns {
		c := &db.Campaigns[i]
		db.campaignByID[c.ID] = c
		for j := range c.AdGroups {
			ag := &c.AdGroups[j]
			db.adGroupByID[ag.ID] = ag
			for k := range ag.Ads {
				db.adByID[ag.Ads[k].ID] = &ag.Ads[k]
			}
		}
	}
}

func (db *JSONDB) GetCampaign(id CampaignID) *CampaignNode { return db.campaignByID[id] }
func (db *JSONDB) GetAdGroup(id AdGroupID) *AdGroupNode    { return db.adGroupByID[id] }
func (db *JSONDB) GetAd(id AdID) *Ad                       { return db.adByID[id] }

func (db *JSONDB) UpsertCampaign(c Campaign) {
	if cn := db.campaignByID[c.ID]; cn != nil {
		cn.Campaign = c
		return
	}
	db.Campaigns = append(db.Campaigns, CampaignNode{Campaign: c})
	db.campaignByID[c.ID] = &db.Campaigns[len(db.Campaigns)-1]
}

func (db *JSONDB) UpsertAdGroup(ag AdGroup) {
	cn := db.ensureCampaign(ag.CampaignID)
	if an := db.adGroupByID[ag.ID]; an != nil {
		an.AdGroup = ag
		return
	}
	cn.AdGroups = append(cn.AdGroups, AdGroupNode{AdGroup: ag})
	an := &cn.AdGroups[len(cn.AdGroups)-1]
	db.adGroupByID[ag.ID] = an
}

func (db *JSONDB) UpsertAd(ad Ad) {
	an := db.ensureAdGroup(ad.CampaignID, ad.AdGroupID)
	if existing := db.adByID[ad.ID]; existing != nil {
		*existing = ad
		return
	}
	an.Ads = append(an.Ads, ad)
	db.adByID[ad.ID] = &an.Ads[len(an.Ads)-1]
}

func (db *JSONDB) ensureCampaign(id CampaignID) *CampaignNode {
	if cn := db.campaignByID[id]; cn != nil {
		return cn
	}
	db.Campaigns = append(db.Campaigns, CampaignNode{Campaign: Campaign{ID: id}})
	cn := &db.Campaigns[len(db.Campaigns)-1]
	db.campaignByID[id] = cn
	return cn
}

func (db *JSONDB) ensureAdGroup(campaignID CampaignID, adGroupID AdGroupID) *AdGroupNode {
	if an := db.adGroupByID[adGroupID]; an != nil {
		return an
	}
	cn := db.ensureCampaign(campaignID)
	cn.AdGroups = append(cn.AdGroups, AdGroupNode{AdGroup: AdGroup{ID: adGroupID, CampaignID: campaignID}})
	an := &cn.AdGroups[len(cn.AdGroups)-1]
	db.adGroupByID[adGroupID] = an
	return an
}
