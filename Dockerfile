FROM golang:1.19-alpine as BUILD

WORKDIR /app

ADD . /app
RUN go mod download

RUN apk add  protoc
RUN apk update && apk add  make protobuf-dev
RUN CGO_ENABLED=0 go build -o /collablite /app/cmd/server/


FROM alpine

COPY --from=build /collablite /bin

EXPOSE 50051

CMD [ "collablite" ]