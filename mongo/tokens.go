package mongo

import (
	"context"

	"gopkg.in/mgo.v2/bson"
)

// Token is an access token stored in mongo
type Token struct {
	Token string   `json:"token" bson:"token"`
	User  string   `json:"user" bson:"user"`
	Scope []string `json:"scope" bson:"scope"`
}

// GetToken return an usable Token object, empty
func (roach *Roach) GetToken(ctx context.Context, token string) (*Token, error) {
	res := Token{}
	err := roach.Db.Collection("tokens").FindOne(ctx, bson.M{"token": token}).Decode(&res)
	if err != nil {
		return nil, err
	}
	return &res, err
}
