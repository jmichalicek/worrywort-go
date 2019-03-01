package worrywort

import (
	"database/sql"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewFermentor(t *testing.T) {

	createdAt := time.Now()
	updatedAt := time.Now()
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	expected := Fermentor{Id: 1, Name: "Ferm", Description: "A Fermentor", Volume: 5.0, VolumeUnits: GALLON,
		FermentorType: BUCKET, IsActive: true, IsAvailable: true, CreatedBy: &u, CreatedAt: createdAt, UpdatedAt: updatedAt}

	f := NewFermentor(1, "Ferm", "A Fermentor", 5.0, GALLON, BUCKET, true, true, u, createdAt, updatedAt)

	if !reflect.DeepEqual(f, expected) {
		t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(f))
	}
	// if f != expected {
	// 	t.Errorf("Expected:\n%v\n\nGot:\n%v\n", expected, f)
	// }
}

func TestSaveFermentor(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u, err := SaveUser(db, User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"})
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}
	userId := sql.NullInt64{Valid: true, Int64: int64(u.Id)}

	t.Run("New Fermentor", func(t *testing.T) {
		fermentor, err := SaveFermentor(db, Fermentor{Name: "Fermentor", Description: "Fermentor Desc", Volume: 5.0,
			VolumeUnits: GALLON, FermentorType: BUCKET, IsActive: true, IsAvailable: true, UserId: userId})
		if err != nil {
			t.Errorf("%v", err)
		}
		if fermentor.Id == 0 {
			t.Errorf("SaveFermentor did not set id on new Fermentor")
		}

		if fermentor.UpdatedAt.IsZero() {
			t.Errorf("SaveFermentor did not set UpdatedAt")
		}

		if fermentor.CreatedAt.IsZero() {
			t.Errorf("SaveFermentor did not set CreatedAt")
		}
	})

	t.Run("Update Fermentor", func(t *testing.T) {
		fermentor, err := SaveFermentor(db, Fermentor{Name: "Fermentor", Description: "Fermentor Desc", Volume: 5.0,
			VolumeUnits: GALLON, FermentorType: BUCKET, IsActive: true, IsAvailable: true, UserId: userId})
		// set date back in the past so that our date comparison consistenyly works
		fermentor.UpdatedAt = fermentor.UpdatedAt.AddDate(0, 0, -1)
		fermentor.Name = "Updated Name"
		updatedFermentor, err := SaveFermentor(db, fermentor)
		if err != nil {
			t.Errorf("%v", err)
		}
		if updatedFermentor.Name != "Updated Name" {
			t.Errorf("SaveFermentor did not update Name")
		}

		if fermentor.UpdatedAt == updatedFermentor.UpdatedAt {
			t.Errorf("SaveFermentor did not update UpdatedAt")
		}
	})
}

func TestNewSensor(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	expected := Sensor{Id: 1, Name: "Therm1", CreatedBy: &u, CreatedAt: createdAt, UpdatedAt: updatedAt}

	therm := NewSensor(1, "Therm1", &u, createdAt, updatedAt)

	if therm != expected {
		t.Errorf("Expected:\n%v\n\nGot:\n%v\n", expected, therm)
	}

}

func TestFindSensor(t *testing.T) {
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
	sensor := Sensor{Name: "Test Sensor", UserId: userId}
	sensor, err = SaveSensor(db, sensor)
	params := make(map[string]interface{})
	params["user_id"] = u.Id
	params["id"] = sensor.Id
	foundSensor, err := FindSensor(params, db)
	// foundSensor, err := FindSensor(map[string]interface{}{"user_id": u.Id, "id": sensor.Id}, db)
	if err != nil {
		t.Errorf("%v", err)
	}
	if *foundSensor != sensor {
		t.Errorf("Expected: %v\nGot: %v", sensor, foundSensor)
	}

}

func TestSaveSensor(t *testing.T) {
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
		sensor, err := SaveSensor(db, Sensor{Name: "Test Sensor", CreatedBy: &u})
		if err != nil {
			t.Errorf("%v", err)
		}
		if sensor.Id == 0 {
			t.Errorf("SaveSensor did not set id on new Sensor")
		}

		if sensor.UpdatedAt.IsZero() {
			t.Errorf("SaveSensor did not set UpdatedAt")
		}

		if sensor.CreatedAt.IsZero() {
			t.Errorf("SaveSensor did not set CreatedAt")
		}
	})

	t.Run("Update Sensor", func(t *testing.T) {
		sensor, err := SaveSensor(db, Sensor{Name: "Test Sensor", CreatedBy: &u})
		// set date back in the past so that our date comparison consistenyly works
		sensor.UpdatedAt = sensor.UpdatedAt.AddDate(0, 0, -1)
		sensor.Name = "Updated Name"
		updatedSensor, err := SaveSensor(db, sensor)
		if err != nil {
			t.Errorf("%v", err)
		}
		if updatedSensor.Name != "Updated Name" {
			t.Errorf("SaveSensor did not update Name")
		}

		if sensor.UpdatedAt == updatedSensor.UpdatedAt {
			t.Errorf("SaveSensor did not update UpdatedAt")
		}
	})
}

func TestSaveTemperatureMeasurement(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u, err := SaveUser(db, User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"})
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}
	userId := sql.NullInt64{Valid: true, Int64: int64(u.Id)}

	sensor, err := SaveSensor(db, Sensor{Name: "Test Sensor", CreatedBy: &u})
	if err != nil {
		t.Fatalf("%v", err)
	}
	sensorId := sql.NullInt64{Valid: true, Int64: int64(sensor.Id)}

	b, err := SaveBatch(db, Batch{CreatedBy: &u, Name: "Test batch"})
	if err != nil {
		t.Fatalf("%v", err)
	}
	batchId := sql.NullInt64{Valid: true, Int64: int64(b.Id)}

	t.Run("Save New Measurement With All Fields", func(t *testing.T) {
		m, err := SaveTemperatureMeasurement(db,
			TemperatureMeasurement{CreatedBy: &u, UserId: userId, Sensor: &sensor, SensorId: sensorId,
				Temperature: 70.0, Units: FAHRENHEIT, Batch: &b, BatchId: batchId, RecordedAt: time.Now()})
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
		selectCols += fmt.Sprintf("ts.id \"sensor.id\", ts.name \"sensor.name\", ")
		q := `SELECT tm.temperature, tm.units,  ` + strings.Trim(selectCols, ", ") + ` from temperature_measurements tm LEFT JOIN users u ON u.id = tm.user_id LEFT JOIN sensors ts ON ts.id = tm.sensor_id WHERE tm.id = ? AND tm.user_id = ? AND tm.sensor_id = ?`
		query := db.Rebind(q)
		err = db.Get(&newMeasurement, query, m.Id, u.Id, sensor.Id)

		if err != nil {
			t.Errorf("%v", err)
		}
	})

	t.Run("Save New Measurement Without Optional Fields", func(t *testing.T) {
		m, err := SaveTemperatureMeasurement(db,
			TemperatureMeasurement{CreatedBy: &u, UserId: userId, SensorId: sensorId, Sensor: &sensor, Temperature: 70.0, Units: FAHRENHEIT, RecordedAt: time.Now()})
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
		q := `SELECT tm.temperature, tm.units, tm.user_id, tm.sensor_id from temperature_measurements tm LEFT JOIN users u ON u.id = tm.user_id LEFT JOIN sensors ts ON ts.id = tm.sensor_id WHERE tm.id = ? AND tm.user_id = ? AND tm.sensor_id = ?`
		query := db.Rebind(q)
		err = db.Get(&newMeasurement, query, m.Id, u.Id, sensor.Id)

		if err != nil {
			t.Errorf("%v", err)
		}
	})

	t.Run("Update Temperature Measurement", func(t *testing.T) {
		m, err := SaveTemperatureMeasurement(db,
			TemperatureMeasurement{CreatedBy: &u, UserId: userId, SensorId: sensorId, Sensor: &sensor, Temperature: 70.0, Units: FAHRENHEIT, RecordedAt: time.Now()})
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
			t.Errorf("SaveSensor did not update UpdatedAt. Expected: %v\nGot: %v", m.UpdatedAt, updatedMeasurement.UpdatedAt)
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
	b := Batch{UserId: sql.NullInt64{Int64: int64(u.Id), Valid: true}, BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: 5, VolumeInFermentor: 4.5,
		VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedAt: createdAt, UpdatedAt: updatedAt,
		BrewNotes: "Brew Notes", TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer"}
	b, err = SaveBatch(db, b)
	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	batchArgs := make(map[string]interface{})
	batchArgs["user_id"] = u.Id
	batchArgs["id"] = b.Id
	found, err := FindBatch(batchArgs, db)
	if err != nil {
		t.Errorf("Got unexpected error: %s", err)
	} else if b.Id != (*found).Id {
		t.Errorf("Expected: %v\nGot: %v\n", b, *found)
	}
}

func TestFindBatches(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := NewUser(0, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = SaveUser(db, u)
	userId := sql.NullInt64{Int64: int64(u.Id), Valid: true}

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
	b := Batch{Name: "Testing", UserId: userId, BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: 5, VolumeInFermentor: 4.5,
		VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedAt: createdAt, UpdatedAt: updatedAt,
		BrewNotes: "Brew Notes", TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer"}
	b, err = SaveBatch(db, b)

	b2 := Batch{Name: "Testing 2", UserId: userId, BrewedDate: time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond), VolumeBoiled: 5, VolumeInFermentor: 4.5,
		VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedAt: createdAt, UpdatedAt: updatedAt,
		BrewNotes: "Brew Notes", TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer", BottledDate: time.Now().Add(time.Duration(5) * time.Minute).Round(time.Microsecond)}
	b2, err = SaveBatch(db, b2)

	u2batch := Batch{Name: "Testing 2", UserId: sql.NullInt64{Int64: int64(u2.Id), Valid: true}, BrewedDate: time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond), VolumeBoiled: 5, VolumeInFermentor: 4.5,
		VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedAt: createdAt, UpdatedAt: updatedAt,
		BrewNotes: "Brew Notes", TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer", BottledDate: time.Now().Add(time.Duration(5) * time.Minute).Round(time.Microsecond)}

	u2batch, err = SaveBatch(db, u2batch)

	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	// TODO: split up into sub tests for different functionality... no pagination, pagination, etc.
	batches, err := FindBatches(map[string]interface{}{"user_id": userId}, db)
	if err != nil {
		t.Fatalf("\n%v\n", err)
	}

	// DepEqual is not playing nicely here (ie. I don't understand something) so do a very naive check for now.
	// May be worth trying this instead of spew, which has a Diff() function which may tell me what the difference is
	// https://godoc.org/github.com/kr/pretty
	expected := []*Batch{&b, &b2}
	if len(batches) != 2 || expected[0].Id != batches[0].Id || expected[1].Id != batches[1].Id {
		t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected[0]), spew.Sdump(batches[0]))
	}
	// TODO: Cannot figure out WHY these are not equal.
	// Suspect it is because it is lists of different pointers
	// if !reflect.DeepEqual(batches, expected) {
	// 	t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(batches))
	// }
}

func TestInsertBatch(t *testing.T) {}
func TestUpdateBatch(t *testing.T) {}
func TestSaveBatch(t *testing.T)   {}

func TestBatchSenssorAssociations(t *testing.T) {

	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := NewUser(0, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = SaveUser(db, u)
	userId := sql.NullInt64{Int64: int64(u.Id), Valid: true}

	brewedDate := time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond)
	bottledDate := brewedDate.Add(time.Duration(10) * time.Minute).Round(time.Microsecond)
	batch := Batch{Name: "Testing", UserId: userId, BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: 5,
		VolumeInFermentor: 4.5, VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, BrewNotes: "Brew Notes",
		TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer"}
	batch, err = SaveBatch(db, batch)
	if err != nil {
		t.Fatalf("%v", err)
	}

	sensor, err := SaveSensor(db, Sensor{Name: "Test Sensor", CreatedBy: &u})
	if err != nil {
		t.Fatalf("%v", err)
	}

	cleanAssociations := func() {
		q := `DELETE FROM batch_sensor_association WHERE sensor_id = ? AND batch_id = ?`
		q = db.Rebind(q)
		_, err := db.Exec(q, sensor.Id, batch.Id)
		if err != nil {
			panic(err)
		}
	}

	t.Run("AssociateBatchToSensor()", func(t *testing.T) {
		defer cleanAssociations()
		association, err := AssociateBatchToSensor(batch, sensor, "Testing", nil, db)
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		var newAssociation BatchSensor
		q := `SELECT bs.id, bs.sensor_id, bs.batch_id, bs.description, bs.associated_at, bs.disassociated_at, bs.created_at,
			bs.updated_at FROM batch_sensor_association bs WHERE bs.id = ? AND bs.sensor_id = ? AND bs.batch_id = ?
			AND bs.description = ? AND bs.associated_at = ? AND bs.created_at = ? AND bs.updated_at = ?
			AND bs.disassociated_at IS NULL`
		query := db.Rebind(q)
		err = db.Get(&newAssociation, query, association.Id, sensor.Id, batch.Id, "Testing", association.AssociatedAt,
			association.CreatedAt, association.UpdatedAt)

		if err != nil {
			t.Fatalf("%v", err)
		}

		// Make sure these really got set
		if (*association).AssociatedAt.IsZero() {
			t.Errorf("AssociateBatchToSensor did not set AssociatedAt")
		}

		if (*association).UpdatedAt.IsZero() {
			t.Errorf("AssociateBatchToSensor did not set UpdatedAt")
		}

		if (*association).CreatedAt.IsZero() {
			t.Errorf("AssociateBatchToSensor did not set CreatedAt")
		}
	})

	t.Run("UpdateBatchSensorAssociation()", func(t *testing.T) {
		defer cleanAssociations()
		aPtr, err := AssociateBatchToSensor(batch, sensor, "Testing", nil, db)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		association := *aPtr
		association.Description = "Updated"
		updated, err := UpdateBatchSensorAssociation(association, db)
		if err != nil {
			t.Fatalf("%v", err)
		}
		// Making sure the change was persisted to the db
		updated2 := BatchSensor{}
		q := `SELECT id, sensor_id, batch_id, description, associated_at, disassociated_at, created_at,
			updated_at FROM batch_sensor_association bs WHERE id = ? AND sensor_id = ? AND batch_id = ? AND description = ?
			AND disassociated_at IS NULL`
		query := db.Rebind(q)
		err = db.Get(&updated2, query, association.Id, association.SensorId, association.BatchId, "Updated")
		if err != nil {
			t.Fatalf("%v", err)
		}

		if !reflect.DeepEqual(*updated, updated2) {
			t.Errorf("Expected: %s\nGot: %s. Changes may not have persisted to the database.", spew.Sdump(updated), spew.Sdump(updated2))
		}
	})
}
