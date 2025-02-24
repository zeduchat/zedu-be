package migrations

import "github.com/hngprojects/telex_be/internal/models"

// _ = db.AutoMigrate(MigrationModels()...)
func AuthMigrationModels() []interface{} {
	return []interface{}{
		models.GeneralInvitation{},
		models.Group{},
		models.ChannelIntegrationSettings{},
		models.Testimonial{},
		models.APIStatus{},
		models.NewsLetter{},
		models.TelexSlackChannelMapping{},
		models.SlackTelex{},
		models.SlackToken{},
		models.ContactUs{},
		models.BlogFeedback{},
		models.BlogCategory{},
		models.Blog{},
		models.User{},
		models.AccessToken{},
		models.Channels{},
		models.Profile{},
		models.UserChannels{},
		models.Message{},
		models.MagicLink{},
		models.PasswordReset{},
		models.Organisation{},
		models.Permission{},
		models.OrgRole{},
		models.LoginActivity{},
		models.Threads{},
		models.ChannelInvitation{},
		models.Invitation{},
		models.Webhook{},
		models.HistoryWebhook{},
		models.HistoryWebhook{},
		models.OrgUserManagement{},
		models.Mentions{},
		models.HelpCenterCategory{},
		models.HelpCenterArticle{},
		models.Integrations{},
		models.IntegrationSettings{},
		models.NotificationPreferences{},
		models.Plan{},
		models.OrganisationPlan{},
		models.ProcessedStripeWebhook{},
		models.OptIn{},
		models.OrganisationIntegrations{},
		models.OrganisationChannelsIntegrations{},
		models.SlashCommand{},
		models.IntegrationChannel{},
		models.IntegrationOutput{},
		models.CustomIntegrationsSetting{},
	} // an array of db models, example: User{}
}

func AlterColumnModels() []AlterColumn {
	return []AlterColumn{
		// {
		// 	Model: models.OrgUserManagement{},
		// 	TableName: "org_user_managements",
		// 	Column: "role_id",
		// },
		// {
		// 	Model: models.OrgUserManagement{},
		// 	TableName: "org_user_managements",
		// 	Column: "role_id",
		// 	Type:  "uuid",

		// },
	}
}
