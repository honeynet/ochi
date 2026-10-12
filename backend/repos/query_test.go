package repos

import (
	"fmt"
	"testing"

	"github.com/honeynet/ochi/backend/entities"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestQuery(t *testing.T) {
	r := initRepo(t)
	u1, err := r.FindByOwnerId("test@test")
	require.NoError(t, err)
	require.Empty(t, u1)

	q, err := r.Create("123", "hello there", "this is desription", true, nil)
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
	q, err := r.Create("001", "content", "description", true, nil)
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

	q, err := r.Create("001", "content", "description", true, nil)
	require.NoError(t, err)
	require.Equal(t, "001", q.OwnerID)
	require.Equal(t, "content", q.Content)
	require.Equal(t, "description", q.Description)

	q1, err := r.Create("001", "content1", "description1", true, nil)
	require.NoError(t, err)
	require.Equal(t, "001", q1.OwnerID)
	require.Equal(t, "content1", q1.Content)
	require.Equal(t, "description1", q1.Description)

	q2, err := r.Create("001", "content2", "description2", true, nil)
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

	q, err := r.Create("001", "content", "description", true, nil)
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

	err := r.Update("non-existing", "content", "description", true, nil)
	require.ErrorContains(t, err, "non-existing not found")
}

func TestUpdateQuery_Existing(t *testing.T) {
	r := initRepo(t)

	q, err := r.Create("001", "content", "description", true, nil)
	require.NoError(t, err)

	err = r.Update(q.ID, "content-new", "description-new", false, nil)
	require.NoError(t, err)

	q1, err := r.GetByID(q.ID)
	require.NoError(t, err)
	require.Equal(t, "content-new", q1.Content)
	require.Equal(t, "description-new", q1.Description)
	require.Equal(t, false, q1.Active)
}

func initRepo(t *testing.T) *QueryRepo {
	tmp := t.TempDir()
	dbPath := fmt.Sprintf("%s/test.db", tmp)

	db, err := sqlx.Connect("sqlite", dbPath)
	require.NoError(t, err)
	// Release the handle so t.TempDir() can remove the file on Windows.
	t.Cleanup(func() { _ = db.Close() })

	r, err := NewQueryRepo(db)
	require.NoError(t, err)
	require.NotNil(t, r)
	return r
}

func TestQueryTags_RoundTrip(t *testing.T) {
	r := initRepo(t)
	tags := []entities.Tag{{Type: "malware", Value: "mirai"}, {Type: "protocol", Value: "smb"}}

	q, err := r.Create("001", "tcp.port == 445", "smb scans", true, tags)
	require.NoError(t, err)
	require.Equal(t, tags, q.Tags)

	got, err := r.GetByID(q.ID)
	require.NoError(t, err)
	require.Equal(t, tags, got.Tags, "tags come back in the order they were given")

	qs, err := r.FindByOwnerId("001")
	require.NoError(t, err)
	require.Len(t, qs, 1)
	require.Equal(t, tags, qs[0].Tags)
}

func TestQueryTags_UntaggedQueriesHaveAnEmptyList(t *testing.T) {
	r := initRepo(t)
	q, err := r.Create("001", "content", "description", true, nil)
	require.NoError(t, err)

	// Every read returns an empty list rather than nil, so the API sends [] not null.
	got, err := r.GetByID(q.ID)
	require.NoError(t, err)
	require.NotNil(t, got.Tags)
	require.Empty(t, got.Tags)

	qs, err := r.FindByOwnerId("001")
	require.NoError(t, err)
	require.NotNil(t, qs[0].Tags)
}

func TestQueryTags_UpdateReplacesTags(t *testing.T) {
	r := initRepo(t)
	q, err := r.Create("001", "content", "description", true, []entities.Tag{{Type: "a", Value: "1"}, {Type: "b", Value: "2"}})
	require.NoError(t, err)

	require.NoError(t, r.Update(q.ID, "content", "description", true, []entities.Tag{{Type: "c", Value: "3"}}))
	got, err := r.GetByID(q.ID)
	require.NoError(t, err)
	require.Equal(t, []entities.Tag{{Type: "c", Value: "3"}}, got.Tags)

	require.NoError(t, r.Update(q.ID, "content", "description", true, []entities.Tag{}))
	got, err = r.GetByID(q.ID)
	require.NoError(t, err)
	require.Empty(t, got.Tags)
}

func TestQueryTags_DeleteRemovesTags(t *testing.T) {
	r := initRepo(t)
	q, err := r.Create("001", "content", "description", true, []entities.Tag{{Type: "a", Value: "1"}})
	require.NoError(t, err)

	require.NoError(t, r.Delete(q.ID))

	var left int
	require.NoError(t, r.db.Get(&left, "SELECT COUNT(*) FROM query_tags WHERE query_id=?", q.ID))
	require.Zero(t, left, "deleting a query must not leave its tags behind")
}

func TestQueryTags_OwnersDoNotSeeEachOthersTags(t *testing.T) {
	r := initRepo(t)
	_, err := r.Create("alice", "content", "description", true, []entities.Tag{{Type: "owner", Value: "alice"}})
	require.NoError(t, err)
	_, err = r.Create("bob", "content", "description", true, []entities.Tag{{Type: "owner", Value: "bob"}})
	require.NoError(t, err)

	qs, err := r.FindByOwnerId("alice")
	require.NoError(t, err)
	require.Len(t, qs, 1)
	require.Equal(t, []entities.Tag{{Type: "owner", Value: "alice"}}, qs[0].Tags)
}

func TestQueryTags_FailedCreateSavesNothing(t *testing.T) {
	r := initRepo(t)
	// The same tag twice breaks the (query_id, type, value) unique constraint on the
	// second insert, after the query row itself was written.
	dup := []entities.Tag{{Type: "a", Value: "1"}, {Type: "a", Value: "1"}}
	_, err := r.Create("001", "content", "description", true, dup)
	require.Error(t, err)

	qs, err := r.FindByOwnerId("001")
	require.NoError(t, err)
	require.Empty(t, qs, "the query must be rolled back together with its tags")
}

func TestQueryTags_FailedUpdateChangesNothing(t *testing.T) {
	r := initRepo(t)
	q, err := r.Create("001", "content", "description", true, []entities.Tag{{Type: "a", Value: "1"}})
	require.NoError(t, err)

	dup := []entities.Tag{{Type: "b", Value: "2"}, {Type: "b", Value: "2"}}
	require.Error(t, r.Update(q.ID, "new content", "new description", false, dup))

	got, err := r.GetByID(q.ID)
	require.NoError(t, err)
	require.Equal(t, "content", got.Content, "the query must be left as it was")
	require.Equal(t, []entities.Tag{{Type: "a", Value: "1"}}, got.Tags, "and so must its tags")
}
