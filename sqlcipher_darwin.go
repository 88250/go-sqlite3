// Copyright (C) 2026 BitBoxSwiss.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build sqlcipher && !libsqlcipher && darwin
// +build sqlcipher,!libsqlcipher,darwin

package sqlite3

/*
#cgo LDFLAGS: -framework CoreFoundation -framework Security
#cgo CFLAGS: -DSQLCIPHER_CRYPTO_CC
*/
import "C"
