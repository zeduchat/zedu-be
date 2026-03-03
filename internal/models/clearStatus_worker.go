package models

type ClearUserStatusJobArgs struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
}

func (ClearUserStatusJobArgs) Kind() string { return "clear_user_status" }
