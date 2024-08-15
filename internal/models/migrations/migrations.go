package migrations

import "github.com/hngprojects/telex_be/internal/models"

// _ = db.AutoMigrate(MigrationModels()...)
func AuthMigrationModels() []interface{} {
	return []interface{}{
		models.ContactUs{},
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
		models.Invitation{},
		models.Invitation{},
		models.Webhook{},
		models.WebhookHistory{},
		models.OrgUserManagement{},
		models.Mentions{},
		models.HelpCenterCategory{},
		models.HelpCenterArticle{},
	} // an array of db models, example: User{}
}

func AlterColumnModels() []AlterColumn {
	return []AlterColumn{
		{
			Model:     &models.OrgRole{},
			TableName: "org_roles",
			Column:    "organisation_id",
			Type:      "uuid",
		},
		{
			Model:     &models.Permission{},
			TableName: "permissions",
			Column:    "category",
			Type:      "varchar",
		},
	}
}
