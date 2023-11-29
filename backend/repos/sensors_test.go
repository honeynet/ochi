package repos

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestGetSesors(t *testing.T) {
	
}


func initRepoForSensors(t *testing.T) *SensorRepo {
	tmp := t.TempDir()
	dbPath := fmt.Sprintf("%s/test.db", tmp)
	
	fmt.Println("path ", dbPath)
	db, err := sqlx.Connect("sqlite3", dbPath)
	require.NoError(t, err)

	// defer os.Remove("./querytest.db")
	r, err := NewSensorRepo(db)
	require.NoError(t, err)
	require.NotNil(t, r)
	return r
}

