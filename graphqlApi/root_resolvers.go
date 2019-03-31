package graphqlApi

import (
	"context"
	// "fmt"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"log"
	// "os"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"
	    "encoding/base64"
)

// log.SetFlags(log.LstdFlags | log.Lshortfile)
var SERVER_ERROR = errors.New("Unexpected server error.")

// This also could be handled in middleware, but then I would need two separate
// schemas and routes - one for authenticated stuff, one for
var NOT_AUTHTENTICATED_ERROR = errors.New("User must be authenticated")

// Takes a time.Time and returns nil if the time is zero or pointer to the time string formatted as RFC3339
func nullableDateString(dt time.Time) *string {
	if dt.IsZero() {
		return nil
	}
	dtString := dt.Format(time.RFC3339)
	return &dtString
}

func dateString(dt time.Time) string {
	return dt.Format(time.RFC3339)
}

// move these somewhere central
type pageInfo struct {
	HasNextPage     bool
	HasPreviousPage bool
}

func (r pageInfo) HASNEXTPAGE() bool     { return r.HasNextPage }
func (r pageInfo) HASPREVIOUSPAGE() bool { return r.HasPreviousPage }

type Resolver struct {
	// todo: should be Db?
	// do not really need this now that it is coming in on context so code is inconsistent.
	// but on context is considered "not good"... I could pass this around instead, but would then
	// need to either attach a Resolver or db to every single data type, which also kind of sucks
	db *sqlx.DB
}

/* This is the root resolver */
func NewResolver(db *sqlx.DB) *Resolver {
	// Lshortfile tells me too little - filename, but not which package it is in, etc.
	// Llongfile tells me too much - the full path at build from the go root. I really just need from the project root dir.
	log.SetFlags(log.LstdFlags | log.Llongfile)
	return &Resolver{db: db}
}

func (r *Resolver) CurrentUser(ctx context.Context) (*userResolver, error) {
	// This ensures we have the right type from the context
	// may change to just "authMiddleware" or something though so that
	// a single function can exist to get user from any of the auth methods
	// or just write a separate function for that here instead of using it from authMiddleware.
	// TODO: should check errors
	u, _ := authMiddleware.UserFromContext(ctx)
	ur := userResolver{u: &u}
	return &ur, nil
}

// handle errors by returning error with 403?
// func sig: func (r *Resolver) Batch(ctx context.Context, args struct{ ID graphql.ID }) (*batchResolver, error) {
func (r *Resolver) Batch(ctx context.Context, args struct{ ID graphql.ID }) (*batchResolver, error) {
	// TODO: panic on error, no user, etc.
	u, _ := authMiddleware.UserFromContext(ctx)
	var err error
	batchArgs := make(map[string]interface{})
	// TODO: Or if batch is publicly readable by anyone?
	batchArgs["user_id"] = u.Id
	batchArgs["uuid"] = args.ID
	// batchArgs["id"], err = strconv.ParseInt(string(args.ID), 10, 0)

	if err != nil {
		log.Printf("%v", err)
		return nil, nil
	}

	batchPtr, err := worrywort.FindBatch(batchArgs, r.db)
	if err != nil {
		// do not expose sql errors
		if err != sql.ErrNoRows {
			log.Printf("%v", err)
		}
		return nil, nil
	}
	if batchPtr == nil {
		return nil, nil
	}
	return &batchResolver{b: batchPtr}, nil
}

func (r *Resolver) Batches(ctx context.Context, args struct {
	First *int
	After *string
}) (*batchConnection, error) {
	u, _ := authMiddleware.UserFromContext(ctx)
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, SERVER_ERROR
	}

	batches, err := worrywort.FindBatches(map[string]interface{}{"user_id": u.Id}, db)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("%v", err)
		return nil, err
	}
	edges := []*batchEdge{}
	for index, _ := range batches {
		resolvedBatch := batchResolver{b: batches[index]}
		// should base64 encode this cursor, but whatever for now
		edge := &batchEdge{Node: &resolvedBatch, Cursor: string(resolvedBatch.ID())}
		edges = append(edges, edge)
	}
	hasNextPage := false
	hasPreviousPage := false
	return &batchConnection{
		PageInfo: &pageInfo{HasNextPage: hasNextPage, HasPreviousPage: hasPreviousPage},
		Edges:    &edges}, nil
}

func (r *Resolver) BatchSensorAssociations(ctx context.Context, args struct {
	First    *int
	After    *string
	BatchId  *string
	SensorId *string
}) (*batchSensorAssociationConnection, error) {
	u, _ := authMiddleware.UserFromContext(ctx)
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, SERVER_ERROR
	}
	var offset int
	queryparams := map[string]interface{}{"user_id": u.Id}

	if args.After != nil && *args.After != "" {
		// TODO: Put this somewhere reusable!!
		raw, err := base64.StdEncoding.DecodeString(*args.After)
		if err != nil {
			panic(err)
		}
		var cursordata struct{ Offset int }
		json.Unmarshal(raw, &cursordata)
		offset = cursordata.Offset
		queryparams["offset"] = offset
	}

	if args.First != nil {
		queryparams["limit"] = *args.First + 1 // +1 to easily see if there are more
	}

	associations, err := worrywort.FindBatchSensorAssociations(queryparams, db)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("%v", err)
		return nil, err
	}

	hasNextPage := false
	hasPreviousPage := false
	edges := []*batchSensorAssociationEdge{}
	for i, assoc := range associations {
		if i <= *args.First {
			resolved := batchSensorAssociationResolver{assoc: assoc}
			// TODO: Not 100% sure about this. We have current offset + current index + 1 where the extra 1
			// is added so that the offset value in the cursor will be to start at the NEXT item, which feels odd
			// since the param used is "After". This could optionally add the 1 to the incoming data
			// which might feel more natural
			cursorval := offset + i + 1
			edge := &batchSensorAssociationEdge{Node: &resolved, Cursor: MakeOffsetCursor(cursorval)}
			edges = append(edges, edge)
		} else {
			// had one more than was actually requested, there is a next page
			hasNextPage = true
		}
	}

	return &batchSensorAssociationConnection{
		PageInfo: &pageInfo{HasNextPage: hasNextPage, HasPreviousPage: hasPreviousPage},
		Edges:    &edges}, nil
}

func (r *Resolver) Fermentor(ctx context.Context, args struct{ ID graphql.ID }) (*fermentorResolver, error) {
	// authUser, _ := authMiddleware.UserFromContext(ctx)
	// TODO: Implement correctly!  Look up the Fermentor with FindFermentor
	return nil, errors.New("Not Implemented") // so that it is obvious this is no implemented
}

func (r *Resolver) Sensor(ctx context.Context, args struct{ ID graphql.ID }) (*sensorResolver, error) {
	user, _ := authMiddleware.UserFromContext(ctx)
	var resolved *sensorResolver
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, SERVER_ERROR
	}

	sensor, err := worrywort.FindSensor(map[string]interface{}{"uuid": string(args.ID), "user_id": *user.Id}, db)
	if err != nil {
		log.Printf("%v", err)
		return nil, nil // maybe error should be returned
	} else if sensor != nil && sensor.Uuid != "" {
		// TODO: check for Uuid is a hack because I need to rework FindSensor to return nil
		// if it did not find a sensor
		resolved = &sensorResolver{s: sensor}
	} else {
		resolved = nil
	}
	return resolved, err
}

func (r *Resolver) Sensors(ctx context.Context, args struct {
	First *int
	After *string
}) (*sensorConnection, error) {
	authUser, _ := authMiddleware.UserFromContext(ctx)
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, SERVER_ERROR
	}

	// Now get the temperature sensors, build out the info
	sensors, err := worrywort.FindSensors(map[string]interface{}{"user_id": authUser.Id}, db)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("%v", err)
		return nil, err
	}
	edges := []*sensorEdge{}
	for index, _ := range sensors {
		sensorResolver := sensorResolver{s: sensors[index]}
		// should base64 encode this cursor, but whatever for now
		edge := &sensorEdge{Node: &sensorResolver, Cursor: string(sensorResolver.ID())}
		edges = append(edges, edge)
	}
	hasNextPage := false
	hasPreviousPage := false
	return &sensorConnection{
		PageInfo: &pageInfo{HasNextPage: hasNextPage, HasPreviousPage: hasPreviousPage},
		Edges:    &edges}, nil
}

// Returns a single resolved TemperatureMeasurement by ID, owned by the authenticated user
func (r *Resolver) TemperatureMeasurement(ctx context.Context, args struct{ ID graphql.ID }) (*temperatureMeasurementResolver, error) {
	authUser, _ := authMiddleware.UserFromContext(ctx)
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, SERVER_ERROR
	}
	var resolved *temperatureMeasurementResolver
	measurementId := string(args.ID)
	measurement, err := worrywort.FindTemperatureMeasurement(
		map[string]interface{}{"id": measurementId, "user_id": authUser.Id}, db)
	if err != nil {
		log.Printf("%v", err)
	} else if measurement != nil {
		resolved = &temperatureMeasurementResolver{m: measurement}
	}
	return resolved, err
}

func (r *Resolver) TemperatureMeasurements(ctx context.Context, args struct {
	First    *int
	After    *string
	SensorId *string
	BatchId  *string
}) (*temperatureMeasurementConnection, error) {
	authUser, _ := authMiddleware.UserFromContext(ctx)
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, SERVER_ERROR
	}

	// TODO: pagination, the rest of the optional filter params
	measurements, err := worrywort.FindTemperatureMeasurements(map[string]interface{}{"user_id": authUser.Id}, db)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("%v", err)
		return nil, err
	}
	edges := []*temperatureMeasurementEdge{}
	for index, _ := range measurements {
		measurementResolver := temperatureMeasurementResolver{m: measurements[index]}
		// should base64 encode this cursor, but whatever for now
		edge := &temperatureMeasurementEdge{Node: &measurementResolver, Cursor: string(measurementResolver.ID())}
		edges = append(edges, edge)
	}
	hasNextPage := false
	hasPreviousPage := false
	return &temperatureMeasurementConnection{
		PageInfo: &pageInfo{HasNextPage: hasNextPage, HasPreviousPage: hasPreviousPage},
		Edges:    &edges}, nil
}

// Input types
// Create a temperatureMeasurement... review docs on how to really implement this
type createTemperatureMeasurementInput struct {
	RecordedAt  string //time.Time
	Temperature float64
	SensorId    graphql.ID
	Units       string // it seems this graphql server cannot handle mapping enum to struct inputs
}

// Mutation Payloads
type createTemperatureMeasurementPayload struct {
	t *temperatureMeasurementResolver
}

func (c createTemperatureMeasurementPayload) TemperatureMeasurement() *temperatureMeasurementResolver {
	return c.t
}

// Mutations

// Create a temperature measurementId
// TODO: move me to temperature_measurement.go ??
func (r *Resolver) CreateTemperatureMeasurement(ctx context.Context, args *struct {
	Input *createTemperatureMeasurementInput
}) (*createTemperatureMeasurementPayload, error) {
	// TODO: use db from context rather than r.Db for consistency throughout API
	u, _ := authMiddleware.UserFromContext(ctx)

	var inputPtr *createTemperatureMeasurementInput = args.Input
	// TODO: make sure input was not nil. Technically the schema does this for us
	// but might be safer to handle here, too, or at least have a test case for it.
	var input createTemperatureMeasurementInput = *inputPtr
	var unitType worrywort.TemperatureUnitType

	// bleh.  Too bad this lib doesn't map the input types with enums/iota correctly
	if input.Units == "FAHRENHEIT" {
		unitType = worrywort.FAHRENHEIT
	} else {
		unitType = worrywort.CELSIUS
	}

	sensorId, err := strconv.ParseInt(string(input.SensorId), 10, 32)
	sensorPtr, err := worrywort.FindSensor(map[string]interface{}{"id": sensorId, "user_id": u.Id}, r.db)
	if err != nil {
		// TODO: Probably need a friendlier error here or for our payload to have a shopify style userErrors
		// and then not ever return nil from this either way...maybe
		if err != sql.ErrNoRows {
			log.Printf("%v", err)
		}
		// TODO: only return THIS error if it really does not exist.  May need other errors
		// for other stuff
		return nil, errors.New("Specified Sensor does not exist.")
	}

	// for actual iso 8601, use "2006-01-02T15:04:05-0700"
	// TODO: test parsing both
	recordedAt, err := time.Parse(time.RFC3339, input.RecordedAt)
	if err != nil {
		// TODO: See what the actual error types are and try to return friendlier errors which are not golang specific messaging
		return nil, err
	}

	t := worrywort.TemperatureMeasurement{Sensor: sensorPtr, SensorId: sensorPtr.Id,
		Temperature: input.Temperature, Units: unitType, RecordedAt: recordedAt, CreatedBy: &u, UserId: u.Id}
	if err := t.Save(r.db); err != nil {
		log.Printf("Failed to save TemperatureMeasurement: %v\n", err)
		return nil, err
	}
	tr := temperatureMeasurementResolver{m: &t}
	result := createTemperatureMeasurementPayload{t: &tr}
	return &result, nil
}

func (r *Resolver) Login(args *struct {
	Username string
	Password string
}) (*authTokenResolver, error) {
	user, err := worrywort.AuthenticateLogin(args.Username, args.Password, r.db)
	// TODO: Check for errors which should not be exposed?  Or for known good errors to expose
	// and return something more generic + log if unexpected?
	if err != nil {
		return nil, err
	}

	token, err := worrywort.GenerateTokenForUser(*user, worrywort.TOKEN_SCOPE_ALL)
	if err != nil {
		log.Printf("*****ERRR*****\n%v\n", err)
		return nil, err
	}
	tokenPtr := &token

	err = tokenPtr.Save(r.db)
	if err != nil {
		log.Printf("%s", err)
		return nil, err
	}
	atr := authTokenResolver{t: token}
	return &atr, err
}
