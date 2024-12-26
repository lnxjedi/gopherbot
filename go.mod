module github.com/lnxjedi/gopherbot/v2

go 1.23.3

replace github.com/lnxjedi/gopherbot/robot => ./robot

// replace github.com/go-git/go-git/v5 => ./replacements/go-git

require (
	github.com/aws/aws-sdk-go v1.44.152
	github.com/duosecurity/duo_api_golang v0.0.0-20221117185402-091daa09e19d
	github.com/emersion/go-textwrapper v0.0.0-20200911093747-65d896831594
	github.com/joho/godotenv v1.4.0
	github.com/jordan-wright/email v0.0.0-20200121133829-a0b5c5b58bb6
	github.com/lnxjedi/gopherbot/robot v0.0.0-20221211204919-1966e9d9cfec
	github.com/lnxjedi/readline v0.0.0-20200213173224-cdfc6ee4b159
	github.com/pquerna/otp v1.3.0
	github.com/robfig/cron v1.2.0
	github.com/slack-go/slack v0.12.2
	golang.org/x/sys v0.27.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	dario.cat/mergo v1.0.0 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/ProtonMail/go-crypto v1.0.0 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/chzyer/test v1.0.0 // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/dop251/goja v0.0.0-20241024094426-79f3a7efcdbd // indirect
	github.com/dop251/goja_nodejs v0.0.0-20240728170619-29b559befffc // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.5.0 // indirect
	github.com/go-git/go-git/v5 v5.12.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/pprof v0.0.0-20240727154555-813a5fbdbec8 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/skeema/knownhosts v1.2.2 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/traefik/yaegi v0.16.1 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	golang.org/x/crypto v0.29.0 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.31.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

// *** When using a fork - in this case, iaburton's safer-socketmode
replace github.com/slack-go/slack => github.com/lnxjedi/slack v0.1.2

// *** For local Slack lib dev - comment out any other replace for slack above
// replace github.com/slack-go/slack => ../slack
