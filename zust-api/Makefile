postgres:
	sudo docker run --name postgres17 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=123456 -d postgres:17.5-alpine3.22

createdb: 
	sudo docker exec -it postgres17 createdb --username=root --owner=root zust

dropdb: 
	sudo docker exec -it postgres17 dropdb zust

initschema:
	sudo docker exec -i postgres17 psql -U root -d zust < ./db/schema/schema.sql

destroyschema:
	sudo docker exec -i postgres17 psql -U root -d zust < ./db/schema/destroy.sql

psql: 
	sudo docker exec -it postgres17 psql -U root -d zust

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

run:
	go run cmd/main.go

.PHONY: postgres createdb dropdb initschema destroyschema psql sqlc test run 