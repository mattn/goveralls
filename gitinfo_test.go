package main

import (
	"os"
	"testing"
)

func TestLoadBranchFromEnv(t *testing.T) {
	var tests = []struct {
		testCase       string
		envs           map[string]string
		expectedBranch string
	}{
		{
			"all vars defined",
			map[string]string{
				"GIT_BRANCH":           "master",
				"CIRCLE_BRANCH":        "circle-master",
				"TRAVIS_BRANCH":        "travis-master",
				"CI_BRANCH":            "ci-master",
				"APPVEYOR_REPO_BRANCH": "appveyor-master",
			},
			"master",
		},
		{
			"all except GIT_BRANCH",
			map[string]string{
				"CIRCLE_BRANCH":        "circle-master",
				"TRAVIS_BRANCH":        "travis-master",
				"CI_BRANCH":            "ci-master",
				"APPVEYOR_REPO_BRANCH": "appveyor-master",
			},
			"circle-master",
		},
		{
			"all except GIT_BRANCH and CIRCLE_BRANCH",
			map[string]string{
				"TRAVIS_BRANCH":        "travis-master",
				"CI_BRANCH":            "ci-master",
				"APPVEYOR_REPO_BRANCH": "appveyor-master",
			},
			"travis-master",
		},
		{
			"only CI_BRANCH defined",
			map[string]string{
				"CI_BRANCH": "ci-master",
			},
			"ci-master",
		},
		{
			"only APPVEYOR_REPO_BRANCH defined",
			map[string]string{
				"APPVEYOR_REPO_BRANCH": "appveyor-master",
			},
			"appveyor-master",
		},
		{
			"no branch var defined",
			map[string]string{},
			"",
		},
	}
	for _, test := range tests {
		resetBranchEnvs(test.envs)
		envBranch := loadBranchFromEnv()
		if envBranch != test.expectedBranch {
			t.Errorf("%s: wrong branch returned. Expected %q, but got %q", test.testCase, test.expectedBranch, envBranch)
		}
	}
}

func resetBranchEnvs(values map[string]string) {
	for _, envVar := range []string{"CI_BRANCH", "CIRCLE_BRANCH", "GIT_BRANCH", "TRAVIS_BRANCH", "APPVEYOR_REPO_BRANCH"} {
		os.Unsetenv(envVar)
	}
	for k, v := range values {
		os.Setenv(k, v)
	}
}
