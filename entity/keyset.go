package entity

import (
	"context"

	"github.com/google/uuid"
)

type KeyMaterial struct {
	UserID uuid.UUID
	Scope  string
	Keys   map[string][]byte
}

type keyMaterialContextKey struct{}

func WithKeyMaterial(ctx context.Context, materials map[string]KeyMaterial) context.Context {
	return context.WithValue(ctx, keyMaterialContextKey{}, materials)
}

func KeyMaterialFor(ctx context.Context, scope string) (KeyMaterial, bool) {
	materials, ok := ctx.Value(keyMaterialContextKey{}).(map[string]KeyMaterial)
	if !ok {
		return KeyMaterial{}, false
	}
	material, ok := materials[scope]
	return material, ok
}
