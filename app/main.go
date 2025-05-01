package main

import (
	"fmt"

	"github.com/Hack-Nocturne/cfs3/utils"
	"github.com/Hack-Nocturne/cfs3/vars"
	"github.com/Hack-Nocturne/cfs3/worker"
)

func main() {
	config, cfgErr := worker.LoadNProcessConfig("cfs3.config.json")
	if cfgErr != nil {
		fmt.Println("❌ Failure loading config:", cfgErr)
		return
	}

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

	StartCFUpload(config.ProjectName)
	// ToDo: Implement bulk add to database
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
