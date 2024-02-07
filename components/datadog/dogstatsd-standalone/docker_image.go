package dogstatsdstandalone

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
)

const (
	defaultDogstatsdImageRepo = "gcr.io/datadoghq/dogstatsd"
	defaultDogstatsdImageTag  = "latest"
)

func dockerDogstatsdFullImagePath(e *config.CommonEnvironment, repositoryPath string) string {
	// return dogstatsd image path if defined
	if e.DogstatsdFullImagePath() != "" {
		return e.DogstatsdFullImagePath()
	}

	// if agent pipeline id and commit sha are defined, use the image from the pipeline pushed on agent QA registry
	if e.PipelineID() != "" && e.CommitSHA() != "" {
		return utils.BuildDockerImagePath("669783387624.dkr.ecr.us-east-1.amazonaws.com/dogstatsd", fmt.Sprintf("%s-%s", e.PipelineID(), e.CommitSHA()))
	}

	if repositoryPath == "" {
		repositoryPath = defaultDogstatsdImageRepo
	}

	return utils.BuildDockerImagePath(repositoryPath, defaultDogstatsdImageTag)
}
