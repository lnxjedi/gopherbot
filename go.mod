module github.com/lnxjedi/gopherbot/v2

go 1.24.0

replace github.com/lnxjedi/gopherbot/robot => ./robot

replace github.com/lnxjedi/gopherbot/test => ./test

replace github.com/chzyer/readline => ./golib/readline

// replace github.com/go-git/go-git/v5 => ./replacements/go-git

require (
	github.com/aws/aws-sdk-go-v2 v1.41.1
	github.com/aws/aws-sdk-go-v2/config v1.32.7
	github.com/aws/aws-sdk-go-v2/credentials v1.19.7
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.20.31
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.54.0
	github.com/aws/smithy-go v1.24.0
	github.com/dop251/goja v0.0.0-20260106131823-651366fbe6e3
	github.com/dop251/goja_nodejs v0.0.0-20251015164255-5e94316bedaf
	github.com/duosecurity/duo_api_golang v0.0.0-20250430191550-ac36954387e7
	github.com/emersion/go-textwrapper v0.0.0-20200911093747-65d896831594
	github.com/chzyer/readline v1.5.1
	github.com/go-git/go-git/v5 v5.16.4
	github.com/joho/godotenv v1.5.1
	github.com/jordan-wright/email v4.0.1-0.20210109023952-943e75fe5223+incompatible
	github.com/lnxjedi/gopherbot/robot v0.0.0-20221211204919-1966e9d9cfec
	github.com/lnxjedi/gopherbot/test v0.0.0-00010101000000-000000000000
	github.com/pquerna/otp v1.5.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/slack-go/slack v0.17.3
	github.com/traefik/yaegi v0.16.1
	github.com/yuin/gopher-lua v1.1.1
	golang.org/x/crypto v0.47.0
	golang.org/x/sys v0.40.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	dario.cat/mergo v1.0.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.1.6 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.32.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.6 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/chzyer/test v1.0.0 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/cyphar/filepath-securejoin v0.4.1 // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.2 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/pprof v0.0.0-20240727154555-813a5fbdbec8 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/pjbgf/sha1cd v0.3.2 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/skeema/knownhosts v1.3.1 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)
