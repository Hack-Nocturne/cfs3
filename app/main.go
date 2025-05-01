package main

import (
	"fmt"

	"github.com/Hack-Nocturne/cfs3/utils"
	"github.com/Hack-Nocturne/cfs3/vars"
)

func main() {
	// ToDo: Implement env based configuration
}

func StartCFUpload(accountId, apiToken, directory, projectName string) error {
	vars.CF_ACCOUNT_ID = accountId
	vars.CF_API_TOKEN = apiToken

	uploadArgs := utils.PagesDeployOptions{
		Directory:   directory,
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
