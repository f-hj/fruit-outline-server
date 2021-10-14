package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/f-hj/fruit-outline-server/mongo"
	"github.com/f-hj/fruit-outline-server/outline"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	// GitHash is set at compilation if available
	GitHash = "0"

	// Version is set at compilation if tag
	Version = "0.0.0"

	// BuildTime is set at compilation
	BuildTime = "0000-00-00 UTC"

	// BuildHost is set at compilation
	BuildHost = "unknown"
)

func main() {
	fmt.Println("fruit-outline-server")
	fmt.Println("git hash: " + GitHash)
	fmt.Println("version: " + Version)
	fmt.Println("build time: " + BuildTime)
	fmt.Println("build host: " + BuildHost)
	fmt.Println()

	db, err := mongo.New(mongo.Config{
		Host:     os.Getenv("MONGO_HOST"),
		Port:     "27017",
		User:     os.Getenv("MONGO_USER"),
		Password: os.Getenv("MONGO_PASS"),
		Database: os.Getenv("MONGO_DB"),
	})
	if err != nil {
		panic(err)
	}
	log.Println("mongo ok")

	ss, err := outline.New()
	if err != nil {
		panic(err)
	}

	startOutlineServer(db, ss)
	log.Println("outline ok")

	startWebServer(db, ss)
}

func startWebServer(db mongo.Roach, ss *outline.SSServer) {
	e := echo.New()
	e.Use(middleware.CORS())

	e.GET("/", func(c echo.Context) error {
		return c.JSONPretty(http.StatusOK, map[string]string{
			"version":   Version,
			"gitHash":   GitHash,
			"buildTime": BuildTime,
			"buildHost": BuildHost,
		}, " ")
	})
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	v1 := e.Group("/v1")
	v1.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
		token, err := db.GetToken(c.Request().Context(), key)
		if err != nil {
			return false, echo.NewHTTPError(http.StatusBadRequest, "bad token")
		}
		for _, scope := range token.Scope {
			if scope == "outline" {
				c.Set("user", token.User)
				return true, nil
			}
		}
		return false, echo.NewHTTPError(http.StatusBadRequest, "bad scope")
	}))

	v1.GET("/users", func(c echo.Context) error {
		users, err := db.ListUsers(c.Request().Context(), c.Get("user").(string))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, users)
	})

	v1.PUT("/users", func(c echo.Context) error {
		user := &mongo.OutlineUser{}
		err := c.Bind(&user)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "only json")
		}

		user.User = c.Get("user").(string)

		id, err := db.AddUser(c.Request().Context(), user)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "cannot add user")
		}

		go reloadOutlineServer(db, ss)

		return c.JSON(http.StatusOK, struct {
			ID     string `json:"id"`
			Cipher string `json:"cipher"`
			Secret string `json:"secret"`
		}{
			ID:     id,
			Cipher: user.Cipher,
			Secret: user.Secret,
		})
	})

	v1.DELETE("/users/:id", func(c echo.Context) error {
		usr, err := db.GetUser(c.Request().Context(), c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "cannot find user")
		}

		if usr.User != c.Get("user").(string) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}

		err = db.DeleteUser(c.Request().Context(), usr.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "cannot delete user")
		}

		go reloadOutlineServer(db, ss)

		return c.JSON(http.StatusOK, struct {
			Success bool `json:"success"`
		}{
			Success: true,
		})
	})

	if err := e.Start(":80"); err != nil {
		panic(err)
	}
}

func reloadOutlineServer(db mongo.Roach, ss *outline.SSServer) {
	users, err := db.ListAllUsers(context.Background())
	if err != nil {
		panic(err)
	}

	ss.Reload(users)
}

// startOutlineServer will exit when successfully listening
func startOutlineServer(db mongo.Roach, ss *outline.SSServer) {
	users, err := db.ListAllUsers(context.Background())
	if err != nil {
		panic(err)
	}

	if err := ss.Start(users); err != nil {
		panic(err)
	}
}
