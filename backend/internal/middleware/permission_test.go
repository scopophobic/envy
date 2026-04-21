package middleware

import (
	"testing"

	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
)

func TestHasPermissionInOrg(t *testing.T) {
	orgA := uuid.New()
	orgB := uuid.New()
	user := &models.User{
		OrgMemberships: []models.OrgMember{
			{
				OrgID: orgA,
				Role: models.Role{
					Permissions: []models.Permission{{Name: models.PermissionMembersManage}},
				},
			},
			{
				OrgID: orgB,
				Role: models.Role{
					Permissions: []models.Permission{{Name: models.PermissionSecretsRead}},
				},
			},
		},
	}

	if !HasPermissionInOrg(user, orgA, models.PermissionMembersManage) {
		t.Fatalf("expected permission in matching org")
	}
	if HasPermissionInOrg(user, orgB, models.PermissionMembersManage) {
		t.Fatalf("permission should not leak across orgs")
	}
}

