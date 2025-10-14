package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/Hack-Nocturne/cfs3/types"
	"github.com/Hack-Nocturne/cfs3/utils"
	"github.com/Hack-Nocturne/cfs3/vars"
	"github.com/Hack-Nocturne/cfs3/worker"
	_ "github.com/joho/godotenv/autoload"
)

func init() {
	os.MkdirAll(vars.UPLOAD_BASE_DIR, 0o755)
}

func main() {
	defer func() { os.RemoveAll(vars.UPLOAD_BASE_DIR) }()

	configFile := "cfs3.config.json"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	config, cfgErr := worker.LoadNProcessConfig(configFile)
	if cfgErr != nil {
		fmt.Println("❌ Failure loading config:", cfgErr)
		return
	}

	vars.IS_PATCH_MODE = config.Mode == "patch"

	if config.Mode == "remove" {
		meta, meErr := worker.FetchAllMetaExcluding(config.ProjectName, config.FilesRemove)
		if meErr != nil {
			fmt.Println("❌ Failure fetching existing meta:", meErr)
			return
		}

		vars.EXISTING_META = meta
	} else {
		meta, maErr := worker.FetchAllMeta(config.ProjectName)
		if maErr != nil {
			fmt.Println("❌ Failure fetching existing meta:", maErr)
			return
		}

		vars.EXISTING_META = meta
	}

	cloned := utils.Clone(vars.EXISTING_META)
	StartCFUpload(config.ProjectName)

	switch config.Mode {
	case "patch":
		objects := buildObjects(vars.EXISTING_META, cloned, config.FilesPatch, config.By, config.ProjectName)
		worker.BulkAddObjects(objects)
	case "remove":
		worker.BulkRemoveObjects(config.FilesRemove)
	}

	fmt.Println("💾 Metadata updated successfully.")
}

func StartCFUpload(projectName string) error {
	uploadArgs := utils.PagesDeployOptions{
		Directory:   vars.UPLOAD_BASE_DIR,
		AccountId:   vars.CF_ACCOUNT_ID,
		ProjectName: projectName,
		SkipCaching: false,
	}

	deployResp, err := Deploy(uploadArgs)
	if err != nil {
		fmt.Println("❌ Deployment failed: " + err.Error())
		return err
	}

	fmt.Println("💫 Deployment completed with ID: " + deployResp.ID)
	fmt.Println("🌐 Take a peek over " + deployResp.URL)
	return nil
}

func buildObjects(all, existing map[string]types.FileContainer, filePatches []worker.FilePatch, by, projName string) []worker.Object {
	objects := make([]worker.Object, 0, len(all)-len(existing))

	for _, file := range filePatches {
		if _, exists := existing[file.Remote]; exists {
			continue
		}

		fileContainer, exists := all[file.Remote]
		if !exists {
			continue
		}

		metaJsonBytes, mrErr := json.Marshal(file.Metadata)
		if mrErr != nil {
			fmt.Println("❌ Error marshalling metadata:", mrErr)
			continue
		}

		metaJson := string(metaJsonBytes)

		objects = append(objects, worker.Object{
			Hash:        fileContainer.Hash,
			RelPath:     file.Remote,
			Name:        path.Base(file.LocalFile),
			AddedBy:     &by,
			ProjectName: projName,
			Metadata:    &metaJson,
		})
	}

	return objects
}
