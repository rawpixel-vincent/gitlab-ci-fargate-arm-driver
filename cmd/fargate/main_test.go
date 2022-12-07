package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/config"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/cli"
)

func TestLoadCliArgsEnvVars(t *testing.T) {
	originalTaskDefinition := "default-task-def:1"
	originalPlatformVersion := "LATEST"
	overrideTaskDefinition := "another-task-def:1"
	overridePlatformVersion := "1.4.0"

	tests := map[string]struct {
		taskDefinitionOverrideValue  string
		platformVersionOverrideValue string
		expectedTaskDefinition       string
		expectedPlatformVersion      string
	}{
		"Should keep original values if nothing received by command line or env variable": {
			taskDefinitionOverrideValue: "",
			expectedTaskDefinition:      originalTaskDefinition,
			expectedPlatformVersion:     originalPlatformVersion,
		},
		"Should override task definition if received by command line or env variable": {
			taskDefinitionOverrideValue: overrideTaskDefinition,
			expectedTaskDefinition:      overrideTaskDefinition,
			expectedPlatformVersion:     originalPlatformVersion,
		},
		"Should override platform version if received by command line or env variable": {
			platformVersionOverrideValue: overridePlatformVersion,
			expectedTaskDefinition:       originalTaskDefinition,
			expectedPlatformVersion:      overridePlatformVersion,
		},
		"Should override platform version and task definition if received by command line or env variable": {
			taskDefinitionOverrideValue:  overrideTaskDefinition,
			platformVersionOverrideValue: overridePlatformVersion,
			expectedTaskDefinition:       overrideTaskDefinition,
			expectedPlatformVersion:      overridePlatformVersion,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			oldTaskDefGlobalValue := global.TaskDefinition
			oldPlatformVersionGlobalValue := global.PlatformVersion
			global.TaskDefinition = tt.taskDefinitionOverrideValue
			global.PlatformVersion = tt.platformVersionOverrideValue

			defer func() {
				global.TaskDefinition = oldTaskDefGlobalValue
				global.PlatformVersion = oldPlatformVersionGlobalValue
			}()

			testContext := createCliContextForTests(originalTaskDefinition, originalPlatformVersion)
			assert.Equal(t, originalTaskDefinition, testContext.Config().Fargate.TaskDefinition)
			assert.Equal(t, originalPlatformVersion, testContext.Config().Fargate.PlatformVersion)

			err := loadCliArgsEnvVars(testContext)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedTaskDefinition, testContext.Config().Fargate.TaskDefinition)
			assert.Equal(t, tt.expectedPlatformVersion, testContext.Config().Fargate.PlatformVersion)
		})
	}
}

func createCliContextForTests(taskDefinition string, platformVersion string) *cli.Context {
	cliCtx := new(cli.Context)
	cliCtx.SetConfig(config.Global{
		Fargate: config.Fargate{
			TaskDefinition:  taskDefinition,
			PlatformVersion: platformVersion,
		},
	})

	return cliCtx
}
