module gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/aws/aws-sdk-go v1.29.19
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/mitchellh/gox v1.0.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/afero v1.2.2
	github.com/stretchr/testify v1.4.0
	github.com/urfave/cli v1.22.2
	gitlab.com/ayufan/golang-cli-helpers v0.0.0-20171103152739-a7cf72d604cd
	gitlab.com/gitlab-org/gitlab-runner v12.5.0+incompatible
	golang.org/x/crypto v0.0.0-20200214034016-1d94cc7ab1c6
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b // indirect
	golang.org/x/sys v0.0.0-20200217220822-9197077df867 // indirect
)
