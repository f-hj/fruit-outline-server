package mongo

import (
	"context"
	"fmt"
	"log"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Roach is a pointer to mongo connection
type Roach struct {
	Client *mongo.Client
	Db     *mongo.Database
	cfg    Config
}

// Config holds the configuration used for instantiating a new Roach.
type Config struct {
	// Address that locates our postgres instance
	Host string
	// Port to connect to
	Port string
	// User that has access to the database
	User string
	// Password so that the user can login
	Password string
	// Database to connect to (must have been created priorly)
	Database string
}

// New create an instance of Roach
func New(cfg Config) (roach Roach, err error) {
	if cfg.Host == "" || cfg.Port == "" || cfg.Database == "" {
		err = errors.Errorf(
			"All mongo fields must be set",
		)
		return
	}

	roach.cfg = cfg

	url := fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	log.Println("Connect mongo to: " + url)

	client, err := mongo.NewClient(options.Client().ApplyURI(url))
	if err != nil {
		log.Println(err)
		err = errors.Wrapf(err,
			"Couldn't open connection to mongo",
		)
		return
	}
	err = client.Connect(context.Background())
	if err != nil {
		log.Println(err)
		err = errors.Wrapf(err,
			"Couldn't connect to mongo",
		)
		return
	}

	roach.Client = client
	roach.Db = client.Database(cfg.Database)
	return
}
