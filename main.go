package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var recipes []Recipe
var ctx context.Context
var err error
var client *mongo.Client

func init() {
	recipes = make([]Recipe, 0)
	file, _ := ioutil.ReadFile("recipes.json")
	_ = json.Unmarshal([]byte(file), &recipes)

	ctx = context.Background()

	client, err = mongo.Connect(ctx,

		options.Client().ApplyURI(os.Getenv("MONGO_URI")))

	if err = client.Ping(context.TODO(),

		readpref.Primary()); err != nil {

		log.Fatal(err)

	}

	log.Println("Connected to MongoDB")

	// var listOfRecipes []interface{}
	// for _, recipe := range recipes {
	// 	listOfRecipes = append(listOfRecipes, recipe)
	// }

	// collection := client.Database(os.Getenv("MONGO_DATABASE")).Collection("recipes")
	// insertManyResult, err := collection.InsertMany(ctx, listOfRecipes)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Println("Inserted recipes: ", len(insertManyResult.InsertedIDs))
}

type Recipe struct {
	ID           primitive.ObjectID `json:"id" bson:"_id"`
	Name         string             `json:"name" bson:"name"`
	Tags         []string           `json:"tags" bson:"tags"`
	Ingredients  []string           `json:"ingredients" bson:"ingredients"`
	Instructions []string           `json:"instructions"`
	PublishedAt  time.Time          `json:"publishedAt"`
}

func NewRecipeHandler(c *gin.Context) {
	var recipe Recipe
	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error()})
		return
	}

	recipe.ID = primitive.NewObjectID()
	recipe.PublishedAt = time.Now()
	collection := client.Database(os.Getenv("MONGO_DATABASE")).Collection("recipes")
	_, err := collection.InsertOne(ctx, recipe)
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while inserting a new recipe"})
		return
	}
	c.JSON(http.StatusOK, recipe)
}

func ListRecipeHandler(c *gin.Context) {
	collection := client.Database(os.Getenv("MONGO_DATABASE")).Collection("recipes")
	cur, err := collection.Find(ctx, bson.M{})

	if err != nil {

		c.JSON(http.StatusInternalServerError,

			gin.H{"error": err.Error()})

		return

	}

	defer cur.Close(ctx)

	recipes := make([]Recipe, 0)

	for cur.Next(ctx) {

		var recipe Recipe

		cur.Decode(&recipe)

		recipes = append(recipes, recipe)

	}

	c.JSON(http.StatusOK, recipes)
}

func UpdateRecipeHandler(c *gin.Context) {
	id := c.Param("id")

	var recipe Recipe

	if err := c.ShouldBindJSON(&recipe); err != nil {

		c.JSON(http.StatusBadRequest, gin.H{

			"error": err.Error()})

		return

	}
	collection := client.Database(os.Getenv("MONGO_DATABASE")).Collection("recipes")
	objectId, _ := primitive.ObjectIDFromHex(id)
	_, err = collection.UpdateOne(ctx, bson.M{
		"_id": objectId,
	}, bson.D{{"$set", bson.D{
		{"name", recipe.Name},
		{"instructions", recipe.Instructions},
		{"ingredients", recipe.Ingredients},
		{"tags", recipe.Tags},
	}}})
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Recipe has been updated"})
}

// func DeleteRecipeHandler(c *gin.Context) {
// 	id := c.Param("id")
// 	index := -1
// 	for i := 0; i < len(recipes); i++ {
// 		if recipes[i].ID == id {
// 			index = i
// 		}
// 	}

// 	if index == -1 {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Receipe not found"})
// 		return
// 	}
// 	recipes = append(recipes[:index], recipes[index+1:]...)
// 	c.JSON(http.StatusOK, gin.H{"message": "Recipe has been deleted"})
// }

func SearchRecipesHandler(c *gin.Context) {
	tag := c.Query("tag")
	listOfRecipes := make([]Recipe, 0)
	for i := 0; i < len(recipes); i++ {
		found := false
		for _, t := range recipes[i].Tags {
			if strings.EqualFold(t, tag) {
				found = true
			}
		}
		if found {
			listOfRecipes = append(listOfRecipes, recipes[i])
		}
	}
	c.JSON(http.StatusOK, listOfRecipes)
}

func main() {
	router := gin.Default()
	router.POST("/recipes", NewRecipeHandler)
	router.GET("/recipes", ListRecipeHandler)
	router.PUT("/recipes/:id", UpdateRecipeHandler)
	// router.DELETE("/recipes/:id", DeleteRecipeHandler)
	router.GET("/recipes/search", SearchRecipesHandler)
	router.Run()
}
