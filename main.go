package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/validator.v2"
)

// DB stores the database session imformation
type DB struct {
	collection *mongo.Collection
}

// Task object
type Task struct {
	ID        interface{} `json:"id" bson:"_id,omitempty"`
	Title     string      `json:"title" bson:"title" validate:"nonzero"`
	Body      string      `json:"body" bson:"body"`
	Completed bool        `json:"completed" bson:"completed"`
	CreatedAt time.Time   `json:"createdat" bson:"createdat"`
}

// creatTask func creates a new task
func (db *DB) creatTask(w http.ResponseWriter, r *http.Request) {
	var task Task
	d := json.NewDecoder(r.Body)
	d.Decode(&task)

	// "TODO":we are gonna validate the data in frontend
	if err := validator.Validate(task); err != nil {
		log.Println(err.Error())
		http.Error(w, "invalid json request", http.StatusBadRequest)
		return
	}
	task.CreatedAt = time.Now()
	_, err := db.collection.InsertOne(context.TODO(), task)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("created succesfully!"))
}

// deleteTask removes one task from the db
func (db *DB) deleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	objectID, _ := primitive.ObjectIDFromHex(vars["id"])
	filter := bson.M{"_id": objectID}

	_, err := db.collection.DeleteOne(context.TODO(), filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Deleted succesfully!"))
}

// getTask fetches all tasks
func (db *DB) getTasks(w http.ResponseWriter, r *http.Request) {
	var tasks []Task

	filter := bson.M{}
	cur, err := db.collection.Find(context.TODO(), filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	// defer cur.Close(context.TODO()) // it's not needed if we dont use the loop
	// if its too big,use the loop
	/*
		for cur.Next(context.TODO()) {
			var task Task
			err := cur.Decode(&task)
			if err != nil {
				log.Fatal(err)
			}
			tasks = append(tasks, task)
		}

		if err := cur.Err(); err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	*/

	if err = cur.All(context.TODO(), &tasks); err != nil {
		log.Fatal(err)
	}
	if tasks == nil {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Let's create some task first"))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)

}

// completeTask just updates the field completed (partially update)
func (db *DB) completeTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	task := make(map[string]bool)

	d := json.NewDecoder(r.Body)
	d.Decode(&task)

	objectID, _ := primitive.ObjectIDFromHex(vars["id"])
	filter := bson.M{"_id": objectID}

	// partially update
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "completed", Value: task["completed"]}}}}
	_, err := db.collection.UpdateOne(context.TODO(), filter, update)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("task completed succesfully!"))
}

// updateTask modifies the data of given resource
func (db *DB) updateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var task Task
	d := json.NewDecoder(r.Body)
	d.Decode(&task)

	objectID, _ := primitive.ObjectIDFromHex(vars["id"])
	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": &task}
	_, err := db.collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Updated succesfully!"))
}

func main() {

	addr := flag.String("addr", ":8080", "HTTP network address")
	flag.Parse()

	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(context.TODO())

	collection := client.Database("taskDB").Collection("tasks")
	db := &DB{collection: collection}

	router := mux.NewRouter()

	router.HandleFunc("/", db.creatTask).Methods("POST")
	router.HandleFunc("/{id:[a-zA-Z0-9]*}", db.deleteTask).Methods("DELETE")
	router.HandleFunc("/", db.getTasks).Methods("GET")
	router.HandleFunc("/{id:[a-zA-Z0-9]*}", db.completeTask).Methods("PATCH")
	router.HandleFunc("/{id:[a-zA-Z0-9]*}", db.updateTask).Methods("PUT")

	srv := &http.Server{
		Addr:    *addr,
		Handler: router,
	}
	log.Printf("starting server on %s", *addr)
	log.Fatal(srv.ListenAndServe())
}
