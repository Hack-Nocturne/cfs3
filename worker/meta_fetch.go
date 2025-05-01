package worker

import (
	"math/rand"
	"path/filepath"

	"github.com/Hack-Nocturne/cfs3/types"
	"github.com/Hack-Nocturne/cfs3/utils"
)

func FetchAllMeta(projName string) (map[string]types.FileContainer, error) {
	var objects []Object
	if err := db.Where("project_name = ?", projName).Find(&objects).Error; err != nil {
		return nil, err
	}

	return createObjectMap(objects), nil
}

func FetchAllMetaExcluding(projName string, ids []int64) (map[string]types.FileContainer, error) {
	var objects []Object
	if err := db.Where("project_name = ? AND id NOT IN ?", projName, ids).Find(&objects).Error; err != nil {
		return nil, err
	}

	return createObjectMap(objects), nil
}

func createObjectMap(objects []Object) map[string]types.FileContainer {
	objectsMap := make(map[string]types.FileContainer, len(objects))

	for _, obj := range objects {
		// Cloudflare only cares about the file hash so fake other fields
		fileContainer := types.FileContainer{
			ContentType: utils.ExtToMimeType(filepath.Ext(obj.RelPath)),
			Path:        "/home/admin/" + obj.RelPath,
			SizeInBytes: rand.Int63n(10485760) + 1024, // Random size between 1KB and 10MB
			Hash:        obj.Hash,
		}

		objectsMap[obj.RelPath] = fileContainer
	}

	return objectsMap
}
