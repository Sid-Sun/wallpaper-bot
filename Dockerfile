FROM golang:latest

# Setup the proper workdir
WORKDIR /root/bot
COPY ./go.mod ./
COPY ./go.sum ./
# Install the required dependencies
RUN go mod download
# Copy indivisual files at the end to leverage caching
COPY ./LICENSE ./
COPY ./README.md ./
COPY ./main.go ./
COPY ./utils.go ./
RUN go build

#Executable command needs to be static
CMD ["/root/bot/wallpaper-bot"]
