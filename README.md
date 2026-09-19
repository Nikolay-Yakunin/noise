# noise

This is perlin noise!!! !!! !!! [source](https://habr.com/ru/articles/142592/)

## Usage

### cli

```sh
go run cmd/cli/main.go [width] [height] [scale] [persistence] [octaves]
```

Example
```sh
go run cmd/cli/main.go 2560 1440 16 0.5 5
```

### server

```sh
go run cmd/server/main.go
```

Go to [web interface](http://localhost:9090/)
Or use api (see server main.go)

## Result

![image](./noise.png)
