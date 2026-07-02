.PHONY: run build

# .ONESHELL ensures all lines run in the same shell,
# so the & background jobs are visible to wait.
.ONESHELL:
run:
	go run ./cmd/users-service & \
	go run ./cmd/orders-service & \
	go run ./cmd/gateway & \
	wait

build:
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/users-service ./cmd/users-service
	go build -o bin/orders-service ./cmd/orders-service

run-built: build
	./bin/users-service & \
	./bin/orders-service & \
	./bin/gateway & \
	wait
