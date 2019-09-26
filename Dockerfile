FROM golang:alpine as build
ENV APP_DIR=$GOPATH/src/github.com/xetys/node-label-controller
WORKDIR $APP_DIR
COPY . $APP_DIR
ENV GO11MODULE=ON
RUN go get
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /node-label-controller


FROM scratch as prod
COPY --from=build /node-label-controller /

ENTRYPOINT ["/node-label-controller"]
