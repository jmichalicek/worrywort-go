package main

import (
	"fmt"
	"github.com/go-chi/chi"
	chimiddleware "github.com/go-chi/chi/middleware"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/jmichalicek/worrywort-server-go/graphql_api"
	"github.com/jmichalicek/worrywort-server-go/middleware"
	"github.com/jmichalicek/worrywort-server-go/rest_api"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	// "github.com/davecgh/go-spew/spew"
)

var schema *graphql.Schema

// Returns a function for looking up a user by token for middleware.NewTokenAuthHandler()
// which closes over the db needed to look up the user
func newTokenAuthLookup(db *sqlx.DB) func(token string) (*worrywort.User, error) {
	return func(token string) (*worrywort.User, error) {
		// TODO: return the token? That could be more useful in many places than just the user.
		t, err := worrywort.AuthenticateUserByToken(token, db)
		return &t.User, err
	}
}

func main() {
	// For now, force postgres
	// TODO: write something to parse db uri?
	// I suspect this already would and I just didn't read the docs correctly.
	// Using LookupEnv because I will probably add some sane defaults... such as localhost for
	dbName, _ := os.LookupEnv("DATABASE_NAME")
	dbUser, _ := os.LookupEnv("DATABASE_USER")
	dbPassword, _ := os.LookupEnv("DATABASE_PASSWORD")
	dbHost, _ := os.LookupEnv("DATABASE_HOST")
	dbPort, dbPortSet := os.LookupEnv("DATABASE_PORT")
	if !dbPortSet {
		dbPort = "5432" // again, assume postgres
	}
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, _ := sqlx.Connect("postgres", connectionString)
	schema = graphql.MustParseSchema(graphql_api.Schema, graphql_api.NewResolver(db))

	// could do a middleware in this style to add db to the context like I used to, but more middleware friendly.
	// Could also do that to add a logger, etc. For now, that stuff is getting attached to each handler
	tokenAuthHandler := middleware.NewTokenAuthHandler(newTokenAuthLookup(db))
	authRequiredHandler := middleware.NewLoginRequiredHandler()

	// Not really sure I needed to switch to Chi here instead of the built in stuff.
	// TODO: Test the actual server being built here. Will maybe need some restructuring of this
	// to have a master struct which has the router, routes, etc.
	// See https://github.com/nerdyc/testable-golang-web-service
	// will want something like this
	// type Server struct {
	//   Db *sqlx.DB
	//   Router CHI_ROUTER_TYPE
	// }
	r := chi.NewRouter()
	r.Use(chimiddleware.Compress(5, "text/html", "application/javascript"))
	r.Use(chimiddleware.Logger)
	r.Use(tokenAuthHandler)
	r.Handle("/graphql", &graphql_api.Handler{Db: db, Handler: &relay.Handler{Schema: schema}})
	r.Method("POST", "/api/v1/measurement", authRequiredHandler(&rest_api.MeasurementHandler{Db: db}))
	// TODO: need to manually handle CORS? Chi has some cors stuff, yay
	// https://github.com/graph-gophers/graphql-go/issues/74#issuecomment-289098639
	uri, uriSet := os.LookupEnv("WORRYWORTD_HOST")
	if !uriSet {
		uri = ":8080"
	}
	log.Printf("WorryWort now listening on %s\n", uri)
	log.Fatal(http.ListenAndServe(uri, r))
}
