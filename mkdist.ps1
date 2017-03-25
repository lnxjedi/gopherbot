# creates a .zip file with the file structure to run UVa-ITS-gopherbot

$version = "" + (Select-String -Path .\bot\bot.go -pattern "var Version = ") 

$Start = $version.IndexOf('"')
$end = $version.LastIndexOf('"')

$version = $version.Substring($start + 1, $end - $start - 1)

Write-Output "Building for Windows 64bit"
go build

$fileName = "gopherbot-" + $version + "-windows-amd64.zip"

$list = "gopherbot.exe", "LICENSE", "README.md", ".\brain", ".\conf", ".\doc", ".\example.gopherbot", ".\lib", ".\licenses", ".\misc", ".\plugins" 

Write-Output "Creating archive $filename"
Compress-Archive -path $list -DestinationPath $fileName -force