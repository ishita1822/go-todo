package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var rnd *renderer.Render
var db *mgo.Database
var mongoDBURL string

// mongodb://localhost:27017
const (
	hostname       string = "localhost"
	dbName         string = "demo_todo"
	collectionName string = "todo"
	port           string = ":27018"
)

type todoModel struct {
	ID        bson.ObjectId `bson:"_id,omitempty"`
	Title     string        `bson:title`
	Completed bool          `bson:"completed"`
	CreatedAT time.Time     `bson:"createdAt"`
}

type todo struct {
	ID        string    `json:"od"`
	Title     string    `json:title`
	Completed bool      `json:"completed"`
	CreatedAT time.Time `json:"created_at"`
}

func init() {
	rnd = renderer.New()
	mongoDBURL = "mongodb://" + hostname + port
	sess, err := mgo.Dial(mongoDBURL)
	checkErr(err)
	sess.SetMode(mgo.Monotonic, true)
}

// w -> response  r-> request
func homeHandler(w http.ResponseWriter, r *http.Request) {
	err := rnd.Template(w, http.StatusOK, []string{"static/home.tpi"}, nil)
	checkErr(err)
}

func fetchTodos(w http.ResponseWriter, r *http.Request) {
	todos := []todoModel{}

	err := db.C(collectionName).Find(bson.M{}).All(&todos)
	if err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"Message": "Failed to fetch todo",
			"error":   err,
		})
		return
	}

	todoList := []todo{}

	for _, t := range todos {
		todoList = append(todoList, todo{
			ID:        t.ID.Hex(),
			Title:     t.Title,
			Completed: t.Completed,
			CreatedAT: t.CreatedAT,
		})
	}

	rnd.JSON(w, http.StatusOK, renderer.M{
		"data": todoList,
	})
}

func createTodo(w http.ResponseWriter, r *http.Request) {
	var t todo

	err := json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		rnd.JSON(w, http.StatusProcessing, err)
		return
	}

	if t.Title == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"Message": "The title is required",
		})
		return
	}

	tm := todoModel{
		ID:        bson.NewObjectId(),
		Title:     t.Title,
		Completed: false,
		CreatedAT: time.Now(),
	}

	err = db.C(collectionName).Insert(&tm)
	if err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"Message": "Failed to save todo ",
			"error":   err,
		})
		return
	}

	rnd.JSON(w, http.StatusCreated, renderer.M{
		"Message": "Todo created successfully",
		"todo_id": tm.ID.Hex(),
	})
}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	if !bson.IsObjectIdHex(id) {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"Message": "The id is invalid",
		})
		return
	}

	err := db.C(collectionName).RemoveId(bson.ObjectIdHex(id))
	if err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"Message": "Failed to delete todo",
			"error":   err,
		})
		return
	}

	rnd.JSON(w, http.StatusOK, renderer.M{
		"Message": "todo deleted successfully",
	})
}

func updateTodo(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	if !bson.IsObjectIdHex(id) {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"Message": "The id is invalid",
		})
		return
	}

	var t todo

	err := json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		rnd.JSON(w, http.StatusProcessing, err)
		return
	}

	if t.Title == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"Message": "The title field id is reqiured",
		})
		return
	}

	err = db.C(collectionName).Update(
		bson.M{"_id": bson.ObjectIdHex(id)},
		bson.M{"title": t.Title, "completed": t.Completed},
	)
	if err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"Message": "Failed to update todo",
			"error":   err,
		})
		return
	}
}

func main() {
	// stopChannel := make(chan os.Signal)
	// signal.Notify(stopChannel, os.Interrupt)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", homeHandler)
	r.Mount("/todo", todoHandlers())

	server := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("Listening on port ", port)

		if err := server.ListenAndServe(); err != nil {
			log.Printf("listen: %s\n", err)
		}
	}()

	// <-stopChannel
	// log.Println("Shutting down server...")
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// server.Shutdown(ctx)
	// defer cancel()
	// log.Println("Server Gracefully stopped")
}

func todoHandlers() http.Handler {
	rg := chi.NewRouter()
	rg.Group(func(r chi.Router) {
		r.Get("/", fetchTodos)
		r.Post("/", createTodo)
		r.Put("/{id}", updateTodo)
		r.Delete("/{id}", deleteTodo)
	})

	return rg
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
