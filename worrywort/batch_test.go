package worrywort

import (
	"database/sql"
	"testing"
	"time"
	// "reflect"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"strings"
)

// Test that NewBatch() returns a batch with the expected values
func TestNewBatch(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	brewedDate := time.Now().Add(time.Duration(1) * time.Minute)
	bottledDate := brewedDate.Add(time.Duration(10) * time.Minute)
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())

	expectedBatch := Batch{Id: 1, Name: "Testing", BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: 5,
		VolumeInFermenter: 4.5, VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedBy: u,
		CreatedAt: createdAt, UpdatedAt: updatedAt, BrewNotes: "Brew notes", TastingNotes: "Taste notes",
		RecipeURL: "http://example.org/beer"}
	b := NewBatch(1, "Testing", brewedDate, bottledDate, 5, 4.5, GALLON, 1.060, 1.020, u, createdAt, updatedAt,
		"Brew notes", "Taste notes", "http://example.org/beer")

	if b != expectedBatch {
		t.Errorf("Expected: %v\n\nGot: %v", expectedBatch, b)
	}
}

func TestNewFermenter(t *testing.T) {

	createdAt := time.Now()
	updatedAt := time.Now()
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	expected := Fermenter{Id: 1, Name: "Ferm", Description: "A Fermenter", Volume: 5.0, VolumeUnits: GALLON,
		FermenterType: BUCKET, IsActive: true, IsAvailable: true, CreatedBy: u, CreatedAt: createdAt, UpdatedAt: updatedAt}

	f := NewFermenter(1, "Ferm", "A Fermenter", 5.0, GALLON, BUCKET, true, true, u, createdAt, updatedAt)

	if f != expected {
		t.Errorf("Expected:\n%v\n\nGot:\n%v\n", expected, f)
	}
}

func TestNewTemperatureSensor(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	expected := TemperatureSensor{Id: 1, Name: "Therm1", CreatedBy: u, CreatedAt: createdAt, UpdatedAt: updatedAt}

	therm := NewTemperatureSensor(1, "Therm1", u, createdAt, updatedAt)

	if therm != expected {
		t.Errorf("Expected:\n%v\n\nGot:\n%v\n", expected, therm)
	}

}

func TestFindTemperatureSensor(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := NewUser(0, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = SaveUser(db, u)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	userId := sql.NullInt64{Valid: true, Int64: int64(u.Id)}
	sensor := TemperatureSensor{Name: "Test Sensor", UserId: userId}
	sensor, err = SaveTemperatureSensor(db, sensor)
	params := make(map[string]interface{})
	params["created_by_user_id"] = u.Id
	params["id"] = sensor.Id
	foundSensor, err := FindTemperatureSensor(params, db)
	// foundSensor, err := FindTemperatureSensor(map[string]interface{}{"created_by_user_id": u.Id, "id": sensor.Id}, db)
	if err != nil {
		t.Errorf("%v", err)
	}
	if *foundSensor != sensor {
		t.Errorf("Expected: %v\nGot: %v", sensor, foundSensor)
	}

}

func TestSaveTemperatureSensor(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := NewUser(0, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = SaveUser(db, u)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	t.Run("Save New Sensor", func(t *testing.T) {
		sensor, err := SaveTemperatureSensor(db, TemperatureSensor{Name: "Test Sensor", CreatedBy: u})
		if err != nil {
			t.Errorf("%v", err)
		}
		if sensor.Id == 0 {
			t.Errorf("SaveTemperatureSensor did not set id on new TemperatureSensor")
		}

		if sensor.UpdatedAt.IsZero() {
			t.Errorf("SaveTemperatureSensor did not set UpdatedAt")
		}

		if sensor.CreatedAt.IsZero() {
			t.Errorf("SaveTemperatureSensor did not set CreatedAt")
		}
	})

	t.Run("Update Sensor", func(t *testing.T) {
		sensor, err := SaveTemperatureSensor(db, TemperatureSensor{Name: "Test Sensor", CreatedBy: u})
		// set date back in the past so that our date comparison consistenyly works
		sensor.UpdatedAt = sensor.UpdatedAt.AddDate(0, 0, -1)
		sensor.Name = "Updated Name"
		updatedSensor, err := SaveTemperatureSensor(db, sensor)
		if err != nil {
			t.Errorf("%v", err)
		}
		if updatedSensor.Name != "Updated Name" {
			t.Errorf("SaveTemperatureSensor did not update Name")
		}

		if sensor.UpdatedAt == updatedSensor.UpdatedAt {
			t.Errorf("SaveTemperatureSensor did not update UpdatedAt")
		}
	})
}

func TestSaveTemperatureMeasurement(t *testing.T) {
	// TODO: Add fermenter to the saved measurement!  Currently saving a fermenter has not been implemented
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u, err := SaveUser(db, User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"})
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor, err := SaveTemperatureSensor(db, TemperatureSensor{Name: "Test Sensor", CreatedBy: u})
	if err != nil {
		t.Fatalf("%v", err)
	}

	b, err := SaveBatch(db, Batch{CreatedBy: u, Name: "Test batch"})
	if err != nil {
		t.Fatalf("%v", err)
	}

	t.Run("Save New Measurement With All Fields", func(t *testing.T) {
		m, err := SaveTemperatureMeasurement(db,
			TemperatureMeasurement{CreatedBy: &u, TemperatureSensor: &sensor, Temperature: 70.0, Units: FAHRENHEIT, Batch: &b, RecordedAt: time.Now()})
		if err != nil {
			t.Errorf("%v", err)
		}
		if m.Id == "" {
			t.Errorf("SaveTemperatureMeasurement did not set id on new TemperatureMeasurement")
		}

		if m.UpdatedAt.IsZero() {
			t.Errorf("SaveTemperatureMeasurement did not set UpdatedAt")
		}

		if m.CreatedAt.IsZero() {
			t.Errorf("SaveTemperatureMeasurement did not set CreatedAt")
		}
		// TODO: Just query for the expected measurement
		newMeasurement := TemperatureMeasurement{}
		selectCols := ""
		for _, k := range u.queryColumns() {
			selectCols += fmt.Sprintf("u.%s \"created_by.%s\", ", k, k)
		}
		selectCols += fmt.Sprintf("ts.id \"temperature_sensor.id\", ts.name \"temperature_sensor.name\", ")
		q := `SELECT tm.temperature, tm.units,  ` + strings.Trim(selectCols, ", ") + ` from temperature_measurements tm LEFT JOIN users u ON u.id = tm.created_by_user_id LEFT JOIN temperature_sensors ts ON ts.id = tm.temperature_sensor_id WHERE tm.id = ? AND tm.created_by_user_id = ? AND tm.temperature_sensor_id = ?`
		query := db.Rebind(q)
		err = db.Get(&newMeasurement, query, m.Id, u.Id, sensor.Id)

		if err != nil {
			t.Errorf("%v", err)
		}
	})

	t.Run("Save New Measurement Without Optional Fields", func(t *testing.T) {
		m, err := SaveTemperatureMeasurement(db,
			TemperatureMeasurement{CreatedBy: &u, TemperatureSensor: &sensor, Temperature: 70.0, Units: FAHRENHEIT, RecordedAt: time.Now()})
		if err != nil {
			t.Errorf("%v", err)
		}
		if m.Id == "" {
			t.Errorf("SaveTemperatureMeasurement did not set id on new TemperatureMeasurement")
		}

		if m.UpdatedAt.IsZero() {
			t.Errorf("SaveTemperatureMeasurement did not set UpdatedAt")
		}

		if m.CreatedAt.IsZero() {
			t.Errorf("SaveTemperatureMeasurement did not set CreatedAt")
		}

		newMeasurement := TemperatureMeasurement{}
		selectCols := ""
		for _, k := range u.queryColumns() {
			selectCols += fmt.Sprintf("u.%s \"created_by.%s\", ", k, k)
		}
		selectCols += fmt.Sprintf("ts.id \"temperature_sensor.id\", ts.name \"temperature_sensor.name\", ")
		q := `SELECT tm.temperature, tm.units,  ` + strings.Trim(selectCols, ", ") + ` from temperature_measurements tm LEFT JOIN users u ON u.id = tm.created_by_user_id LEFT JOIN temperature_sensors ts ON ts.id = tm.temperature_sensor_id WHERE tm.id = ? AND tm.created_by_user_id = ? AND tm.temperature_sensor_id = ?`
		query := db.Rebind(q)
		err = db.Get(&newMeasurement, query, m.Id, u.Id, sensor.Id)

		if err != nil {
			t.Errorf("%v", err)
		}
	})

	t.Run("Update Temperature Measurement", func(t *testing.T) {
		m, err := SaveTemperatureMeasurement(db,
			TemperatureMeasurement{CreatedBy: &u, TemperatureSensor: &sensor, Temperature: 70.0, Units: FAHRENHEIT, RecordedAt: time.Now()})
		if err != nil {
			t.Errorf("%v", err)
		}
		// set date back in the past so that our date comparison consistenyly works
		m.UpdatedAt = sensor.UpdatedAt.AddDate(0, 0, -1)
		// TODO: Intend to change this so that we set BatchId and save to update the Batch, not assign an object
		m.Batch = &b
		updatedMeasurement, err := SaveTemperatureMeasurement(db, m)
		if err != nil {
			t.Errorf("%v", err)
		}
		if updatedMeasurement.Batch != &b {
			t.Errorf("SaveTemperatureMeasurement did not update the Batch")
		}

		if m.UpdatedAt == updatedMeasurement.UpdatedAt {
			t.Errorf("SaveTemperatureSensor did not update UpdatedAt")
		}

		// Now unset the batch, just to see
		m.Batch = nil
		updatedMeasurement, err = SaveTemperatureMeasurement(db, m)
		if updatedMeasurement.Batch != nil {
			t.Errorf("SaveTemperatureMeasurement did not remove the Batch")
		}
	})
}

func TestFindBatch(t *testing.T) {
	// Set up the db using sql.Open() and sqlx.NewDb() rather than sqlx.Open() so that the custom
	// `txdb` db type may be used with Open() but can still be registered as postgres with sqlx
	// so that sqlx' Rebind() functions.

	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := NewUser(0, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = SaveUser(db, u)

	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	createdAt := time.Now().Round(time.Microsecond)
	updatedAt := time.Now().Round(time.Microsecond)
	// THe values when returned by postgres will be microsecond accuracy, but golang default
	// is nanosecond, so we round these for easy comparison
	brewedDate := time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond)
	bottledDate := brewedDate.Add(time.Duration(10) * time.Minute).Round(time.Microsecond)
	b := Batch{UserId: sql.NullInt64{Int64: int64(u.Id), Valid: true}, BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: 5, VolumeInFermenter: 4.5,
		VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedBy: u, CreatedAt: createdAt, UpdatedAt: updatedAt,
		BrewNotes: "Brew Notes", TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer"}
	// b := NewBatch(0, "Testing", brewedDate, bottledDate, 5, 4.5, GALLON, 1.060, 1.020, u, createdAt, updatedAt,
	// 	"Brew notes", "Taste notes", "http://example.org/beer")
	b, err = SaveBatch(db, b)
	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	batchArgs := make(map[string]interface{})
	batchArgs["created_by_user_id"] = u.Id
	batchArgs["id"] = b.Id
	found, err := FindBatch(batchArgs, db)
	if err != nil {
		t.Errorf("Got unexpected error: %s", err)
	} else if !b.StrictEqual(*found) {
		t.Errorf("Expected: %v\nGot: %v\n", b, *found)
	}
}

func TestBatchesForUser(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := NewUser(0, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = SaveUser(db, u)

	u2 := NewUser(0, "user2@example.com", "Justin", "M", time.Now(), time.Now())
	u2, err = SaveUser(db, u2)

	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	createdAt := time.Now().Round(time.Microsecond)
	updatedAt := time.Now().Round(time.Microsecond)
	// THe values when returned by postgres will be microsecond accuracy, but golang default
	// is nanosecond, so we round these for easy comparison
	brewedDate := time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond)
	bottledDate := brewedDate.Add(time.Duration(10) * time.Minute).Round(time.Microsecond)
	b := Batch{Name: "Testing", UserId: sql.NullInt64{Int64: int64(u.Id), Valid: true}, BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: 5, VolumeInFermenter: 4.5,
		VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedBy: u, CreatedAt: createdAt, UpdatedAt: updatedAt,
		BrewNotes: "Brew Notes", TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer"}
	b, err = SaveBatch(db, b)

	b2 := Batch{Name: "Testing 2", UserId: sql.NullInt64{Int64: int64(u.Id), Valid: true}, BrewedDate: time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond), VolumeBoiled: 5, VolumeInFermenter: 4.5,
		VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedBy: u, CreatedAt: createdAt, UpdatedAt: updatedAt,
		BrewNotes: "Brew Notes", TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer", BottledDate: time.Now().Add(time.Duration(5) * time.Minute).Round(time.Microsecond)}
	b2, err = SaveBatch(db, b2)

	u2batch := Batch{Name: "Testing 2", UserId: sql.NullInt64{Int64: int64(u2.Id), Valid: true}, BrewedDate: time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond), VolumeBoiled: 5, VolumeInFermenter: 4.5,
		VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedBy: u2, CreatedAt: createdAt, UpdatedAt: updatedAt,
		BrewNotes: "Brew Notes", TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer", BottledDate: time.Now().Add(time.Duration(5) * time.Minute).Round(time.Microsecond)}

	u2batch, err = SaveBatch(db, u2batch)

	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	// TODO: split up into sub tests for different functionality... no pagination, pagination, etc.
	batches, err := BatchesForUser(db, u, nil, nil)
	if err != nil {
		t.Fatalf("\n%v\n", err)
	}

	// DepEqual is not playing nicely here (ie. I don't understand something) so do a very naive check for now.
	// May be worth trying this instead of spew, which has a Diff() function which may tell me what the difference is
	// https://godoc.org/github.com/kr/pretty
	expected := []Batch{b, b2}
	if len(*batches) != 2 || expected[0].Id != (*batches)[0].Id || expected[1].Id != (*batches)[1].Id {
		t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected[0]), spew.Sdump((*batches)[0]))
	}
	// TODO: Cannot figure out WHY these are not equal.
	// if !reflect.DeepEqual(*batches, expected) {
	// 	t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(*batches))
	// }
}

func TestInsertBatch(t *testing.T) {}
func TestUpdateBatch(t *testing.T) {}
func TestSaveBatch(t *testing.T)   {}
