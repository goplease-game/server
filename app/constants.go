package app

import "golang.org/x/crypto/bcrypt"

// DefaultBCryptCost defines the default complexity (cost factor) used when hashing
// passwords with the bcrypt algorithm.
const DefaultBCryptCost = bcrypt.DefaultCost
