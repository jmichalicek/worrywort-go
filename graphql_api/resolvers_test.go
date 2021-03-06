package graphql_api

import (
	"context"
	"database/sql"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"reflect"
	"testing"
	"time"
	// "fmt"
	"github.com/google/go-cmp/cmp"
)

func setUpTestDb() (*sqlx.DB, error) {
	_db, err := sql.Open("txdb", "one")
	if err != nil {
		return nil, err
	}

	db := sqlx.NewDb(_db, "postgres")
	if err != nil {
		return nil, err
	}

	return db, nil
}

// utility to add a given number of minutes to a time.Time and round to match
// what postgres returns
func addMinutes(d time.Time, increment int) time.Time {
	return d.Add(time.Duration(increment) * time.Minute).Round(time.Microsecond)
}

// Make a standard, generic batch for testing
// optionally attach the user
func makeTestBatch(u worrywort.User, attachUser bool) worrywort.Batch {
	bottledDate := addMinutes(time.Now(), 10)
	b := worrywort.Batch{Name: "Testing", BrewedDate: addMinutes(time.Now(), 1), BottledDate: &bottledDate,
		VolumeBoiled: 5, VolumeInFermentor: 4.5, VolumeUnits: worrywort.GALLON, OriginalGravity: 1.060, FinalGravity: 1.020,
		UserId: u.Id, BrewNotes: "Brew notes", TastingNotes: "Taste notes", RecipeURL: "http://example.org/beer",
		UUID: uuid.New().String()}
	if attachUser {
		b.CreatedBy = &u
	}
	return b
}

func TestUserResolver(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	createdAt := time.Now()
	updatedAt := time.Now()
	uid := int64(1)
	u := worrywort.User{Id: &uid, UUID: uuid.New().String(), Email: "user@example.com", FullName: "Justin Michalicek",
		Username: "worrywort", CreatedAt: createdAt, UpdatedAt: updatedAt}
	r := userResolver{u: &u}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = r.ID()
		expected := graphql.ID(u.UUID)
		if ID != expected {
			t.Errorf("Expected: %v, got: %v", expected, ID)
		}
	})

	t.Run("FullName()", func(t *testing.T) {
		var firstName string = r.FullName()
		expected := "Justin Michalicek"
		if firstName != expected {
			t.Errorf("Expected: %v, got: %v", expected, firstName)
		}
	})

	t.Run("Username()", func(t *testing.T) {
		var username string = r.Username()
		expected := "worrywort"
		if username != expected {
			t.Errorf("Expected: %v, got: %v", expected, username)
		}
	})

	t.Run("Email()", func(t *testing.T) {
		var email string = r.Email()
		expected := "user@example.com"
		if email != expected {
			t.Errorf("Expected: %v, got: %v", expected, email)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		// TODO: Would like to test that these come out as rfc3339 strings...
		var dt DateTime = r.CreatedAt()
		expected := u.CreatedAt
		if dt != (DateTime{expected}) {
			t.Errorf("Expected: %v, got: %v", expected, dt)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var dt DateTime = r.UpdatedAt()
		expected := u.UpdatedAt
		if dt != (DateTime{expected}) {
			t.Errorf("Expected: %v, got %v", expected, dt)
		}
	})
}

func TestBatchResolver(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.User{Email: "user@example.com", FullName: "Justin Michalicek", Username: "worrywort"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	brewed := makeTestBatch(u, true)
	bId := int64(1)
	brewed.Id = &bId
	unbrewed := makeTestBatch(u, true)
	unbrewed.BrewedDate = time.Time{}
	unbrewed.BottledDate = nil

	br := batchResolver{b: &brewed}
	unbr := batchResolver{b: &unbrewed}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = br.ID()
		expected := graphql.ID(brewed.UUID)
		if ID != expected {
			t.Errorf("Expected: %v, got: %v", expected, ID)
		}
	})

	t.Run("Name()", func(t *testing.T) {
		var name string = br.Name()
		expected := "Testing"
		if name != expected {
			t.Errorf("Expected: %v, got: %v", expected, name)
		}
	})

	t.Run("BrewNotes()", func(t *testing.T) {
		var notes string = br.BrewNotes()
		expected := "Brew notes"
		if notes != expected {
			t.Errorf("Expected: %v, got: %v", expected, notes)
		}
	})

	t.Run("TastingNotes()", func(t *testing.T) {
		var notes string = br.TastingNotes()
		expected := "Taste notes"
		if notes != expected {
			t.Errorf("Expected: %v, got: %v", expected, notes)
		}
	})

	t.Run("BrewedDate()", func(t *testing.T) {
		var dt *DateTime = br.BrewedDate()
		expected := DateTime{brewed.BrewedDate}
		if *dt != expected {
			t.Errorf("Expected: %v, got: %v", expected, dt)
		}

		unbrewedDate := unbr.BrewedDate()
		if unbrewedDate != nil {
			t.Errorf("Expected: nil but got %v", unbrewedDate)
		}
	})

	t.Run("BottledDate()", func(t *testing.T) {
		var dt *DateTime = br.BottledDate()
		expected := DateTime{*brewed.BottledDate}
		if *dt != expected {
			t.Errorf("Expected: %v, got: %v", expected, dt)
		}

		unbrewedDate := unbr.BottledDate()
		if unbrewedDate != nil {
			t.Errorf("Expected: nil but got %v", unbrewedDate)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var dt DateTime = br.CreatedAt()
		expected := brewed.CreatedAt //.Format(time.RFC3339)
		if dt.Time != expected {
			t.Errorf("Expected: %v, got: %v", expected, dt)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var dt DateTime = br.UpdatedAt()
		expected := brewed.UpdatedAt //.Format(time.RFC3339)
		if dt.Time != expected {
			t.Errorf("Expected: %v, got %v", expected, dt)
		}
	})

	t.Run("VolumeBoiled()", func(t *testing.T) {
		var actual *float64 = br.VolumeBoiled()
		expected := brewed.VolumeBoiled
		// direct comparison seems to be ok, probably since no math is happening
		// but may be better to do like this:
		// if math.Abs(*boiled - expected) > .0000000001 {
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("VolumeInFermentor()", func(t *testing.T) {
		var actual *float64 = br.VolumeInFermentor()
		expected := brewed.VolumeInFermentor
		// direct comparison seems to be ok, probably since no math is happening
		// but may be better to do like this:
		// if math.Abs(*boiled - expected) > .0000000001 {
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("OriginalGravity()", func(t *testing.T) {
		var actual *float64 = br.OriginalGravity()
		expected := brewed.OriginalGravity
		// direct comparison seems to be ok, probably since no math is happening
		// but may be better to do like this:
		// if math.Abs(*boiled - expected) > .0000000001 {
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("FinalGravity()", func(t *testing.T) {
		var actual *float64 = br.FinalGravity()
		expected := brewed.FinalGravity
		// direct comparison seems to be ok, probably since no math is happening
		// but may be better to do like this:
		// if math.Abs(*boiled - expected) > .0000000001 {
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("VolumeUnits()", func(t *testing.T) {
		var actual worrywort.VolumeUnitType = br.VolumeUnits()
		expected := brewed.VolumeUnits

		if actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("RecipeURL()", func(t *testing.T) {
		var actual string = br.RecipeURL()
		expected := brewed.RecipeURL

		if actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("CreatedBy() with User struct populated", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "db", db)

		actual, err := br.CreatedBy(ctx)
		if err != nil {
			t.Errorf("%v", err)
		}
		expected := userResolver{u: brewed.CreatedBy}
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("CreatedBy() without User populated", func(t *testing.T) {
		batchNoUser := makeTestBatch(u, false)
		err = batchNoUser.Save(db)
		if err != nil {
			t.Fatalf("%v", err)
		}
		ctx := context.Background()
		ctx = context.WithValue(ctx, "db", db)
		r := batchResolver{b: &batchNoUser}
		actual, err := r.CreatedBy(ctx)
		if err != nil {
			t.Errorf("%v", err)
		}
		expected := &userResolver{u: &u}

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(actual))
		}
	})
}

func TestFermentorResolver(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)

	u := worrywort.User{Email: "user@example.com", FullName: "Justin Michalicek", Username: "worrywort"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}
	fId := int64(1)
	f := worrywort.Fermentor{Id: &fId, CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: "Ferm", Description: "A Fermentor", Volume: 5.0, VolumeUnits: worrywort.GALLON,
		FermentorType: worrywort.BUCKET, IsActive: true, IsAvailable: true, CreatedBy: &u, UserId: u.Id,
		UUID: uuid.New().String()}
	r := fermentorResolver{f: &f}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = r.ID()
		expected := graphql.ID(f.UUID)
		if ID != expected {
			t.Errorf("Expected: %v, got: %v", expected, ID)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var dt DateTime = r.CreatedAt()
		expected := DateTime{f.CreatedAt}
		if dt != expected {
			t.Errorf("Expected: %v, got: %v", expected, dt)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var dt DateTime = r.UpdatedAt()
		expected := DateTime{f.UpdatedAt}
		if dt != expected {
			t.Errorf("Expected: %v, got %v", expected, dt)
		}
	})

	t.Run("CreatedBy()", func(t *testing.T) {
		var actual *userResolver = r.CreatedBy(ctx)
		expected := userResolver{u: f.CreatedBy}
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("CreatedBy() without User populated", func(t *testing.T) {
		var f2 worrywort.Fermentor = f
		f2.CreatedBy = nil
		r := fermentorResolver{f: &f2}
		actual := r.CreatedBy(ctx)
		expected := &userResolver{u: &u}

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(actual))
		}
	})
}

func TestSensorResolver(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)

	u := worrywort.User{Email: "user@example.com", FullName: "Justin Michalicek", Username: "worrywort"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sId := int64(1)
	sensor := worrywort.Sensor{Id: &sId, Name: "Therm1", UserId: u.Id, CreatedBy: &u, CreatedAt: time.Now(),
		UpdatedAt: time.Now(), UUID: uuid.New().String()}
	r := sensorResolver{s: &sensor}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = r.ID()
		expected := graphql.ID(sensor.UUID)
		if ID != expected {
			t.Errorf("Expected: %v, got: %v", expected, ID)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var dt DateTime = r.CreatedAt()
		expected := DateTime{sensor.CreatedAt}
		if dt != expected {
			t.Errorf("Expected: %v, got: %v", expected, dt)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var dt DateTime = r.UpdatedAt()
		expected := DateTime{sensor.UpdatedAt}
		if dt != expected {
			t.Errorf("Expected: %v, got %v", expected, dt)
		}
	})

	t.Run("CreatedBy()", func(t *testing.T) {
		var actual *userResolver = r.CreatedBy(ctx)
		expected := userResolver{u: sensor.CreatedBy}
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("CreatedBy() without User populated", func(t *testing.T) {
		var s2 worrywort.Sensor = sensor
		s2.CreatedBy = nil
		s2.Id = nil
		if err := s2.Save(db); err != nil {
			t.Fatalf("%v", err)
		}
		r2 := sensorResolver{s: &s2}
		actual := r2.CreatedBy(ctx)
		expected := &userResolver{u: &u}

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(actual))
		}
	})
}

func TestTemperatureMeasurementResolver(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)

	u := worrywort.User{Email: "user@example.com", FullName: "Justin Michalicek", Username: "worrywort"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	if err != nil {
		t.Fatalf("%v", err)
	}
	sensor := worrywort.Sensor{Name: "Therm1", UserId: u.Id, CreatedBy: &u}
	if err := sensor.Save(db); err != nil {
		t.Fatalf("%v", err)
	}
	batch := makeTestBatch(u, false)
	if err := batch.Save(db); err != nil {
		t.Fatalf("%v", err)
	}
	assoc, err := worrywort.AssociateBatchToSensor(&batch, &sensor, "", nil, db)
	if err != nil {
		t.Fatalf("%v", err)
	}
	timeRecorded := assoc.AssociatedAt.Add(time.Minute * time.Duration(1))
	measurement := worrywort.TemperatureMeasurement{Temperature: 64.26, Units: worrywort.FAHRENHEIT,
		RecordedAt: timeRecorded, SensorId: sensor.Id, Sensor: &sensor, CreatedBy: &u, UserId: u.Id, CreatedAt: time.Now(),
		UpdatedAt: time.Now()}
	if err := measurement.Save(db); err != nil {
		t.Fatalf("%v", err)
	}
	resolver := temperatureMeasurementResolver{m: &measurement}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = resolver.ID()
		expected := graphql.ID(measurement.Id)
		if ID != expected {
			t.Errorf("\nExpected: %v\ngot: %v", expected, ID)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var dt DateTime = resolver.CreatedAt()
		expected := DateTime{measurement.CreatedAt}
		if dt != expected {
			t.Errorf("\nExpected: %v\ngot: %v", expected, dt)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var dt DateTime = resolver.UpdatedAt()
		expected := DateTime{measurement.UpdatedAt}
		if dt != expected {
			t.Errorf("\nExpected: %v\ngot %v", expected, dt)
		}
	})

	t.Run("Temperature()", func(t *testing.T) {
		temp := resolver.Temperature()
		if measurement.Temperature != temp {
			t.Errorf("\nExpected: %v\ngot: %v", measurement.Temperature, temp)
		}
	})

	t.Run("Units()", func(t *testing.T) {
		units := resolver.Units()
		if measurement.Units != units {
			t.Errorf("\nExpected: %v\ngot: %v", measurement.Units, units)
		}
	})

	t.Run("Batch()", func(t *testing.T) {
		b := resolver.Batch(ctx)
		expected := batchResolver{b: &batch}
		cmpOpts := []cmp.Option{
			cmp.AllowUnexported(*b),
			cmp.AllowUnexported(*b.b),
		}
		if !cmp.Equal(*b, expected, cmpOpts...) {
			t.Errorf(cmp.Diff(*b, expected, cmpOpts...))
		}
	})

	t.Run("Sensor()", func(t *testing.T) {
		ts := resolver.Sensor(ctx)
		expected := sensorResolver{s: measurement.Sensor}
		if expected != *ts {
			t.Errorf("\nExpected: %v\ngot: %v", expected, ts)
		}
	})

	t.Run("CreatedBy() with User attached", func(t *testing.T) {
		// TODO: This test with user not already populated
		var actual *userResolver = resolver.CreatedBy(ctx)
		expected := userResolver{u: measurement.CreatedBy}
		if *actual != expected {
			t.Errorf("\nExpected: %v\ngot %v", expected, actual)
		}
	})

}

func TestAuthTokenResolver(t *testing.T) {
	uId := int64(1)
	u := worrywort.User{Id: &uId, Email: "user@example.com", FullName: "Justin Michalicek", Username: "worrywort",
		CreatedAt: time.Now(), UpdatedAt: time.Now()}

	token := worrywort.NewLoginToken("token", u, worrywort.TOKEN_SCOPE_ALL)
	token.Id = "tokenid"
	r := authTokenResolver{t: token}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = r.ID()
		expected := graphql.ID(token.ForAuthenticationHeader())
		if ID != expected {
			t.Errorf("\nExpected: %v\ngot: %v", expected, ID)
		}
	})

	t.Run("Token()", func(t *testing.T) {
		var tokenStr string = r.Token()
		expected := token.ForAuthenticationHeader()
		if tokenStr != expected {
			t.Errorf("\nExpected: %v\ngot: %v", expected, tokenStr)
		}
	})
}

// Placeholder tests which were copied/pasted and so going to fail
func TestBatchSensorAssociationResolver(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)

	u := worrywort.User{Email: "user@example.com", FullName: "Justin Michalicek", Username: "worrywort"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor := worrywort.Sensor{Name: "Therm1", UserId: u.Id, CreatedBy: &u}
	if err := sensor.Save(db); err != nil {
		t.Fatalf("%v", err)
	}
	batch := makeTestBatch(u, true)
	if err := batch.Save(db); err != nil {
		t.Fatalf("%v", err)
	}
	association := worrywort.BatchSensor{
		BatchId: batch.Id, SensorId: sensor.Id, Batch: &batch, Sensor: &sensor, Description: "Description",
		AssociatedAt: time.Now()}
	resolver := batchSensorAssociationResolver{assoc: &association}

	// t.Run("ID()", func(t *testing.T) {
	// 	var ID graphql.ID = r.ID()
	// 	expected := graphql.ID(token.ForAuthenticationHeader())
	// 	if ID != expected {
	// 		t.Errorf("\nExpected: %v\ngot: %v", expected, ID)
	// 	}
	// })

	t.Run("Batch()", func(t *testing.T) {
		// Test when Batch is set
		// modify to take/pass in context so batch can be looked up by id
		b := resolver.Batch()
		expected := batchResolver{b: association.Batch}
		if expected != *b {
			t.Errorf("\nExpected: %v\ngot: %v", expected, *b)
		}
		// Test when only BatchId is set?
	})

	t.Run("Sensor()", func(t *testing.T) {
		s := resolver.Sensor()
		expected := sensorResolver{s: &sensor}
		if expected != *s {
			t.Errorf("\nExpected: %v\ngot: %v", expected, *s)
		}
	})

	t.Run("AssociatedAt()", func(t *testing.T) {
		a := resolver.AssociatedAt()
		expected := DateTime{association.AssociatedAt}
		if a != expected {
			t.Errorf("\nExpected: %s\nGot: %v", expected, a)
		}
	})

	t.Run("DisassociatedAt()", func(t *testing.T) {
		d := resolver.DisassociatedAt()
		if d != nil {
			t.Errorf("\nExpected: nil\nGot: %v", d)
		}

		n := time.Now()
		association.DisassociatedAt = &n
		d = resolver.DisassociatedAt()
		expected := DateTime{*association.DisassociatedAt}
		if *d != expected {
			t.Errorf("\nExpected: %s\nGot: %s", expected, *d)
		}

	})

	t.Run("Description()", func(t *testing.T) {
		d := resolver.Description()
		// or compare to association.Description? Always torn on this...
		// Make sure it matches the value expected or make sure it matches the value of
		// the object populating it, whatever that might be?
		if *d != "Description" {
			t.Errorf("\nExpected: Description\ngot: %v", d)
		}
	})
}
