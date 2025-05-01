package worker

import (
	"github.com/Hack-Nocturne/cfs3/types"
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
		fileContainer := types.FileContainer{
			Path:        obj.Path,
			SizeInBytes: obj.SizeInBytes,
			Hash:        obj.Hash,
			ContentType: obj.ContentType,
		}

		objectsMap[obj.Path] = fileContainer
	}

	return objectsMap
}
