# creates a .zip file with the file structure to run UVa-ITS-gopherbot

[String]$clean = git status | select-string -pattern "nothing to commit, working tree clean"
if ( $clean -eq "" ) {
  write-output "Your working tree isn't clean, aborting build"
  exit 1
}

$version = "" + (Select-String -Path .\bot\bot.go -pattern "var Version = ")

$Start = $version.IndexOf('"')
$end = $version.LastIndexOf('"')

$version = $version.Substring($start + 1, $end - $start - 1)
[String]$commit = git log -1 | select-string -pattern "commit "
$commit = $commit.split(" ")[1]
$gofile = @"
package bot

func init(){
	commit="$commit"
}
"@

Write-Output "Building for Windows 64bit"
go build

$fileName = "gopherbot-" + $version + "-windows-amd64.zip"

$list = "gopherbot.exe", "LICENSE", "README.md", ".\brain", ".\conf", ".\doc", ".\example.gopherbot", ".\lib", ".\licenses", ".\misc", ".\plugins"

Write-Output "Creating archive $filename"
Compress-Archive -path $list -DestinationPath $fileName -force
