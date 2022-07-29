module github.com/lnxjedi/gopherbot/v2

go 1.16

require (
	github.com/Jeffail/gabs v1.4.0
	github.com/aws/aws-sdk-go v1.40.34
	github.com/chzyer/logex v1.1.10 // indirect
	github.com/chzyer/test v0.0.0-20180213035817-a1ea475d72b1 // indirect
	github.com/duosecurity/duo_api_golang v0.0.0-20200206192355-a9725220d6ca
	github.com/emersion/go-textwrapper v0.0.0-20160606182133-d0e65e56babe
	github.com/ghodss/yaml v1.0.0
	github.com/gopackage/ddp v0.0.0-20170117053602-652027933df4
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/jordan-wright/email v0.0.0-20200121133829-a0b5c5b58bb6
	github.com/lnxjedi/readline v0.0.0-20200213173224-cdfc6ee4b159
	github.com/lnxjedi/robot v0.1.8
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/onsi/gomega v1.9.0 // indirect
	github.com/pquerna/otp v1.3.0
	github.com/robfig/cron v1.2.0
	github.com/slack-go/slack v0.11.2
	github.com/stretchr/testify v1.5.1
	golang.org/x/sys v0.0.0-20210423082822-04245dca01da
)

replace github.com/slack-go/slack => github.com/lnxjedi/slack v0.1.1

// *** For local Slack lib dev - comment out any other replace for slack above
// replace github.com/slack-go/slack => ../slack
