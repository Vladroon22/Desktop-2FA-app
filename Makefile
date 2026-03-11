.PHONY: app

app:
	go build -ldflags="-X main.appVersion=1.0.3" -o ./2fa cmd/main.go 

