package sqlx

type (
	Data struct {
		Code   string `db:"code"`
		Source string `db:"source"`
		Data   []byte `db:"data"`
	}
)
