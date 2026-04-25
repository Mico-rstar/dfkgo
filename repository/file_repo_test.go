package repository

import (
	"dfkgo/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestUser(t *testing.T, repo *UserRepo) *model.User {
	user := &model.User{
		Email:        "filetest@example.com",
		PasswordHash: "$2a$10$fakehash",
	}
	err := repo.Create(user)
	require.NoError(t, err)
	return user
}

func TestFileRepo_CreateAndFindByFileUID(t *testing.T) {
	db, err := InitTestDB()
	require.NoError(t, err)
	userRepo := NewUserRepo(db)
	repo := NewFileRepo(db)

	user := createTestUser(t, userRepo)
	file := &model.File{
		FileUID:      "file_abc123",
		UserID:       user.ID,
		FileName:     "test.jpg",
		MimeType:     "image/jpeg",
		FileSize:     1024,
		Modality:     "image",
		MD5:          "d41d8cd98f00b204e9800998ecf8427e",
		OSSBucket:    "dfkgo-files",
		OSSObjectKey: "files/1/file_abc123.jpg",
		OSSURL:       "https://dfkgo-files.oss.aliyuncs.com/files/1/file_abc123.jpg",
		UploadStatus: "pending",
	}
	err = repo.Create(file)
	assert.NoError(t, err)
	assert.NotZero(t, file.ID)

	found, err := repo.FindByFileUID("file_abc123")
	assert.NoError(t, err)
	assert.Equal(t, "test.jpg", found.FileName)
}

func TestFileRepo_FindByUserAndMD5(t *testing.T) {
	db, err := InitTestDB()
	require.NoError(t, err)
	userRepo := NewUserRepo(db)
	repo := NewFileRepo(db)

	user := createTestUser(t, userRepo)
	file := &model.File{
		FileUID:      "file_md5test",
		UserID:       user.ID,
		FileName:     "test.png",
		MimeType:     "image/png",
		FileSize:     2048,
		Modality:     "image",
		MD5:          "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
		OSSBucket:    "dfkgo-files",
		OSSObjectKey: "files/1/file_md5test.png",
		OSSURL:       "https://dfkgo-files.oss.aliyuncs.com/files/1/file_md5test.png",
		UploadStatus: "completed",
	}
	err = repo.Create(file)
	require.NoError(t, err)

	found, err := repo.FindByUserAndMD5(user.ID, "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4")
	assert.NoError(t, err)
	assert.Equal(t, "file_md5test", found.FileUID)
}

func TestFileRepo_UpdateUploadStatus(t *testing.T) {
	db, err := InitTestDB()
	require.NoError(t, err)
	userRepo := NewUserRepo(db)
	repo := NewFileRepo(db)

	user := createTestUser(t, userRepo)
	file := &model.File{
		FileUID:      "file_status",
		UserID:       user.ID,
		FileName:     "vid.mp4",
		MimeType:     "video/mp4",
		FileSize:     4096,
		Modality:     "video",
		MD5:          "00112233445566778899aabbccddeeff",
		OSSBucket:    "dfkgo-files",
		OSSObjectKey: "files/1/file_status.mp4",
		OSSURL:       "https://dfkgo-files.oss.aliyuncs.com/files/1/file_status.mp4",
		UploadStatus: "pending",
	}
	err = repo.Create(file)
	require.NoError(t, err)

	err = repo.UpdateUploadStatus("file_status", "completed")
	assert.NoError(t, err)

	found, err := repo.FindByFileUID("file_status")
	assert.NoError(t, err)
	assert.Equal(t, "completed", found.UploadStatus)
}

func TestFileRepo_FindByID(t *testing.T) {
	db, err := InitTestDB()
	require.NoError(t, err)
	userRepo := NewUserRepo(db)
	repo := NewFileRepo(db)

	user := createTestUser(t, userRepo)
	file := &model.File{
		FileUID:      "file_byid",
		UserID:       user.ID,
		FileName:     "audio.wav",
		MimeType:     "audio/wav",
		FileSize:     8192,
		Modality:     "audio",
		MD5:          "ffeeddccbbaa99887766554433221100",
		OSSBucket:    "dfkgo-files",
		OSSObjectKey: "files/1/file_byid.wav",
		OSSURL:       "https://dfkgo-files.oss.aliyuncs.com/files/1/file_byid.wav",
		UploadStatus: "completed",
	}
	err = repo.Create(file)
	require.NoError(t, err)

	found, err := repo.FindByID(file.ID)
	assert.NoError(t, err)
	assert.Equal(t, "file_byid", found.FileUID)
}
