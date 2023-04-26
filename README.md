goveralls
=========

[Go](http://golang.org) integration for [Coveralls.io](http://coveralls.io)
continuous code coverage tracking system.

# Installation

`goveralls` requires a working Go installation (Go-1.13 or higher).

```bash
$ go install github.com/mattn/goveralls@latest
```

# Usage

For non-public repositories you will need a Coveralls.io token; it is found at the bottom of your
repository's page when you are logged in to Coveralls.io.  Each repository has its own token.

```bash
$ cd $GOPATH/src/github.com/yourusername/yourpackage
$ goveralls -repotoken your_repos_coveralls_token
```

It is advised to set the environment variable `COVERALLS_TOKEN` to your token so you do
not have to specify it at each invocation through `-repotoken`, which is less secure.

You can also run this reporter for multiple passes with the flag `-parallel` or
by setting the environment variable `COVERALLS_PARALLEL=true` (see [coveralls
docs](https://docs.coveralls.io/parallel-build-webhook) for more details).

`goveralls` will run the entire test suite when no coverage profile files are provided with `-coverprofile`;
you can use `-flags` to pass extra flags to the Go tests run by goveralls.

## Environment variables

Some metadata used when submitting a Coveralls.io job can be specified via any of the mentioned environment variables:

* job ID: `COVERALLS_SERVICE_JOB_ID`, `TRAVIS_JOB_ID`, `CIRCLE_BUILD_NUM`, `APPVEYOR_JOB_ID`, `SEMAPHORE_BUILD_NUMBER`, `BUILD_NUMBER`, `BUILDKITE_BUILD_ID`, `DRONE_BUILD_NUMBER`, `BUILDKITE_BUILD_NUMBER`, `CI_BUILD_ID`, `GITHUB_RUN_ID`
* branch name: `GIT_BRANCH`, `GITHUB_HEAD_REF`, `GITHUB_REF`,	`CIRCLE_BRANCH`, `TRAVIS_BRANCH`, `CI_BRANCH`, `APPVEYOR_REPO_BRANCH`, `WERCKER_GIT_BRANCH`, `DRONE_BRANCH`, `BUILDKITE_BRANCH`, `BRANCH_NAME`
* pull request number: `CIRCLE_PR_NUMBER`, `TRAVIS_PULL_REQUEST`, `APPVEYOR_PULL_REQUEST_NUMBER`, `PULL_REQUEST_NUMBER`, `BUILDKITE_PULL_REQUEST`, `DRONE_PULL_REQUEST`, `CI_PR_NUMBER`
* pull request number (extracted with regex): `CI_PULL_REQUEST`, `GITHUB_EVENT_NAME`
* repository token: `COVERALLS_TOKEN`
* repository token file: `COVERALLS_TOKEN_FILE`
* parallel flag: `COVERALLS_PARALLEL`
* Coveralls.io job flag name: `COVERALLS_FLAG_NAME`
* repository name: `GITHUB_REPOSITORY`

### Special cases

* when `TRAVIS_JOB_ID` is specified then `-service` will automatically be set to `travis-ci`.
* when `GITHUB_EVENT_NAME` is set to `pull_request` then the file at `GITHUB_EVENT_PATH` will be attempted read and used to parse the pull request number and the Git HEAD reference.

# Continuous Integration

It is possible to use goveralls with any Continuous Integration platform; integration with the most common ones is explained below with some examples.

## GitHub Actions

[shogo82148/actions-goveralls](https://github.com/marketplace/actions/actions-goveralls) is available on GitHub Marketplace.
It provides the shorthand of the GitHub Actions YAML configure.

```yaml
name: Quality
on: [push, pull_request]
jobs:
  test:
    name: Test with Coverage
    runs-on: ubuntu-latest
    steps:
    - name: Set up Go
      uses: actions/setup-go@v2
      with:
        go-version: '1.16'
    - name: Check out code
      uses: actions/checkout@v2
    - name: Install dependencies
      run: |
        go mod download
    - name: Run Unit tests
      run: |
        go test -race -covermode atomic -coverprofile=covprofile ./...
    - name: Install goveralls
      run: go install github.com/mattn/goveralls@latest
    - name: Send coverage
      env:
        COVERALLS_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      run: goveralls -coverprofile=covprofile -service=github
    # or use shogo82148/actions-goveralls
    # - name: Send coverage
    #   uses: shogo82148/actions-goveralls@v1
    #   with:
    #     path-to-profile: covprofile
```

## Travis CI

### GitHub Integration

Enable Travis-CI on your GitHub repository settings.

For a **public** GitHub repository put bellow's `.travis.yml`.

```yml
language: go
go:
  - tip
before_install:
  - go install github.com/mattn/goveralls@latest
script:
  - $GOPATH/bin/goveralls -service=travis-ci
```

For a **public** GitHub repository, it is not necessary to define your repository key (`COVERALLS_TOKEN`).

For a **private** GitHub repository put bellow's `.travis.yml`. If you use **travis pro**, you need to specify `-service=travis-pro` instead of `-service=travis-ci`.

```yml
language: go
go:
  - tip
before_install:
  - go install github.com/mattn/goveralls@latest
script:
  - $GOPATH/bin/goveralls -service=travis-pro
```

Store your Coveralls API token in `Environment variables`.

```
COVERALLS_TOKEN = your_token_goes_here
```

or you can store token using [travis encryption keys](https://docs.travis-ci.com/user/encryption-keys/). Note that this is the token provided in the page for that specific repository on Coveralls. This is *not* one that was created from the "Personal API Tokens" area under your Coveralls account settings.

```
$ gem install travis
$ travis encrypt COVERALLS_TOKEN=your_token_goes_here --add env.global
```

travis will add `env` block as following example:

```yml
env:
  global:
    secure: xxxxxxxxxxxxx
```

### For others:

```
$ go install github.com/mattn/goveralls@latest
$ go test -covermode=count -coverprofile=profile.cov
$ goveralls -coverprofile=profile.cov -service=travis-ci
```

## Drone.io

Store your Coveralls API token in `Environment Variables`:

```
COVERALLS_TOKEN=your_token_goes_here
```

Replace the `go test` line in your `Commands` with these lines:

```
$ go install github.com/mattn/goveralls@latest
$ goveralls -service drone.io
```

`goveralls` automatically use the environment variable `COVERALLS_TOKEN` as the
default value for `-repotoken`.

You can use the `-v` flag to see verbose output from the test suite:

```
$ goveralls -v -service drone.io
```

## CircleCI

Store your Coveralls API token as an [Environment Variable](https://circleci.com/docs/environment-variables).

In your `circle.yml` add the following commands under the `test` section.

```yml
test:
  pre:
    - go install github.com/mattn/goveralls@latest
  override:
    - go test -v -cover -race -coverprofile=/home/ubuntu/coverage.out
  post:
    - /home/ubuntu/.go_workspace/bin/goveralls -coverprofile=/home/ubuntu/coverage.out -service=circle-ci -repotoken=$COVERALLS_TOKEN
```

For more information, See https://docs.coveralls.io/go

## Semaphore

Store your Coveralls API token in `Environment Variables`:

```
COVERALLS_TOKEN=your_token_goes_here
```

More instructions on how to do this can be found in the [Semaphore documentation](https://semaphoreci.com/docs/exporting-environment-variables.html).

Replace the `go test` line in your `Commands` with these lines:

```
$ go install github.com/mattn/goveralls@latest
$ goveralls -service semaphore
```

`goveralls` automatically use the environment variable `COVERALLS_TOKEN` as the
default value for `-repotoken`.

You can use the `-v` flag to see verbose output from the test suite:

```
$ goveralls -v -service semaphore
```

## Jenkins CI

Add your Coveralls API token as a credential in Jenkins (see [Jenkins documentation](https://www.jenkins.io/doc/book/using/using-credentials/#configuring-credentials)).

Then declare it as the environment variable `COVERALLS_TOKEN`:
```groovy
pipeline {
    agent any
    stages {
        stage('Test with coverage') {
            steps {
                sh 'go test ./... -coverprofile=coverage.txt -covermode=atomic'
            }
        }
        stage('Upload to coveralls.io') {
            environment {
                COVERALLS_TOKEN     = credentials('coveralls-token')
            }
            steps {
                sh 'goveralls -coverprofile=coverage.txt'
            }
        }
    }
}
```

See also [related Jenkins documentation](https://www.jenkins.io/doc/book/pipeline/jenkinsfile/#for-secret-text-usernames-and-passwords-and-secret-files).

It is also possible to let goveralls run the code coverage on its own without providing a coverage profile file.

## TeamCity

Store your Coveralls API token in `Environment Variables`:

```
COVERALLS_TOKEN=your_token_goes_here
```

Setup build steps:

```
$ go install github.com/mattn/goveralls@latest
$ export PULL_REQUEST_NUMBER=%teamcity.build.branch%
$ goveralls -service teamcity -jobid %teamcity.build.id% -jobnumber %build.number%
```

`goveralls` will automatically use the environment variable `COVERALLS_TOKEN` as the
default value for `-repotoken`.

You can use the `-v` flag to see verbose output.


## Gitlab CI

Store your Coveralls API token as an [Environment Variable](https://docs.gitlab.com/ee/ci/variables/#create-a-custom-variable-in-the-ui) named `COVERALLS_TOKEN`.

```yml
test:
  timeout: 30m
  stage: test
  artifacts:
    paths:
      - coverage.txt
  dependencies:
    - build:env
  when: always
  script:
    - go test -covermode atomic -coverprofile=coverage.txt ./...
    - go install github.com/mattn/goveralls@latest
    - goveralls -service=gitlab -coverprofile=coverage.txt
```

## Coveralls Enterprise

If you are using Coveralls Enterprise and have a self-signed certificate, you need to skip certificate verification:

```shell
$ goveralls -insecure
```

# Authors

* Yasuhiro Matsumoto (a.k.a. mattn)
* haya14busa

# License

under the MIT License: http://mattn.mit-license.org/2016
