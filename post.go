package goredditads

type PostType string

const (
	PostTypeCarousel PostType = "CAROUSEL"
	PostTypeImage    PostType = "IMAGE"
	PostTypeText     PostType = "TEXT"
	PostTypeVideo    PostType = "VIDEO"
)

type CallToAction string

const (
	CallToActionApplyNow      CallToAction = "Apply Now"
	CallToActionContactUs     CallToAction = "Contact Us"
	CallToActionDownload      CallToAction = "Download"
	CallToActionGetAQuote     CallToAction = "Get a Quote"
	CallToActionGetShowtimes  CallToAction = "Get Showtimes"
	CallToActionInstall       CallToAction = "Install"
	CallToActionLearnMore     CallToAction = "Learn More"
	CallToActionOrderNow      CallToAction = "Order Now"
	CallToActionPlayNow       CallToAction = "Play Now"
	CallToActionPreorderNow   CallToAction = "Pre-order Now"
	CallToActionSeeMenu       CallToAction = "See Menu"
	CallToActionShopNow       CallToAction = "Shop Now"
	CallToActionSignUp        CallToAction = "Sign Up"
	CallToActionViewMore      CallToAction = "View More"
	CallToActionWatchNow      CallToAction = "Watch Now"
	CallToActionBookNow       CallToAction = "Book Now"
	CallToActionBuyTickets    CallToAction = "Buy Tickets"
	CallToActionGetDirections CallToAction = "Get Directions"
	CallToActionListenNow     CallToAction = "Listen Now"
	CallToActionReadMore      CallToAction = "Read More"
	CallToActionSubscribe     CallToAction = "Subscribe"
	CallToActionVisitStore    CallToAction = "Visit Store"
	CallToActionDonateNow     CallToAction = "Donate Now"
	CallToActionRemindMe      CallToAction = "Remind Me"
)

type PostContent struct {
	MediaURL       string       `json:"media_url,omitempty"`
	DestinationURL string       `json:"destination_url,omitempty"`
	CallToAction   CallToAction `json:"call_to_action,omitempty"`
	Caption        string       `json:"caption,omitempty"`
}

type Post struct {
	ID           PostID        `json:"id,omitempty"`
	Headline     string        `json:"headline,omitempty"`
	Type         PostType      `json:"type,omitempty"`
	Content      []PostContent `json:"content,omitempty"`
	ThumbnailURL string        `json:"thumbnail_url,omitempty"`
	PostURL      string        `json:"post_url,omitempty"`
}
