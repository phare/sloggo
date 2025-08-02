dev:
	docker build --tag 'sloggo:local' .
	docker run -p 8080:8080 -p 514:514 -p 6514:6514 sloggo:local

test:
	cd backend && go test -cover -v ./...
