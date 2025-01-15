module github.com/lnxjedi/gopherbot/v2

go 1.23.4

replace github.com/lnxjedi/gopherbot/robot => ./robot

replace github.com/lnxjedi/gopherbot/test => ./test

// replace github.com/go-git/go-git/v5 => ./replacements/go-git

require (
	github.com/aws/aws-sdk-go v1.44.152
	github.com/dop251/goja v0.0.0-20241024094426-79f3a7efcdbd
	github.com/dop251/goja_nodejs v0.0.0-20240728170619-29b559befffc
	github.com/duosecurity/duo_api_golang v0.0.0-20221117185402-091daa09e19d
	github.com/emersion/go-textwrapper v0.0.0-20200911093747-65d896831594
	github.com/go-git/go-git/v5 v5.12.0
	github.com/joho/godotenv v1.4.0
	github.com/jordan-wright/email v0.0.0-20200121133829-a0b5c5b58bb6
	github.com/lnxjedi/gopherbot/robot v0.0.0-20221211204919-1966e9d9cfec
	github.com/lnxjedi/readline v0.0.0-20200213173224-cdfc6ee4b159
	github.com/pquerna/otp v1.3.0
	github.com/robfig/cron v1.2.0
	github.com/slack-go/slack v0.12.2
	github.com/traefik/yaegi v0.16.1
	github.com/yuin/gopher-lua v1.1.1
	golang.org/x/crypto v0.31.0
	golang.org/x/sys v0.28.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloud.google.com/go v0.117.0 // indirect
	cloud.google.com/go/auth v0.13.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.6 // indirect
	cloud.google.com/go/compute/metadata v0.6.0 // indirect
	cloud.google.com/go/firestore v1.18.0 // indirect
	cloud.google.com/go/longrunning v0.6.2 // indirect
	dario.cat/mergo v1.0.0 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/ProtonMail/go-crypto v1.0.0 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/chzyer/test v1.0.0 // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.5.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/pprof v0.0.0-20240727154555-813a5fbdbec8 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/googleapis/gax-go/v2 v2.14.1 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/lnxjedi/gopherbot/test v0.0.0-00010101000000-000000000000 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/skeema/knownhosts v1.2.2 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.54.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.54.0 // indirect
	go.opentelemetry.io/otel v1.31.0 // indirect
	go.opentelemetry.io/otel/metric v1.31.0 // indirect
	go.opentelemetry.io/otel/trace v1.31.0 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/oauth2 v0.25.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/time v0.9.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	google.golang.org/api v0.216.0 // indirect
	google.golang.org/genproto v0.0.0-20241118233622-e639e219e697 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241209162323-e6fa225c2576 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250102185135-69823020774d // indirect
	google.golang.org/grpc v1.69.2 // indirect
	google.golang.org/protobuf v1.36.1 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)

// *** When using a fork - in this case, iaburton's safer-socketmode
replace github.com/slack-go/slack => github.com/lnxjedi/slack v0.1.2

// *** For local Slack lib dev - comment out any other replace for slack above
// replace github.com/slack-go/slack => ../slack
