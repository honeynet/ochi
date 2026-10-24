package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPermissionsFor(t *testing.T) {
	assert.Equal(t, []Permission{}, PermissionsFor(nil))
	assert.Equal(t, []Permission{PermListUsers, PermAddSensorByID}, PermissionsFor([]Role{RoleAdmin, RoleAdmin}))
	assert.Equal(t, []Permission{}, PermissionsFor([]Role{"unknown"}))
}

func TestHasPermission(t *testing.T) {
	assert.True(t, HasPermission([]Role{RoleAdmin}, PermListUsers))
	assert.True(t, HasPermission([]Role{RoleAdmin}, PermAddSensorByID))
	assert.False(t, HasPermission(nil, PermListUsers))
	assert.False(t, HasPermission([]Role{"unknown"}, PermAddSensorByID))
}
