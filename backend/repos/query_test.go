package repos

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestQuery(t *testing.T) {
	r := initRepo(t)
	u1, err := r.FindByOwnerId("test@test")
	require.NoError(t, err)
	require.Empty(t, u1)

	q, err := r.Create("123", "hello there", "this is desription", true)
	require.NoError(t, err)
	require.NotEmpty(t, q.ID)

	u1, err = r.FindByOwnerId("123")
	require.NoError(t, err)
	require.Equal(t, len(u1), 1)

	require.Equal(t, u1[0], q)

	err = r.Delete("Not found")
	require.Error(t, err)

	err = r.Delete(q.ID)
	require.NoError(t, err)

	u1, err = r.FindByOwnerId("123")
	require.NoError(t, err)
	require.Empty(t, u1)
}

func TestGetById_Missing(t *testing.T) {
	r := initRepo(t)

	_, err := r.GetByID("non-existing")
	require.ErrorContains(t, err, "sql: no rows in result set")
}

func TestGetById_Existing(t *testing.T) {
	r := initRepo(t)
	q, err := r.Create("001", "content", "description", true)
	require.NoError(t, err)
	require.Equal(t, "001", q.OwnerID)
	require.Equal(t, "content", q.Content)
	require.Equal(t, "description", q.Description)

	q1, err := r.GetByID(q.ID)
	require.NoError(t, err)
	require.Equal(t, q, q1)
}

func TestFindByOwnerId_Missing(t *testing.T) {
	r := initRepo(t)

	qs, err := r.FindByOwnerId("non-existing")
	require.NoError(t, err)
	require.Empty(t, qs)
}

func TestFindByOwnerId_Existing(t *testing.T) {
	r := initRepo(t)

	q, err := r.Create("001", "content", "description", true)
	require.NoError(t, err)
	require.Equal(t, "001", q.OwnerID)
	require.Equal(t, "content", q.Content)
	require.Equal(t, "description", q.Description)

	q1, err := r.Create("001", "content1", "description1", true)
	require.NoError(t, err)
	require.Equal(t, "001", q1.OwnerID)
	require.Equal(t, "content1", q1.Content)
	require.Equal(t, "description1", q1.Description)

	q2, err := r.Create("001", "content2", "description2", true)
	require.NoError(t, err)
	require.Equal(t, "001", q2.OwnerID)
	require.Equal(t, "content2", q2.Content)
	require.Equal(t, "description2", q2.Description)

	qs, err := r.FindByOwnerId("001")
	require.NoError(t, err)
	require.Contains(t, qs, q)
	require.Contains(t, qs, q1)
	require.Contains(t, qs, q2)
}

func TestDeleteQuery_Missing(t *testing.T) {
	r := initRepo(t)

	err := r.Delete("non-existing")
	require.ErrorContains(t, err, "non-existing not found")
}

func TestDeleteQuery_Existing(t *testing.T) {
	r := initRepo(t)

	q, err := r.Create("001", "content", "description", true)
	require.NoError(t, err)

	err = r.Delete(q.ID)
	require.NoError(t, err)

	err = r.Delete(q.ID)
	require.Error(t, err)

	_, err = r.GetByID(q.ID)
	require.Error(t, err)
}

func TestUpdateQuery_Missing(t *testing.T) {
	r := initRepo(t)

	err := r.Update("non-existing", "content", "description", true)
	require.ErrorContains(t, err, "non-existing not found")
}

func TestUpdateQuery_Existing(t *testing.T) {
	r := initRepo(t)

	q, err := r.Create("001", "content", "description", true)
	require.NoError(t, err)

	err = r.Update(q.ID, "content-new", "description-new", false)
	require.NoError(t, err)

	q1, err := r.GetByID(q.ID)
	require.Equal(t, "content-new", q1.Content)
	require.Equal(t, "description-new", q1.Description)
	require.Equal(t, false, q1.Active)
}

func initRepo(t *testing.T) *QueryRepo {
	tmp := t.TempDir()
	dbPath := fmt.Sprintf("%s/test.db", tmp)
	db, err := sqlx.Connect("sqlite3", dbPath)
	require.NoError(t, err)

	// defer os.Remove("./querytest.db")
	r, err := NewQueryRepo(db)
	require.NoError(t, err)
	require.NotNil(t, r)
	return r
}
