package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Upgrader configures the upgrade from HTTP to Websocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// MongoDB client and collection
var client *mongo.Client
var collection *mongo.Collection

// Profile struct to hold profile data used during Signup and login
type Profile struct {
	ProfileName     string `json:"profileName"`
	ProfilePassword string `json:"profilePassword"`
	TotalTrophies   uint16 `json:"totalTrophies"`
	Status          string `json:"status,omitempty"`
	Country         string `json:"country,omitempty"`
	ProfileImageURL string `json:"profileImageURL,omitempty"`
}

// Achievements struct to hold only the achievements completed so far by the user
type Achievements struct {
	// Achievements field which stores array of elements where each element can be of any type
	Achievements []any `json:"achievements"`
}

// Response struct to hold the JSON response message used during sending json message during login
type Response struct {
	Message         string `json:"message,omitempty"`
	ProfileName     string `json:"profileName,omitempty"`
	ProfilePassword string `json:"profilePassword,omitempty"`
	// You should use Omitempty because this will skip this field if not set
	// If we use omitempty that field will be not included in the final json if the field is empty
	TotalTrophies   uint16 `json:"totalTrophies"`
	Status          string `json:"status,omitempty"`
	Country         string `json:"country,omitempty"`
	ProfileImageURL string `json:"profileImageURL,omitempty"`
}

// Structure of message from client triggered when joinging and leaving the websocket server used when clearing the map entries
type Message struct {
	Action       string `json:"action"`
	RoomId       string `json:"roomId"`
	ProfileName  string `json:"profileName"`
	PlayerPoints uint16 `json:"playerPoints,omitempty"`
	// This field will hold the total trophies a user got when he enters the server
	TotalTrophies uint16 `json:"totalTrophies"`
}

// Defining a struct to hold both the websocket connection and its profile name
type PlayerInfo struct {
	Connection   *websocket.Conn
	ProfileName  string
	PlayerPoints int
	// Will store the total trophies scored so far
	TotalTrophies uint16 `json:"totalTrophies"`
}

// A map to store room id as key and array of 2 strings as profile name of two players
var playersInQueue = make(map[string][]PlayerInfo)

// Connect to MongoDB and set the quiz database and profile collection
func connectMongoDB() {
	// Load environment variables from .env file
	envErr := godotenv.Load()
	if envErr != nil {
		log.Fatal("Error loading .env file")
	}

	// Get the MongoDB URI from the environment
	uri := os.Getenv("MONGO_CONNECTION_URI")
	if uri == "" {
		log.Fatal("MONGO_URI not set in .env file")
	}
	// Use the SetServerAPIOptions() method to set the Stable API version to 1
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)
	// Create a new client and connect to the server
	var err error
	client, err = mongo.Connect(context.TODO(), opts)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	// Connect to the quiz database and the profile collection
	collection = client.Database("quiz").Collection("profile")

	// Confirm the connection
	fmt.Println("Connected to MongoDB, database: quiz, collection: profile")
}

// Checking if the profile name already exists
func checkProfileNameExists(w http.ResponseWriter, r *http.Request) {
	// Get the profile-name query from the query parameter
	profileName := r.URL.Query().Get("profile-name")

	isRetreiveProfileImage := r.URL.Query().Get("get-profile-image")

	if profileName == "" {
		http.Error(w, "Missing profile-name parameter", http.StatusBadRequest)
		return
	}

	// Searching for a document with the given profileName in the MongoDB collection
	var existingProfile Profile
	err := collection.FindOne(context.TODO(), bson.M{"profileName": profileName}).Decode(&existingProfile)

	// If no document is found, the profile name is not taken, so send an OK response
	if err == mongo.ErrNoDocuments {
		json.NewEncoder(w).Encode(Response{Message: "notTaken"})
		return
	} else if err != nil {
		// Other errors (e.g., database connection issues)
		http.Error(w, "Error checking username", http.StatusInternalServerError)
		return
	}
	// To store profile image url
	var profileImageURL string
	// If we want to get the profile image url from cloudinary
	if isRetreiveProfileImage == "true" {
		cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
		cloudAPIKey := os.Getenv("CLOUDINARY_API_KEY")
		cloudSecret := os.Getenv("CLOUDINARY_API_SECRET")
		cld, err := cloudinary.NewFromParams(cloudName, cloudAPIKey, cloudSecret)
		if err != nil {
			http.Error(w, "Failed to initialize Cloudinary", http.StatusInternalServerError)
			return
		}

		// Context for the upload process
		ctx := context.Background()

		// Uploading the image
		res, err := cld.Admin.Asset(ctx, admin.AssetParams{PublicID: fmt.Sprintf("Duel of Wits/%s", profileName)})
		if err != nil {
			fmt.Print("Error when retreiving profile image information")
			return
		}
		profileImageURL = res.SecureURL
		fmt.Printf("profile image url is %v", profileImageURL)
	}
	// If no error, it means the username exists, so send a conflict response with profile details
	w.Header().Set("Content-Type", "application/json")
	// Send the profile name and password in the JSON response
	response := Response{
		Message:         "taken",
		ProfileName:     existingProfile.ProfileName,
		ProfilePassword: existingProfile.ProfilePassword,
		TotalTrophies:   existingProfile.TotalTrophies,
		Status:          existingProfile.Status,
		Country:         existingProfile.Country,
		ProfileImageURL: profileImageURL,
	}
	//fmt.Print(response)
	json.NewEncoder(w).Encode(response)
}

// Save profile data to MongoDB
func createProfile(w http.ResponseWriter, r *http.Request) {
	// Decode the JSON request body into a Profile struct
	var profile Profile
	err := json.NewDecoder(r.Body).Decode(&profile)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	// Insert profile data into MongoDB
	_, err = collection.InsertOne(context.TODO(), bson.M{
		"profileName":     profile.ProfileName,
		"profilePassword": profile.ProfilePassword,
		"totalTrophies":   0,
		"status":          "",
		"country":         "",
		// Initialize achievements as an empty array
		"achievements": []interface{}{},
	})
	if err != nil {
		http.Error(w, "Failed to save profile", http.StatusInternalServerError)
		return
	}

	// Success response
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "Profile saved successfully")
}

// For updating profile data
func updateProfileData(w http.ResponseWriter, r *http.Request) {
	// profile variable holds structure data structure
	var profile Profile
	// Decoding request body
	err := json.NewDecoder(r.Body).Decode(&profile)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	// Check if the profileName is provided
	if profile.ProfileName == "" {
		http.Error(w, "profile name is required", http.StatusBadRequest)
		return
	}
	filter := bson.M{"profileName": profile.ProfileName} // Find by profileName
	update := bson.M{
		"$set": bson.M{
			"profileName": profile.ProfileName,
			"status":      profile.Status,
			"country":     profile.Country,
		},
	}
	_, err = collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		http.Error(w, "Failed to update profile data", http.StatusInternalServerError)
		return
	}
	// Success response
	w.WriteHeader(http.StatusCreated)
	// Writing response to the response writer
	w.Write([]byte(`{"message":"Profile updated successfully"}`))
}

// For getting leaderboard data
func getLeaderboardData(w http.ResponseWriter, r *http.Request) {
	opts := options.Find().SetSort(bson.D{{"totalTrophies", -1}}).SetLimit(10)
	// Retrieves documents
	cursor, err := collection.Find(context.TODO(), bson.M{}, opts)
	if err != nil {
		http.Error(w, "Failed to retreive leaderboard data", http.StatusInternalServerError)
		fmt.Print("error occured")
		return
	}
	var leaderboard []Profile
	// All iterates the cursor and decodes each document into results , the results parameter must be a pointer to a slice
	if err := cursor.All(context.TODO(), &leaderboard); err != nil {
		http.Error(w, "Failed to decode leaderboard data", http.StatusInternalServerError)
		return
	}
	// All good the cursor is iterated and each document is decoded into results (from bson - binary JSON format to go struct data structure)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(leaderboard); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// Getting achievements data

func getAchievementData(w http.ResponseWriter, r *http.Request) {
	profileName := r.URL.Query().Get("profileName")
	if profileName == "" {
		http.Error(w, "Missing profile name parameter", http.StatusBadRequest)
	}
	// achievements is a variable to store the document returned from database
	var achievements Achievements
	err := collection.FindOne(context.TODO(), bson.M{"profileName": profileName}).Decode(&achievements)
	if err != nil {
		http.Error(w, "Failed to find achievement data", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// Serializing the achievements data into JSON format and writing it to the ResponseWriter
	// First NewEncoder sets up and JSON encoder and the destination where the JSON need to go (Response w in this case)  nd write it to the writer w and Encode performs the actual serialization of the achievements structure
	if err := json.NewEncoder(w).Encode(achievements); err != nil {
		http.Error(w, "Failed to encode achievements", http.StatusInternalServerError)
		return
	}

}

// Handling websocket connections
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()
	// Log and echo the message back to the client
	log.Printf("Client connected!")
	// Infinite loop to keep reading messages and writing messages back
	for {
		_, message, err := ws.ReadMessage()
		// Client disconnects
		if err != nil {
			log.Println("Client disconnected", err)
			break
		}
		var jsonMessage Message
		if err := json.Unmarshal(message, &jsonMessage); err != nil {
			log.Println("Error parsing message:", err)
		}
		// Successfully parsed the json message from client (can be joining or leaving)
		userPlayerName := jsonMessage.ProfileName
		fmt.Printf("%s user name", userPlayerName)
		userAction := jsonMessage.Action
		playerTotalTrophies := jsonMessage.TotalTrophies

		var matchFound bool = false

		// Incase the action is join check the queue for any empty room if not create one and add the user to the room
		if userAction == "connect" {
			// Traverse the queue and find a match for the user
			for roomId, players := range playersInQueue {
				// We found a match for the user
				if len(players) == 1 {

					//* We found a opponent now we have to check if the opponent is equally skilled
					//trophyDifference := math.Abs(float64(playerTotalTrophies) - float64(playersInQueue[roomId][0].TotalTrophies))
					// We found a perfect match

					playersInQueue[roomId] = append(playersInQueue[roomId], PlayerInfo{Connection: ws, ProfileName: userPlayerName, TotalTrophies: playerTotalTrophies})
					matchFound = true

					// Send confirmation to two users that a match is found
					for i := 0; i < 2; i++ {
						if i == 0 {
							// Send a message to first player in the room that match has been found
							confirmationMessage := []byte(fmt.Sprintf(`{"message":"Match found!","opponent":"%s","roomId":"%s"}`, playersInQueue[roomId][1].ProfileName, roomId))
							if err := playersInQueue[roomId][i].Connection.WriteMessage(websocket.TextMessage, confirmationMessage); err != nil {
								log.Printf("Error sending match confirmation message to user\n")
							}
						} else {
							// Send a message to second player in the room that match has been found
							confirmationMessage := []byte(fmt.Sprintf(`{"message":"Match found!","opponent":"%s","roomId":"%s"}`, playersInQueue[roomId][0].ProfileName, roomId))
							if err := playersInQueue[roomId][i].Connection.WriteMessage(websocket.TextMessage, confirmationMessage); err != nil {
								log.Printf("Error sending match confirmation message to user\n")
							}
						}

					}

				}
			}
			// We didnt found a match all room is filled so create a new room
			if !matchFound {
				// Create a Unique roomid for the user because we cannot find any free room for the user
				roomId, err := generateRandomHex(16)
				if err != nil {
					log.Fatalf("Error generating random value: %v", err)
					return
				}
				playersInQueue[roomId] = append(playersInQueue[roomId], PlayerInfo{Connection: ws, ProfileName: userPlayerName, TotalTrophies: playerTotalTrophies})
			}
		} else if userAction == "disconnect" {
			// When users rage quits or when the game is finished in both cases completely delete the room and pick your winner
			for roomId, playerInfo := range playersInQueue {
				// Make index as _ and player contains array of struct containing user name and websocket address
				for _, player := range playerInfo {
					// Remove the room from the queuing server
					//! I guess the pointer in memory if freed
					if player.ProfileName == userPlayerName {
						delete(playersInQueue, roomId)
					}
				}
			}
		} else if userAction == "player_completed" {
			// The player finishes a question
			roomId := jsonMessage.RoomId
			profileName := jsonMessage.ProfileName
			playerPoints := jsonMessage.PlayerPoints
			fmt.Printf("room id: %s, profile name %s, player points :%d", roomId, profileName, playerPoints)
			// The first player is the one who send the total points to the server
			if playersInQueue[roomId][0].ProfileName == profileName {
				// We know have the total points scored by player1 so send the data to player2
				confirmationMessage := []byte(fmt.Sprintf(`{"opponent_total_points":"%d"}`, playerPoints))
				if err := playersInQueue[roomId][1].Connection.WriteMessage(websocket.TextMessage, confirmationMessage); err != nil {
					log.Printf("Error sending message to opponent\n")
				}
			} else {
				// We know have the total points scored by player2 so send the data to player1
				confirmationMessage := []byte(fmt.Sprintf(`{"opponent_total_points":"%d"}`, playerPoints))
				if err := playersInQueue[roomId][0].Connection.WriteMessage(websocket.TextMessage, confirmationMessage); err != nil {
					log.Printf("Error sending message to opponent\n")
				}
			}
		} else if userAction == "match_completed" {
			// When the match is completed remove the room from the server
			roomId := jsonMessage.RoomId
			delete(playersInQueue, roomId)
		}
	}
}
func updateProfileImage(w http.ResponseWriter, r *http.Request) {
	// Parsing the multipart form (2mb max size)
	err := r.ParseMultipartForm(2 << 20)
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}
	// Retreiving the file
	file, _, err := r.FormFile("profileImage")
	// Retreiving the profile name
	profileName := r.FormValue("profileName")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	// Releases file handle where file handle will consume system resource (file descriptors - used to read,write or manage file without directly manipulating the underlying data structures in the OS)
	defer file.Close()
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	cloudAPIKey := os.Getenv("CLOUDINARY_API_KEY")
	cloudSecret := os.Getenv("CLOUDINARY_API_SECRET")
	// Initializing cloudinary instance
	cld, err := cloudinary.NewFromParams(cloudName, cloudAPIKey, cloudSecret)
	if err != nil {
		fmt.Print("Failed to initialize cloudinary")
		return
	}
	// Context for the upload process
	ctx := context.Background()

	// Uploading the image
	if _, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:   "Duel of Wits",
		PublicID: profileName,
	}); err != nil {
		fmt.Print("Failed to upload image")
		return
	}
	fmt.Printf("Profile image uploaded successfully!\n")
}

// For creating random hex values using crypto module this is the room
func generateRandomHex(length int) (string, error) {
	// Calculate the number of bytes needed
	bytes := length / 2
	if length%2 != 0 {
		bytes++
	}
	// Create a byte slice to hold the random bytes
	randomBytes := make([]byte, bytes)
	// Read random bytes from the crypto/rand source
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	// Convert the bytes to a hexadecimal string
	return hex.EncodeToString(randomBytes)[:length], nil
}
func main() {

	// Connect to MongoDB
	connectMongoDB()
	mux := http.NewServeMux()
	// Configure CORS to allow requests from your frontend
	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	})
	// Setting up CORS
	handler := cors.Handler(mux)
	// Websocket connection
	mux.HandleFunc("/ws", handleConnections)
	// Setting HTTP endpoint for saving profile data
	mux.HandleFunc("/create-profile", createProfile)
	// Setting HTTP endpoint for checking profile name (used to check if the username exists or not while creating account and when during login auth)
	mux.HandleFunc("/check-profile", checkProfileNameExists)
	// Updating profile data
	mux.HandleFunc("/update-profile-data", updateProfileData)
	// Getting leaderboard data
	mux.HandleFunc("/leaderboard-data", getLeaderboardData)
	// For getting achievement data of a user
	mux.HandleFunc("/get-achievement-data", getAchievementData)
	// Run this function to store profile image in cloudinary
	mux.HandleFunc("/update-profile-image", updateProfileImage)

	// Start the server on port 5000
	fmt.Println("Websocket server started on port 5000")
	err := http.ListenAndServe(":5000", handler)
	if err != nil {
		log.Fatal("ListenAndServe error:", err)
	}
}
