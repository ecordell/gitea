package authzed

import (
	goctx "context"
	"fmt"
	"os"
	"strconv"

	v0 "github.com/authzed/authzed-go/proto/authzed/api/materialize/v0"
	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/grpcutil"
	"google.golang.org/grpc"

	authzed_model "code.gitea.io/gitea/models/authzed"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
)

// SpiceDB Schema:
//
// definition user {}
//
// definition repo {
//    relation owner: user
//    relation collaborator: user
//
//    permission view = owner + collaborator
// }

const (
	SubjectUser          = "user"
	ResourceRepo         = "repo"
	RelationOwner        = "owner"
	RelationCollaborator = "collaborator"
	PermissionView       = "view"
)

var (
	spiceDBEndpoint     = os.Getenv("SPICEDB_ENDPOINT")
	materializeEndpoint = os.Getenv("MATERIALIZE_ENDPOINT")
	authzedToken        = os.Getenv("AUTHZED_TOKEN")

	PermissionsClient    v1.PermissionsServiceClient
	PermissionSetsClient v0.WatchPermissionSetsServiceClient
)

func init() {
	buildClients()
	watchPermissionSets()
}

func buildClients() {
	systemCerts, err := grpcutil.WithSystemCerts(grpcutil.VerifyCA)
	if err != nil {
		panic(fmt.Sprintf("unable to load system CA certificates: %v", err))
	}

	spiceDBGRPCConn, err := grpc.NewClient(
		spiceDBEndpoint,
		systemCerts,
		grpcutil.WithBearerToken(authzedToken),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to  initialize conn: %v", err))
	}
	PermissionsClient = v1.NewPermissionsServiceClient(spiceDBGRPCConn)

	materializeGRPCConn, err := grpc.NewClient(
		materializeEndpoint,
		systemCerts,
		grpcutil.WithBearerToken(authzedToken),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to  initialize conn: %v", err))
	}
	PermissionSetsClient = v0.NewWatchPermissionSetsServiceClient(materializeGRPCConn)
}

func watchPermissionSets() {
	go func() {
		ctx, cancel := goctx.WithCancel(goctx.Background())
		defer cancel()

	OUTER:
		for {
			if PermissionSetsClient == nil {
				buildClients()
			}

			var rev *v1.ZedToken
			wpsr, err := PermissionSetsClient.WatchPermissionSets(ctx, &v0.WatchPermissionSetsRequest{
				OptionalStartingAfter: rev,
			})
			if err != nil {
				panic(fmt.Errorf("failed to open WatchPermissionSets stream: %w", err))
			}

			for {
				val, err := wpsr.Recv()
				if err != nil {
					// reset client
					log.Error("error receiving from WatchPermissionSets stream: %v, val: %v", err, val)
					PermissionSetsClient = nil
					continue OUTER
				}

				if change := val.GetChange(); change != nil {
					log.Info("change: %v\n", change)

					if s := change.GetChildSet(); s != nil {
						s2s := authzed_model.SetToSet{
							ChildType:      s.ObjectType,
							ChildID:        s.ObjectId,
							ChildRelation:  s.PermissionOrRelation,
							ParentType:     change.GetParentSet().ObjectType,
							ParentID:       change.GetParentSet().ObjectId,
							ParentRelation: change.GetParentSet().PermissionOrRelation,
						}
						if _, err := db.GetEngine(ctx).Insert(&s2s); err != nil {
							log.Error("error inserting SetToSet: %v", err)
						}
					} else if s := change.GetChildMember(); s != nil {
						m2s := authzed_model.MemberToSet{
							MemberType:     s.ObjectType,
							MemberID:       s.ObjectId,
							MemberRelation: s.OptionalPermissionOrRelation,
							SetType:        change.GetParentSet().ObjectType,
							SetID:          change.GetParentSet().ObjectId,
							SetRelation:    change.GetParentSet().PermissionOrRelation,
						}
						if _, err := db.GetEngine(ctx).Insert(&m2s); err != nil {
							log.Error("error inserting MemberToSet: %v", err)
						}
					}
				} else if completedRev := val.GetCompletedRevision(); completedRev != nil {
					log.Info("completed rev: %v\n", completedRev)
				} else {
					log.Info("unhandled type: %v\n", val)
				}
			}
		}
	}()
}

func RepoOwnerRel(repo *repo.Repository, owner *user.User) *v1.Relationship {
	return &v1.Relationship{
		Resource: &v1.ObjectReference{
			ObjectType: ResourceRepo,
			ObjectId:   strconv.FormatInt(repo.ID, 10),
		},
		Relation: RelationOwner,
		Subject: &v1.SubjectReference{Object: &v1.ObjectReference{
			ObjectType: SubjectUser,
			ObjectId:   strconv.FormatInt(owner.ID, 10),
		}},
	}
}

func RepoCollaboratorRel(repo *repo.Repository, collaborator *user.User) *v1.Relationship {
	return &v1.Relationship{
		Resource: &v1.ObjectReference{
			ObjectType: ResourceRepo,
			ObjectId:   strconv.FormatInt(repo.ID, 10),
		},
		Relation: RelationCollaborator,
		Subject: &v1.SubjectReference{Object: &v1.ObjectReference{
			ObjectType: SubjectUser,
			ObjectId:   strconv.FormatInt(collaborator.ID, 10),
		}},
	}
}
