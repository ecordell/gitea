// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_24 // nolint

import (
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

type MemberToSet struct {
	MemberType     string `xorm:"NOT NULL"`
	MemberID       string `xorm:"NOT NULL"`
	MemberRelation string
	SetType        string `xorm:"NOT NULL"`
	SetID          string `xorm:"NOT NULL"`
	SetRelation    string `xorm:"NOT NULL"`
}

// TableIndices implements xorm's TableIndices interface
func (*MemberToSet) TableIndices() []*schemas.Index {
	m2sMemberIndex := schemas.NewIndex("m2s_member", schemas.IndexType)
	m2sMemberIndex.AddColumn("member_type", "member_id", "member_relation")

	m2sSetIndex := schemas.NewIndex("m2s_set", schemas.IndexType)
	m2sSetIndex.AddColumn("set_type", "set_id", "set_relation")

	return []*schemas.Index{
		m2sMemberIndex,
		m2sSetIndex,
	}
}

type SetToSet struct {
	ChildType      string `xorm:"NOT NULL"`
	ChildID        string `xorm:"NOT NULL"`
	ChildRelation  string `xorm:"NOT NULL"`
	ParentType     string `xorm:"NOT NULL"`
	ParentID       string `xorm:"NOT NULL"`
	ParentRelation string `xorm:"NOT NULL"`
}

// TableIndices implements xorm's TableIndices interface
func (*SetToSet) TableIndices() []*schemas.Index {
	s2sChildIndex := schemas.NewIndex("s2s_child", schemas.IndexType)
	s2sChildIndex.AddColumn("child_type", "child_id", "child_relation")

	s2sParentIndex := schemas.NewIndex("s2s_parent", schemas.IndexType)
	s2sParentIndex.AddColumn("parent_type", "parent_id", "parent_relation")

	return []*schemas.Index{
		s2sChildIndex,
		s2sParentIndex,
	}
}

func AddPermissionSetsTable(x *xorm.Engine) error {
	if err := x.Sync(&SetToSet{}); err != nil {
		return err
	}
	return x.Sync(&MemberToSet{})
}
