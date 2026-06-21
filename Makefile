run:
	go run ./cmd/users-service &
	go run ./cmd/orders-service &
	go run ./cmd/gateway &
	wait
