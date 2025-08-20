package main

import (
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Test the password hashes we're using in the migration
	testCases := []struct {
		password string
		hash     string
		desc     string
	}{
		{
			password: "admin123",
			hash:     "$2a$10$N9qo8uLOickgx2ZMRZoMye.IjPeOXe.2p5l/q/FQcre8HdkL6Q262",
			desc:     "Admin user password",
		},
		{
			password: "demo123",
			hash:     "$2a$10$8K1p/a0dhrxiH8Tf4di1HuP4lxvlmOyqjLxYiMyIlSaw1uYwy55jG",
			desc:     "Demo user password",
		},
	}

	fmt.Println("Verifying password hashes...")
	fmt.Println("====================================================")

	for _, tc := range testCases {
		fmt.Printf("\nTesting %s:\n", tc.desc)
		fmt.Printf("Password: %s\n", tc.password)
		fmt.Printf("Hash: %s\n", tc.hash)

		err := bcrypt.CompareHashAndPassword([]byte(tc.hash), []byte(tc.password))
		if err != nil {
			fmt.Printf("❌ FAILED: %v\n", err)
			log.Printf("Hash verification failed for %s: %v", tc.desc, err)
		} else {
			fmt.Printf("✅ SUCCESS: Password matches hash\n")
		}
	}

	fmt.Println("\n====================================================")
	fmt.Println("Password verification complete!")
}
