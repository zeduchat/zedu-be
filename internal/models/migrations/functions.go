package migrations

type AlterColumn struct {
	Model     any
	TableName string
	Column    string
	Type      string
}
