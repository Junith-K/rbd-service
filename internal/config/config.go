package config

import (
	"context"
	"log"
	"os"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

var (
	FirebaseApp    *firebase.App
	FirestoreClient *firestore.Client
	AuthClient     *auth.Client
)

// InitFirebase initializes Firebase Admin SDK
func InitFirebase() error {
	ctx := context.Background()

	credentialsPath := os.Getenv("FIREBASE_CREDENTIALS_PATH")
	if credentialsPath == "" {
		credentialsPath = "./serviceAccountKey.json"
	}

	// Check if credentials file exists
	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		log.Printf("⚠️  Firebase credentials not found at %s", credentialsPath)
		log.Println("📝 Please download your Firebase service account key and place it at the specified path")
		return err
	}

	opt := option.WithCredentialsFile(credentialsPath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Printf("Error initializing Firebase app: %v", err)
		return err
	}

	FirebaseApp = app
	log.Println("✅ Firebase app initialized")

	// Initialize Firestore client
	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		log.Printf("Error initializing Firestore: %v", err)
		return err
	}
	FirestoreClient = firestoreClient
	log.Println("✅ Firestore client initialized")

	// Initialize Auth client
	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Printf("Error initializing Auth: %v", err)
		return err
	}
	AuthClient = authClient
	log.Println("✅ Firebase Auth client initialized")

	return nil
}

// CloseFirebase closes Firebase connections
func CloseFirebase() {
	if FirestoreClient != nil {
		FirestoreClient.Close()
		log.Println("🔌 Firestore connection closed")
	}
}
