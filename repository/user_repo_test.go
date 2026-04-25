package repository

import (
	"dfkgo/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepo_CreateAndFindByEmail(t *testing.T) {
	db, err := InitTestDB()
	require.NoError(t, err)
	repo := NewUserRepo(db)

	user := &model.User{
		Email:        "test@example.com",
		PasswordHash: "$2a$10$fakehash",
		Nickname:     "tester",
	}
	err = repo.Create(user)
	assert.NoError(t, err)
	assert.NotZero(t, user.ID)

	found, err := repo.FindByEmail("test@example.com")
	assert.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "tester", found.Nickname)
}

func TestUserRepo_FindByID(t *testing.T) {
	db, err := InitTestDB()
	require.NoError(t, err)
	repo := NewUserRepo(db)

	user := &model.User{
		Email:        "findid@example.com",
		PasswordHash: "$2a$10$fakehash",
	}
	err = repo.Create(user)
	require.NoError(t, err)

	found, err := repo.FindByID(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "findid@example.com", found.Email)
}

func TestUserRepo_FindByEmail_NotFound(t *testing.T) {
	db, err := InitTestDB()
	require.NoError(t, err)
	repo := NewUserRepo(db)

	_, err = repo.FindByEmail("nonexistent@example.com")
	assert.Error(t, err)
}

func TestUserRepo_UpdateProfile(t *testing.T) {
	db, err := InitTestDB()
	require.NoError(t, err)
	repo := NewUserRepo(db)

	user := &model.User{
		Email:        "update@example.com",
		PasswordHash: "$2a$10$fakehash",
		Nickname:     "old",
	}
	err = repo.Create(user)
	require.NoError(t, err)

	err = repo.UpdateProfile(user.ID, map[string]any{"nickname": "new"})
	assert.NoError(t, err)

	found, err := repo.FindByID(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "new", found.Nickname)
}

func TestUserRepo_UpdateAvatar(t *testing.T) {
	db, err := InitTestDB()
	require.NoError(t, err)
	repo := NewUserRepo(db)

	user := &model.User{
		Email:        "avatar@example.com",
		PasswordHash: "$2a$10$fakehash",
	}
	err = repo.Create(user)
	require.NoError(t, err)

	err = repo.UpdateAvatar(user.ID, "https://oss.example.com/avatar.jpg")
	assert.NoError(t, err)

	found, err := repo.FindByID(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "https://oss.example.com/avatar.jpg", found.AvatarURL)
}

func TestUserRepo_DuplicateEmail(t *testing.T) {
	db, err := InitTestDB()
	require.NoError(t, err)
	repo := NewUserRepo(db)

	user1 := &model.User{Email: "dup@example.com", PasswordHash: "hash1"}
	err = repo.Create(user1)
	require.NoError(t, err)

	user2 := &model.User{Email: "dup@example.com", PasswordHash: "hash2"}
	err = repo.Create(user2)
	assert.Error(t, err)
}
