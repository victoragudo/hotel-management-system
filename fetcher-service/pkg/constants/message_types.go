package constants

// Message type constants used by both orchestrator and worker services
const (
	MessageTypeUpdateHotel       = "update_hotel"
	MessageTypeUpdateReview      = "update_review"
	MessageTypeUpdateTranslation = "update_translation"
	MessageTypeFetchTranslation  = "fetch_translation"
	MessageTypeFetchReview       = "fetch_review"
)
