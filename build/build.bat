cd ..\
set GOPATH=%cd%
go build src\main\installer.go
go build src\main\backup.go
rd /s /q _output
md _output
robocopy .\ _output installer.exe
robocopy .\ _output backup.exe
robocopy conf _output\conf
robocopy scripts _output\scripts

del installer.exe
del backup.exe