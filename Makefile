dev:
	docker build --tag 'sloggo:local' .
	docker run -p 8080:8080 -p 5514:5514/udp -p 6514:6514 -e SLOGGO_DEBUG=true sloggo:local

test:
	cd backend && go test -cover -v ./...
