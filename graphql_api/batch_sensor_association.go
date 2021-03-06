package graphql_api

import (
	"context"
	"database/sql"
	"errors"
	// "github.com/davecgh/go-spew/spew"
	"encoding/base64"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/middleware"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"log"
	"time"
)

type associateSensorToBatchInput struct {
	BatchId     string
	SensorId    string
	Description *string
}

type updateBatchSensorAssociationInput struct {
	ID              string
	Description     *string
	AssociatedAt    DateTime
	DisassociatedAt *DateTime
}

type batchSensorAssociationResolver struct {
	// Id string?
	// TODO: Maybe change this to just hold a pointer to the association struct, like other resolvers?
	assoc *worrywort.BatchSensor
	// Batch batchResolver
	// Sensor sensorResolver
	// Description string
	// AssociatedAt string // time!
	// DisassociatedAt *string // time!
}

func (b *batchSensorAssociationResolver) Id() graphql.ID { return graphql.ID(b.assoc.Id) }
func (b *batchSensorAssociationResolver) Batch() *batchResolver {
	// todo: take context and look it up if not already set?
	// Batch(ctx context.Context)
	return &batchResolver{b: b.assoc.Batch}
}
func (b *batchSensorAssociationResolver) Sensor() *sensorResolver {
	return &sensorResolver{s: b.assoc.Sensor}
}
func (b *batchSensorAssociationResolver) Description() *string { return &b.assoc.Description }
func (b *batchSensorAssociationResolver) AssociatedAt() DateTime {
	return DateTime{b.assoc.AssociatedAt}
}

func (b *batchSensorAssociationResolver) DisassociatedAt() *DateTime {
	if b.assoc.DisassociatedAt != nil {
		// nullableDateString
		d := DateTime{*b.assoc.DisassociatedAt}
		return &d
	}
	return nil
}

type batchSensorAssociationEdge struct {
	Cursor string
	Node   *batchSensorAssociationResolver
}

func (r *batchSensorAssociationEdge) CURSOR() string {
	// TODO: base4encode as part of MakeOffsetCursor instead?
	c := base64.StdEncoding.EncodeToString([]byte(r.Cursor))
	return c
}
func (r *batchSensorAssociationEdge) NODE() *batchSensorAssociationResolver { return r.Node }

// Going full relay, I suppose
// the graphql lib needs case-insensitive match of names on the methods
// so the resolver functions are just named all caps... alternately the
// struct members could be named as such to avoid a collision
// idea from https://github.com/deltaskelta/graphql-go-pets-example/blob/ab169fb644b1a00998208e7feede5975214d60da/users.go#L156
type batchSensorAssociationConnection struct {
	// if dataloader is implemented, this could just store the ids (and do a lighter query for those ids) and use dataloader
	// to get each individual edge or sensor and build the edge in the resolver function
	Edges    *[]*batchSensorAssociationEdge
	PageInfo *pageInfo
}

func (r *batchSensorAssociationConnection) PAGEINFO() pageInfo                    { return *r.PageInfo }
func (r *batchSensorAssociationConnection) EDGES() *[]*batchSensorAssociationEdge { return r.Edges }

// Mutation Payloads
type associateSensorToBatchPayload struct {
	assoc *batchSensorAssociationResolver
	// err *userErrorResolver
}

func (c *associateSensorToBatchPayload) BatchSensorAssociation() *batchSensorAssociationResolver {
	return c.assoc
}

// Seems like maybe a nested struct may be in order with the BatchSensorAssociation(), etc.
type updateBatchSensorAssociationPayload struct {
	assoc *batchSensorAssociationResolver
	// err *userErrorResolver
}

func (c *updateBatchSensorAssociationPayload) BatchSensorAssociation() *batchSensorAssociationResolver {
	return c.assoc
}

// func (c *associateSensorToBatchPayload) UserErrors() *userErrorResolver {
// 	return c.err
// }

// TODO: Rename this?  At least it's obvious what it is/does.
// Mutation to associate a sensor with a batch
func (r *Resolver) AssociateSensorToBatch(ctx context.Context, args *struct {
	Input *associateSensorToBatchInput
}) (*associateSensorToBatchPayload, error) {
	u, _ := middleware.UserFromContext(ctx)
	if u == nil {
		return nil, ErrUserNotAuthenticated
	}

	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		log.Printf("No database in context")
		return nil, ErrServerError
	}

	var inputPtr *associateSensorToBatchInput = args.Input
	var input associateSensorToBatchInput = *inputPtr

	batchPtr, err := worrywort.FindBatch(map[string]interface{}{"user_id": *u.Id, "uuid": input.BatchId}, db)
	if err != nil || batchPtr == nil {
		if err != sql.ErrNoRows {
			log.Printf("%v", err)
		}
		// TODO: return as a UserErrors similar to shopify to differentiate from a graphql syntax type error?
		return nil, errors.New("Specified Batch does not exist.")
	}

	// TODO!: Make sure the sensor is not already associated with a batch
	sensorPtr, err := worrywort.FindSensor(map[string]interface{}{"uuid": input.SensorId, "user_id": *u.Id}, db)
	if err != nil || sensorPtr == nil {
		// TODO: Probably need a friendlier error here or for our payload to have a shopify style userErrors
		// and then not ever return nil from this either way...maybe
		if err != sql.ErrNoRows {
			log.Printf("%v", err)
		}
		// TODO: only return THIS error if it really does not exist.  May need other errors
		// for other stuff
		return nil, errors.New("Specified Sensor does not exist.")
	}

	// TODO: Is this correct?  Maybe I really want to associate a sensor with 2 batches, such as for
	// ambient air temperature. Maybe this should only ensure it's not associated with the same batch twice.
	_, err = worrywort.FindBatchSensorAssociation(
		map[string]interface{}{"sensor_id": *sensorPtr.Id, "disassociated_at": nil, "user_id": *u.Id}, db)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("%v", err)
		return nil, ErrServerError
	} else if err == nil {
		// if we did not get any error back, that's actually an error. Wanted sql.ErrNoRows
		return nil, errors.New("Sensor already associated to Batch.")
	}

	var description string
	if input.Description != nil {
		description = *input.Description
	} else {
		description = ""
	}
	association, err := worrywort.AssociateBatchToSensor(batchPtr, sensorPtr, description, nil, db)
	if err != nil {
		log.Printf("%v", err)
		return nil, errors.New("Specified Sensor does not exist.")
	}

	association.Batch = batchPtr
	association.Sensor = sensorPtr
	resolvedAssoc := batchSensorAssociationResolver{assoc: association}
	result := associateSensorToBatchPayload{assoc: &resolvedAssoc}
	return &result, nil
}

func (r *Resolver) UpdatebatchSensorAssociation(ctx context.Context, args *struct {
	Input *updateBatchSensorAssociationInput
}) (*updateBatchSensorAssociationPayload, error) {
	u, _ := middleware.UserFromContext(ctx)
	if u == nil {
		return nil, ErrUserNotAuthenticated
	}
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		log.Printf("No database in context")
		return nil, ErrServerError
	}

	var inputPtr *updateBatchSensorAssociationInput = args.Input
	var input updateBatchSensorAssociationInput = *inputPtr

	var disassociatedAt *time.Time = nil
	if input.DisassociatedAt != nil {
		disassociatedAt = &input.DisassociatedAt.Time
	}
	associatedAt := input.AssociatedAt.Time
	association, err := worrywort.FindBatchSensorAssociation(
		map[string]interface{}{"id": string(input.ID), "user_id": u.Id}, db)

	if err == sql.ErrNoRows {
		return nil, errors.New("BatchSensorAssociation does not exist.")
	}
	if err != nil {
		return nil, err
	}

	var description string
	if input.Description != nil {
		description = *input.Description
	} else {
		description = ""
	}
	association.Description = description
	association.AssociatedAt = associatedAt
	association.DisassociatedAt = disassociatedAt

	association, err = worrywort.UpdateBatchSensorAssociation(*association, db)
	if err != nil {
		return nil, err
	}

	resolvedAssoc := batchSensorAssociationResolver{assoc: association}
	result := updateBatchSensorAssociationPayload{assoc: &resolvedAssoc}
	return &result, nil
}
