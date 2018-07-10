# FMGo

A simple friend management app written on Go

This project is means for simple demonstration of writing a bare minimum Web API application using Go language powered by [Gin web framework](https://github.com/gin-gonic/gin) and [GORM](https://github.com/jinzhu/gorm) as backend DAL layer.

## Run the code

First of all, clone or download this repository into your local machine:

`$ git clone https://github.com/apdevlab/fmgo.git && cd fmgo`

There are two ways to run the code

### The easy way using docker

If you have docker and docker-compose installed, simply run:

`$ docker-compose up`

Wait until you see line similar to `Starting fmgo server version 1.0.0 at :8080`. The app is ready for you to use and listening on port 8080 by default.

### The hard way

You will need Go installed in your local machine

* Install dep dependency manager

  `$ go get -u github.com/golang/dep/cmd/dep`

* Run dep ensure to downloads all dependencies

  `$ dep ensure`

* Copy file default.yml into .env.yml and modify the config to suit your environment

  `$ cp default.yml .env.yml`

* Ensure your database server is running and application table of your choice (by default it is fmgo, you can change it in .env.yml file) is exist

* Run the app. For first run you may want to add `-migrate` switch to run auto db migration.

  `$ go run main.go -migrate`

## API endpoint

By default the app will listen on all interface at port 8080. Here is the list of endpoint curently available

* Ping endpoint `GET /ping`
* Connect friend endpoint `POST /friend/connect`
* List all friend endpoint `POST /friend/list`
* List common friends endpoint `POST /friend/common`
* Subscribe notification endpoint `POST /notification/subscribe`
* Block notification endpoint `POST /notification/block`
* Get subscriber list endpoint `POST /notification/list`
