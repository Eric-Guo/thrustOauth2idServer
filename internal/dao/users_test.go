package dao

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"github.com/go-dev-frame/sponge/pkg/gotest"
	"github.com/go-dev-frame/sponge/pkg/sgorm/query"
	"github.com/go-dev-frame/sponge/pkg/utils"

	"thrust_oauth2id/internal/cache"
	"thrust_oauth2id/internal/database"
	"thrust_oauth2id/internal/model"
)

func newUsersDao() *gotest.Dao {
	testData := &model.Users{}
	testData.ID = 1
	// you can set the other fields of testData here, such as:
	//testData.CreatedAt = time.Now()
	//testData.UpdatedAt = testData.CreatedAt

	// init mock cache
	//c := gotest.NewCache(map[string]interface{}{"no cache": testData}) // to test mysql, disable caching
	c := gotest.NewCache(map[string]interface{}{utils.Uint64ToStr(testData.ID): testData})
	c.ICache = cache.NewUsersCache(&database.CacheType{
		CType: "redis",
		Rdb:   c.RedisClient,
	})

	// init mock dao
	d := gotest.NewDao(c, testData)
	d.IDao = NewUsersDao(d.DB, c.ICache.(cache.UsersCache))

	return d
}

func Test_usersDao_Create(t *testing.T) {
	d := newUsersDao()
	defer d.Close()
	testData := d.TestData.(*model.Users)

	d.SQLMock.ExpectBegin()
	d.SQLMock.ExpectExec("INSERT INTO .*").
		WithArgs(d.GetAnyArgs(testData)...).
		WillReturnResult(sqlmock.NewResult(1, 1))
	d.SQLMock.ExpectCommit()

	err := d.IDao.(UsersDao).Create(d.Ctx, testData)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_usersDao_DeleteByID(t *testing.T) {
	d := newUsersDao()
	defer d.Close()
	testData := d.TestData.(*model.Users)
	expectedSQLForDeletion := "DELETE .*"

	d.SQLMock.ExpectBegin()
	d.SQLMock.ExpectExec(expectedSQLForDeletion).
		WithArgs(testData.ID).
		WillReturnResult(sqlmock.NewResult(int64(testData.ID), 1))
	d.SQLMock.ExpectCommit()

	err := d.IDao.(UsersDao).DeleteByID(d.Ctx, testData.ID)
	if err != nil {
		t.Fatal(err)
	}

	// zero id error
	err = d.IDao.(UsersDao).DeleteByID(d.Ctx, 0)
	assert.Error(t, err)
}

func Test_usersDao_UpdateByID(t *testing.T) {
	d := newUsersDao()
	defer d.Close()
	testData := d.TestData.(*model.Users)

	d.SQLMock.ExpectBegin()
	d.SQLMock.ExpectExec("UPDATE .*").
		WithArgs(d.AnyTime, testData.ID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	d.SQLMock.ExpectCommit()

	err := d.IDao.(UsersDao).UpdateByID(d.Ctx, testData)
	if err != nil {
		t.Fatal(err)
	}

	// zero id error
	err = d.IDao.(UsersDao).UpdateByID(d.Ctx, &model.Users{})
	assert.Error(t, err)

}

func TestUsersDaoUpdateTimestamps(t *testing.T) {
	stamp := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		user   model.Users
		update string
		args   []driver.Value
	}{
		{
			name:   "zero timestamps are omitted",
			update: "`updated_at`=?",
		},
		{
			name: "all timestamps are updated",
			user: model.Users{
				ConfirmationSentAt:  stamp,
				ConfirmedAt:         stamp.Add(time.Second),
				CurrentSignInAt:     stamp.Add(2 * time.Second),
				LastSignInAt:        stamp.Add(3 * time.Second),
				LockedAt:            stamp.Add(4 * time.Second),
				RememberCreatedAt:   stamp.Add(5 * time.Second),
				ResetPasswordSentAt: stamp.Add(6 * time.Second),
			},
			update: "`confirmation_sent_at`=?,`confirmed_at`=?,`current_sign_in_at`=?,`last_sign_in_at`=?," +
				"`locked_at`=?,`remember_created_at`=?,`reset_password_sent_at`=?,`updated_at`=?",
			args: []driver.Value{
				stamp, stamp.Add(time.Second), stamp.Add(2 * time.Second), stamp.Add(3 * time.Second),
				stamp.Add(4 * time.Second), stamp.Add(5 * time.Second), stamp.Add(6 * time.Second),
			},
		},
		{
			name: "only nonzero timestamps are updated",
			user: model.Users{
				CurrentSignInAt: stamp,
				LockedAt:        stamp.Add(time.Second),
			},
			update: "`current_sign_in_at`=?,`locked_at`=?,`updated_at`=?",
			args:   []driver.Value{stamp, stamp.Add(time.Second)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := gotest.NewDao(nil, &tt.user)
			t.Cleanup(d.Close)
			tt.user.ID = 42

			d.SQLMock.ExpectBegin()
			d.SQLMock.ExpectExec(regexp.QuoteMeta("UPDATE `users` SET " + tt.update + " WHERE `id` = ?")).
				WithArgs(append(tt.args, d.AnyTime, tt.user.ID)...).
				WillReturnResult(sqlmock.NewResult(0, 1))
			d.SQLMock.ExpectCommit()

			if err := NewUsersDao(d.DB, nil).UpdateByID(d.Ctx, &tt.user); err != nil {
				t.Fatal(err)
			}
			if err := d.SQLMock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func Test_usersDao_GetByID(t *testing.T) {
	d := newUsersDao()
	defer d.Close()
	testData := d.TestData.(*model.Users)

	// column names and corresponding data
	rows := sqlmock.NewRows([]string{"id"}).
		AddRow(testData.ID)

	d.SQLMock.ExpectQuery("SELECT .*").
		WithArgs(testData.ID, 1).
		WillReturnRows(rows)

	_, err := d.IDao.(UsersDao).GetByID(d.Ctx, testData.ID)
	if err != nil {
		t.Fatal(err)
	}

	err = d.SQLMock.ExpectationsWereMet()
	if err != nil {
		t.Fatal(err)
	}

	// notfound error
	d.SQLMock.ExpectQuery("SELECT .*").
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	_, err = d.IDao.(UsersDao).GetByID(d.Ctx, 2)
	assert.ErrorIs(t, err, database.ErrRecordNotFound)

	wantErr := errors.New("database unavailable")
	d.SQLMock.ExpectQuery("SELECT .*").
		WithArgs(4, 1).
		WillReturnError(wantErr)
	_, err = d.IDao.(UsersDao).GetByID(d.Ctx, 4)
	assert.ErrorIs(t, err, wantErr)
	assert.NoError(t, d.SQLMock.ExpectationsWereMet())
}

func Test_usersDao_GetByColumns(t *testing.T) {
	d := newUsersDao()
	defer d.Close()
	testData := d.TestData.(*model.Users)

	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow(testData.ID, testData.CreatedAt, testData.UpdatedAt)

	d.SQLMock.ExpectQuery("SELECT .*").WillReturnRows(rows)

	_, _, err := d.IDao.(UsersDao).GetByColumns(d.Ctx, &query.Params{
		Page:  0,
		Limit: 10,
		Sort:  "ignore count", // ignore test count(*)
	})
	if err != nil {
		t.Fatal(err)
	}

	err = d.SQLMock.ExpectationsWereMet()
	if err != nil {
		t.Fatal(err)
	}

	// err test
	_, _, err = d.IDao.(UsersDao).GetByColumns(d.Ctx, &query.Params{
		Page:  0,
		Limit: 10,
		Columns: []query.Column{
			{
				Name:  "id",
				Exp:   "<",
				Value: 0,
			},
		},
	})
	assert.Error(t, err)

	// error test
	dao := &usersDao{}
	_, _, err = dao.GetByColumns(context.Background(), &query.Params{Columns: []query.Column{{}}})
	t.Log(err)
}

func Test_usersDao_DeleteByIDs(t *testing.T) {
	d := newUsersDao()
	defer d.Close()
	testData := d.TestData.(*model.Users)

	d.SQLMock.ExpectBegin()
	d.SQLMock.ExpectExec("DELETE .*").
		WithArgs(testData.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	d.SQLMock.ExpectCommit()

	err := d.IDao.(UsersDao).DeleteByIDs(d.Ctx, []uint64{testData.ID})
	if err != nil {
		t.Fatal(err)
	}
}

func Test_usersDao_GetByCondition(t *testing.T) {
	d := newUsersDao()
	defer d.Close()
	testData := d.TestData.(*model.Users)

	// column names and corresponding data
	rows := sqlmock.NewRows([]string{"id"}).
		AddRow(testData.ID)

	d.SQLMock.ExpectQuery("SELECT .*").
		WithArgs(testData.ID, 1).
		WillReturnRows(rows)

	_, err := d.IDao.(UsersDao).GetByCondition(d.Ctx, &query.Conditions{
		Columns: []query.Column{
			{
				Name:  "id",
				Value: testData.ID,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = d.SQLMock.ExpectationsWereMet()
	if err != nil {
		t.Fatal(err)
	}

	// notfound error
	d.SQLMock.ExpectQuery("SELECT .*").
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	_, err = d.IDao.(UsersDao).GetByCondition(d.Ctx, &query.Conditions{
		Columns: []query.Column{
			{
				Name:  "id",
				Value: 2,
			},
		},
	})
	assert.ErrorIs(t, err, database.ErrRecordNotFound)
	assert.NoError(t, d.SQLMock.ExpectationsWereMet())
}

func Test_usersDao_GetByIDs(t *testing.T) {
	d := newUsersDao()
	defer d.Close()
	testData := d.TestData.(*model.Users)

	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow(testData.ID, testData.CreatedAt, testData.UpdatedAt)

	d.SQLMock.ExpectQuery("SELECT .*").
		WithArgs(testData.ID).
		WillReturnRows(rows)

	_, err := d.IDao.(UsersDao).GetByIDs(d.Ctx, []uint64{testData.ID})
	if err != nil {
		t.Fatal(err)
	}

	// Success case - no data found for id 111, should return empty map without error
	d.SQLMock.ExpectQuery("SELECT .*").
		WithArgs(111).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}))

	result, err := d.IDao.(UsersDao).GetByIDs(d.Ctx, []uint64{111})
	assert.NoError(t, err)
	assert.Empty(t, result)

	err = d.SQLMock.ExpectationsWereMet()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_usersDao_GetByLastID(t *testing.T) {
	d := newUsersDao()
	defer d.Close()
	testData := d.TestData.(*model.Users)

	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow(testData.ID, testData.CreatedAt, testData.UpdatedAt)

	d.SQLMock.ExpectQuery("SELECT .*").WillReturnRows(rows)

	_, err := d.IDao.(UsersDao).GetByLastID(d.Ctx, 0, 10, "")
	if err != nil {
		t.Fatal(err)
	}

	err = d.SQLMock.ExpectationsWereMet()
	if err != nil {
		t.Fatal(err)
	}

	// err test - unknown column should cause database error
	d.SQLMock.ExpectQuery("SELECT .*").
		WillReturnError(errors.New("unknown column"))

	_, err = d.IDao.(UsersDao).GetByLastID(d.Ctx, 0, 10, "unknown-column")
	assert.Error(t, err)
}

func Test_usersDao_CreateByTx(t *testing.T) {
	d := newUsersDao()
	defer d.Close()
	testData := d.TestData.(*model.Users)

	d.SQLMock.ExpectBegin()
	d.SQLMock.ExpectExec("INSERT INTO .*").
		WithArgs(d.GetAnyArgs(testData)...).
		WillReturnResult(sqlmock.NewResult(1, 1))
	d.SQLMock.ExpectCommit()

	_, err := d.IDao.(UsersDao).CreateByTx(d.Ctx, d.DB, testData)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_usersDao_DeleteByTx(t *testing.T) {
	d := newUsersDao()
	defer d.Close()
	testData := d.TestData.(*model.Users)
	expectedSQLForDeletion := "DELETE .*"

	d.SQLMock.ExpectBegin()
	d.SQLMock.ExpectExec(expectedSQLForDeletion).
		WithArgs(testData.ID).
		WillReturnResult(sqlmock.NewResult(int64(testData.ID), 1))
	d.SQLMock.ExpectCommit()

	err := d.IDao.(UsersDao).DeleteByTx(d.Ctx, d.DB, testData.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_usersDao_UpdateByTx(t *testing.T) {
	d := newUsersDao()
	defer d.Close()
	testData := d.TestData.(*model.Users)

	d.SQLMock.ExpectBegin()
	d.SQLMock.ExpectExec("UPDATE .*").
		WithArgs(d.AnyTime, testData.ID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	d.SQLMock.ExpectCommit()

	err := d.IDao.(UsersDao).UpdateByTx(d.Ctx, d.DB, testData)
	if err != nil {
		t.Fatal(err)
	}
}
