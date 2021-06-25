package mongo

import (
	"context"

	"gopkg.in/mgo.v2/bson"
)

const collection = "outline_users"

type OutlineUser struct {
	User string `json:"user" bson:"user"` // fruitice user

	ID     string `json:"id" bson:"id"`                   // outline user id
	Cipher string `json:"cipher" bson:"cipher"`           // outline cipher (e.g.: "chacha20-ietf-poly1305")
	Secret string `json:"secret,omitempty" bson:"secret"` // outline secret
}

// GetUser return an usable proxy user
func (r *Roach) GetUser(ctx context.Context, id string) (*OutlineUser, error) {
	res := OutlineUser{}
	err := r.Db.Collection(collection).FindOne(ctx, bson.M{"id": id}).Decode(&res)
	if err != nil {
		return nil, err
	}
	return &res, err
}

// DeleteUser deletes a proxy user
func (r *Roach) DeleteUser(ctx context.Context, id string) error {
	_, err := r.Db.Collection(collection).DeleteOne(ctx, bson.M{"id": id})
	return err
}

// ListUsers list all proxy users for a fruitice user
func (r *Roach) ListUsers(ctx context.Context, user string) ([]*OutlineUser, error) {
	cur, err := r.Db.Collection(collection).Find(ctx, bson.M{"user": user})
	if err != nil {
		return nil, err
	}
	users := []*OutlineUser{}
	for cur.Next(ctx) {
		user := OutlineUser{}
		cur.Decode(&user)
		users = append(users, &user)
	}
	return users, nil
}

// ListAllUsers list all outline users
func (r *Roach) ListAllUsers(ctx context.Context) ([]*OutlineUser, error) {
	cur, err := r.Db.Collection(collection).Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	users := []*OutlineUser{}
	for cur.Next(ctx) {
		user := OutlineUser{}
		cur.Decode(&user)
		users = append(users, &user)
	}
	return users, nil
}

// AddUser for a fruitice user
func (r *Roach) AddUser(ctx context.Context, user *OutlineUser) (string, error) {
	finalUser := &OutlineUser{
		User:   user.User,
		ID:     user.User + "_" + user.ID,
		Secret: user.Secret,
	}

	_, err := r.Db.Collection(collection).InsertOne(ctx, finalUser)
	return finalUser.ID, err
}
