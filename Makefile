dev:
	docker build --tag 'sloggo:local' .
	docker run --network="host" -v sloggo_node_modules:/app/frontend/node_modules sloggo:local

test:
	go test -cover -v ./...
