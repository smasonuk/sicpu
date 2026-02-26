export CGO_CFLAGS="-Wno-deprecated-declarations"

cd ./cmd/desktop
go run main.go