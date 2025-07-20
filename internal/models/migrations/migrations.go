package migrations

import (
	"github.com/hngprojects/telex_be/internal/models"
)

// _ = db.AutoMigrate(MigrationModels()...)
func AuthMigrationModels() []any {
	return []any{
		models.UserPinnedOrganisations{},
		models.OrganisationIntegrations{},
		models.APIStatus{},
		models.AccessToken{},
		models.Blog{},
		models.BlogCategory{},
		models.BlogFeedback{},
		models.ChannelIntegrationSettings{},
		models.ChannelParticipant{},
		models.Group{},
		models.Channels{},
		models.ContactUs{},
		models.CustomIntegrationsSetting{},
		models.DmChannels{},
		models.FcmTokens{},
		models.GeneralInvitation{},
		models.HelpCenterArticle{},
		models.HelpCenterCategory{},
		models.Webhook{},
		models.HistoryWebhook{},
		models.IntegrationOutput{},
		models.IntegrationChannel{},
		models.IntegrationSettings{},
		models.Integrations{},
		models.Invitation{},
		models.LoginActivity{},
		models.MagicLink{},
		models.Mentions{},
		models.NewsLetter{},
		models.NotificationPreferences{},
		models.OptIn{},
		models.OrgRole{},
		models.OrgUserManagement{},
		models.Organisation{},
		models.OrganisationChannelsIntegrations{},
		models.OrganisationPlan{},
		models.PasswordReset{},
		models.Permission{},
		models.PinnedMessage{},
		models.Reaction{},
		models.Plan{},
		models.ProcessedStripeWebhook{},
		models.Profile{},
		models.SavedMessage{},
		models.SlackTelex{},
		models.SlackToken{},
		models.SlashCommand{},
		models.TelexAIUsageLog{},
		models.TelexSlackChannelMapping{},
		models.Testimonial{},
		models.UploadedFileResponse{},
		models.User{},
		models.Admin{},
		models.UserChannels{},
		models.IntegrationBills{},
		models.CreditPackage{},
		models.CreditTransaction{},
		models.CreditUsage{},
	} // an array of db models, example: User{}
}

func AlterColumnModels() []AlterColumn {
	return []AlterColumn{
		// {
		// 	Model: models.OrgUserManagement{},
		// 	TableName: "org_user_managements",
		// 	Column: "is_deactivated",
		// 	Type: "bool",
		// },
	}
}
