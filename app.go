package main

// Document of redigo
// https://godoc.org/github.com/gomodule/redigo/redis

// Commands of redis
// https://redis.io/commands

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	redis "github.com/gomodule/redigo/redis"
	"github.com/labstack/echo"
)

var (
	redisPool *redis.Pool
)

type Renderer struct {
	templates *template.Template
}

func (r *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return r.templates.ExecuteTemplate(w, name, data)
}

func redisIncr(key string) error {
	redisConn := redisPool.Get()
	defer redisConn.Close()

	// 基本的にredisのコマンドをそのまま呼び出す
	_, err := redisConn.Do("INCR", key)
	return err
}

func redisSet(key string, val string) error {
	redisConn := redisPool.Get()
	defer redisConn.Close()

	_, err := redisConn.Do("SET", key, val)
	return err
}

func redisSetInt(key string, val int64) error {
	redisConn := redisPool.Get()
	defer redisConn.Close()

	// 引数はstringにしておく必要がある
	_, err := redisConn.Do("SET", key, strconv.FormatInt(val, 10))
	return err
}

func redisGet(key string) string {
	redisConn := redisPool.Get()
	defer redisConn.Close()

	// 値を取り出す時には redis.XXX を使う
	str, err := redis.String(redisConn.Do("GET", key))
	if err == nil {
		// 正常
	} else if err == redis.ErrNil {
		// nilが返ってきた
		return ""
	} else {
		log.Printf("something wrong: %v\n", str)
		// panic()
		return ""
	}
	return str
}

func redisGetInt(key string) int64 {
	redisConn := redisPool.Get()
	defer redisConn.Close()

	res, err := redis.Int64(redisConn.Do("GET", key)) // keyがない時には0を返す
	if err == nil {
		// 正常
	} else if err == redis.ErrNil {
		// nilが返ってきた
	} else {
		log.Printf("something wrong: %v\n", res)
		// panic()
		return 0
	}
	return res
}

func getInitialize(c echo.Context) error {
	redisConn := redisPool.Get()
	defer redisConn.Close()

	redisConn.Do("FLUSHALL")
	return c.String(204, "")
}

func getObject(c echo.Context) error {
	key := c.QueryParam("key")
	str := redisGet(key)
	return c.String(http.StatusOK, str)
}

func postObject(c echo.Context) error {
	key := c.FormValue("key")
	val := c.FormValue("val")
	if err := redisSet(key, val); err != nil {
		return c.String(http.StatusInternalServerError, "InternalServerError")
	}
	return c.String(http.StatusOK, "Success")
}

func postIncrObject(c echo.Context) error {
	key := c.FormValue("key")
	if err := redisIncr(key); err != nil {
		return c.String(http.StatusInternalServerError, "InternalServerError")
	}
	return c.String(http.StatusOK, "Success")
}

func getIndex(c echo.Context) error {
	return c.Render(http.StatusOK, "index", nil)
}

func main() {
	e := echo.New()

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	// 接続を確立する
	redisPool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%v:6379", redisHost))
			if err != nil {
				log.Fatalln("Can not connect to redis.")
				return nil, err
			}
			return c, err
		},
	}

	e.Renderer = &Renderer{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}

	e.GET("/initialize", getInitialize)
	e.GET("/", getIndex)
	e.GET("/get", getObject)
	e.POST("/set", postObject)
	e.POST("/increment", postIncrObject)

	e.Start(":5000")
}
