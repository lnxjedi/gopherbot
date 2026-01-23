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

// *** When using a fork - in this case, iaburton's safer-socketmode
replace github.com/slack-go/slack => github.com/lnxjedi/slack v0.1.2

// *** For local Slack lib dev - comment out any other replace for slack above
// replace github.com/slack-go/slack => ../slack
