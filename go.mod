module github.com/lnxjedi/gopherbot/v2

go 1.19

require (
	github.com/Jeffail/gabs v1.4.0
	github.com/aws/aws-sdk-go v1.44.152
	github.com/duosecurity/duo_api_golang v0.0.0-20221117185402-091daa09e19d
	github.com/emersion/go-textwrapper v0.0.0-20200911093747-65d896831594
	github.com/ghodss/yaml v1.0.0
	github.com/gopackage/ddp v0.0.4
	github.com/joho/godotenv v1.4.0
	github.com/jordan-wright/email v0.0.0-20200121133829-a0b5c5b58bb6
	github.com/lnxjedi/gopherbot/robot v0.0.0-20221205155336-9f52206a3a2a
	github.com/lnxjedi/readline v0.0.0-20200213173224-cdfc6ee4b159
	github.com/pquerna/otp v1.3.0
	github.com/robfig/cron v1.2.0
	github.com/slack-go/slack v0.11.4
	github.com/stretchr/testify v1.8.1
	golang.org/x/sys v0.3.0
)

require (
	github.com/apex/log v1.9.0 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/chzyer/test v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.1.0 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// *** When using a fork
// replace github.com/slack-go/slack => github.com/lnxjedi/slack v0.1.1

// *** For local Slack lib dev - comment out any other replace for slack above
// replace github.com/slack-go/slack => ../slack
