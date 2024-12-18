package authzed

import "code.gitea.io/gitea/models/db"

type MemberToSet struct {
	MemberType     string `xorm:"NOT NULL"`
	MemberID       string `xorm:"NOT NULL"`
	MemberRelation string
	SetType        string `xorm:"NOT NULL"`
	SetID          string `xorm:"NOT NULL"`
	SetRelation    string `xorm:"NOT NULL"`
}

type SetToSet struct {
	ChildType      string `xorm:"NOT NULL"`
	ChildID        string `xorm:"NOT NULL"`
	ChildRelation  string `xorm:"NOT NULL"`
	ParentType     string `xorm:"NOT NULL"`
	ParentID       string `xorm:"NOT NULL"`
	ParentRelation string `xorm:"NOT NULL"`
}

func init() {
	db.RegisterModel(new(MemberToSet))
	db.RegisterModel(new(SetToSet))
}
