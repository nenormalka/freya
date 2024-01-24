package repo

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/nenormalka/freya/conns/connectors/mocks"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestGetNow(t *testing.T) {
	mock, err := mocks.NewSQLXMock()
	if err != nil {
		t.Fatal(err)
	}

	r := &Repo{db: mock}
	tm := time.Now()

	mock.Mock.ExpectQuery(regexp.QuoteMeta(selectNowSQL)).WillReturnRows(sqlmock.NewRows([]string{"now"}).AddRow(tm))

	now, err := r.GetNow(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, tm.String(), now)
	assert.Nil(t, mock.CloseDB())
	assert.Nil(t, mock.Mock.ExpectationsWereMet())
}
