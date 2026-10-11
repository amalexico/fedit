build1:
	go build -o fedit .

test1:
	go test ./...

clean1:
	rm -f fedit
